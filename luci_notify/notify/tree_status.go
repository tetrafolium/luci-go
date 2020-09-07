// Copyright 2020 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/common/data/stringset"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	"github.com/tetrafolium/luci-go/common/sync/parallel"
	"github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/luci_notify/config"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/router"
)

const botUsername = "luci-notify@appspot.gserviceaccount.com"
const legacyBotUsername = "buildbot@chromium.org"

type treeStatus struct {
	username  string
	message   string
	key       int64
	status    config.TreeCloserStatus
	timestamp time.Time
}

type treeStatusClient interface {
	getStatus(c context.Context, host string) (*treeStatus, error)
	postStatus(c context.Context, host, message string, prevKey int64) error
}

type httpTreeStatusClient struct {
	getFunc  func(context.Context, string) ([]byte, error)
	postFunc func(context.Context, string) error
}

func (ts *httpTreeStatusClient) getStatus(c context.Context, host string) (*treeStatus, error) {
	respJSON, err := ts.getFunc(c, fmt.Sprintf("https://%s/current?format=json", host))
	if err != nil {
		return nil, err
	}

	var r struct {
		Username        string
		CanCommitFreely bool `json:"can_commit_freely"`
		Key             int64
		Date            string
		Message         string
	}
	if err = json.Unmarshal(respJSON, &r); err != nil {
		return nil, errors.Annotate(err, "failed to unmarshal JSON").Err()
	}

	var status config.TreeCloserStatus = config.Closed
	if r.CanCommitFreely {
		status = config.Open
	}

	// Similar to RFC3339, but not quite the same. No time zone is specified,
	// so this will default to UTC, which is correct here.
	const dateFormat = "2006-01-02 15:04:05.999999"
	t, err := time.Parse(dateFormat, r.Date)
	if err != nil {
		return nil, errors.Annotate(err, "failed to parse date from tree status").Err()
	}

	return &treeStatus{
		username:  r.Username,
		message:   r.Message,
		key:       r.Key,
		status:    status,
		timestamp: t,
	}, nil
}

func (ts *httpTreeStatusClient) postStatus(c context.Context, host, message string, prevKey int64) error {
	logging.Infof(c, "Updating status for %s: %q", host, message)

	q := url.Values{}
	q.Add("message", message)
	q.Add("last_status_key", strconv.FormatInt(prevKey, 10))
	u := url.URL{
		Host:     host,
		Scheme:   "https",
		Path:     "/",
		RawQuery: q.Encode(),
	}

	return ts.postFunc(c, u.String())
}

func getHttp(c context.Context, url string) ([]byte, error) {
	response, err := makeHttpRequest(c, url, "GET")
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Annotate(err, "failed to read response body from %q", url).Err()
	}

	return bytes, nil
}

func postHttp(c context.Context, url string) error {
	response, err := makeHttpRequest(c, url, "POST")
	if err != nil {
		return err
	}

	response.Body.Close()

	// If the operation succeeded, the status app will apply the update, and
	// then redirect back to the main page. Let's also check for a 200, as this
	// is a reasonable response and we don't want to depend too heavily on
	// particular implementation details.
	if response.StatusCode == http.StatusFound || response.StatusCode == http.StatusOK {
		return nil
	}
	return fmt.Errorf("POST to %q returned unexpected status code %d", url, response.StatusCode)
}

func makeHttpRequest(c context.Context, url, method string) (*http.Response, error) {
	transport, err := auth.GetRPCTransport(c, auth.AsSelf)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(c)

	response, err := (&http.Client{Transport: transport}).Do(req)
	if err != nil {
		return nil, errors.Annotate(err, "%s request to %q failed", method, url).Err()
	}

	return response, nil
}

// UpdateTreeStatus is the HTTP handler triggered by cron when it's time to
// check tree closers and update tree status if necessary.
func UpdateTreeStatus(c *router.Context) {
	ctx, w := c.Context, c.Writer
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	if err := updateTrees(ctx, &httpTreeStatusClient{getHttp, postHttp}); err != nil {
		logging.WithError(err).Errorf(ctx, "error while updating tree status")
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// updateTrees fetches all TreeClosers from datastore, uses this to determine if
// any trees should be opened or closed, and makes the necessary updates.
func updateTrees(c context.Context, ts treeStatusClient) error {
	// The goal here is, for every project, to atomically fetch the config
	// for that project along with all TreeClosers within it. So if the
	// project config and the set of TreeClosers are updated at the same
	// time, we should always see either both updates, or neither. Also, we
	// want to do it without XG transactions.
	//
	// First we fetch keys for all the projects. Second, for every project,
	// we fetch the full config and all TreeClosers in a transaction. Since
	// these two steps aren't within a transaction, it's possible that
	// changes have occurred in between. But all cases are dealt with:
	//
	// * Updates to project config or TreeClosers aren't a problem since we
	//   only fetch them in the second step anyway.
	// * Deletions of projects are fine, since if we don't find them in the
	//   second fetch we just ignore that project and carry on.
	// * New projects are ignored, and picked up the next time we run.
	q := datastore.NewQuery("Project").KeysOnly(true)
	var projects []*config.Project
	if err := datastore.GetAll(c, q, &projects); err != nil {
		return errors.Annotate(err, "failed to get project keys").Err()
	}

	// Guards access to both treeClosers and closingEnabledProjects.
	mu := sync.Mutex{}
	var treeClosers []*config.TreeCloser
	closingEnabledProjects := stringset.New(0)

	err := parallel.WorkPool(32, func(ch chan<- func() error) {
		for _, project := range projects {
			project := project
			ch <- func() error {
				return datastore.RunInTransaction(c, func(c context.Context) error {
					switch err := datastore.Get(c, project); {
					// The project was deleted since the previous time we fetched it just above.
					// In this case, just move on, since the project is no more.
					case err == datastore.ErrNoSuchEntity:
						return nil
					case err != nil:
						return errors.Annotate(err, "failed to get project").Tag(transient.Tag).Err()
					}

					q := datastore.NewQuery("TreeCloser").Ancestor(datastore.KeyForObj(c, project))
					var treeClosersForProject []*config.TreeCloser
					if err := datastore.GetAll(c, q, &treeClosersForProject); err != nil {
						return errors.Annotate(err, "failed to get tree closers").Tag(transient.Tag).Err()
					}

					mu.Lock()
					defer mu.Unlock()
					treeClosers = append(treeClosers, treeClosersForProject...)
					if project.TreeClosingEnabled {
						closingEnabledProjects.Add(project.Name)
					}

					return nil
				}, nil)
			}
		}
	})
	if err != nil {
		return err
	}

	return parallel.WorkPool(32, func(ch chan<- func() error) {
		for host, treeClosers := range groupTreeClosers(treeClosers) {
			host, treeClosers := host, treeClosers
			ch <- func() error {
				c := logging.SetField(c, "tree-status-host", host)
				return updateHost(c, ts, host, treeClosers, closingEnabledProjects)
			}
		}
	})
}

func groupTreeClosers(treeClosers []*config.TreeCloser) map[string][]*config.TreeCloser {
	byHost := map[string][]*config.TreeCloser{}
	for _, tc := range treeClosers {
		byHost[tc.TreeStatusHost] = append(byHost[tc.TreeStatusHost], tc)
	}

	return byHost
}

func tcProject(tc *config.TreeCloser) string {
	return tc.BuilderKey.Parent().StringID()
}

func updateHost(c context.Context, ts treeStatusClient, host string, treeClosers []*config.TreeCloser, closingEnabledProjects stringset.Set) error {
	treeStatus, err := ts.getStatus(c, host)
	if err != nil {
		return err
	}

	if treeStatus.status == config.Closed && treeStatus.username != botUsername && treeStatus.username != legacyBotUsername {
		// Don't do anything if the tree was manually closed.
		logging.Debugf(c, "Tree is closed and last update was from non-bot user %s; not doing anything", treeStatus.username)
		return nil
	}

	anyEnabled := false
	for _, tc := range treeClosers {
		if closingEnabledProjects.Has(tcProject(tc)) {
			anyEnabled = true
			break
		}
	}

	anyFailingBuild := false
	anyNewBuild := false
	var oldestClosed *config.TreeCloser
	for _, tc := range treeClosers {
		// If any TreeClosers are from projects with tree closing enabled,
		// ignore any TreeClosers *not* from such projects. In general we don't
		// expect different projects to close the same tree, so we're okay with
		// not seeing dry run logging for these TreeClosers in this rare case.
		if anyEnabled && !closingEnabledProjects.Has(tcProject(tc)) {
			continue
		}

		// For opening the tree, we need to make sure *all* builders are
		// passing, not just those that have had new builds. Otherwise we'll
		// open the tree after any new green build, even if the builder that
		// caused us to close it is still failing.
		if tc.Status == config.Closed {
			logging.Debugf(c, "Found failing builder with message: %s", tc.Message)
			anyFailingBuild = true
		}

		// Only pay attention to failing builds from after the last update to
		// the tree. Otherwise we'll close the tree even after people manually
		// open it.
		if tc.Timestamp.Before(treeStatus.timestamp) {
			continue
		}

		anyNewBuild = true

		if tc.Status == config.Closed && (oldestClosed == nil || tc.Timestamp.Before(oldestClosed.Timestamp)) {
			logging.Debugf(c, "Updating oldest failing builder")
			oldestClosed = tc
		}
	}

	var newStatus config.TreeCloserStatus
	if !anyNewBuild {
		// Don't do anything if all the builds are older than the last update
		// to the tree - nothing has changed, so there's no reason to take any
		// action.
		logging.Debugf(c, "No builds newer than last tree update (%s); not doing anything",
			treeStatus.timestamp.Format(time.RFC1123Z))
		return nil
	}
	if !anyFailingBuild {
		// We can open the tree, as no builders are failing, including builders
		// that haven't run since the last update to the tree.
		logging.Debugf(c, "No failing builders; new status is Open")
		newStatus = config.Open
	} else if oldestClosed != nil {
		// We can close the tree, as at least one builder has failed since the
		// last update to the tree.
		logging.Debugf(c, "At least one failing builder; new status is Closed")
		newStatus = config.Closed
	} else {
		// Some builders are failing, but they were already failing before the
		// last update. Don't do anything, so as not to close the tree after a
		// sheriff has manually opened it.
		logging.Debugf(c, "At least one failing builder, but there's a more recent update; not doing anything")
		return nil
	}

	if treeStatus.status == newStatus {
		// Don't do anything if the current status is already correct.
		logging.Debugf(c, "Current status is already correct; not doing anything")
		return nil
	}

	var message string
	if newStatus == config.Open {
		message = fmt.Sprintf("Tree is open (Automatic: %s)", randomMessage(c))
	} else {
		message = fmt.Sprintf("Tree is closed (Automatic: %s)", oldestClosed.Message)
	}

	if anyEnabled {
		return ts.postStatus(c, host, message, treeStatus.key)
	}
	logging.Infof(c, "Would update status for %s to %q", host, message)
	return nil
}

// NOTE: If you want to add a new message, do so in Gatekeeper, not here. The
// full list will be copied over before Gatekeeper is deleted.
var messages = []string{
	"(｡>﹏<｡)",
	"☃",
	"☀ Tree is open ☀",
	"٩◔̯◔۶",
	"☺",
	"(´・ω・`)",
	"(΄◞ิ౪◟ิ‵ )",
	"(╹◡╹)",
	"♩‿♩",
	"(/･ω･)/",
	" ʅ(◔౪◔ ) ʃ",
	"ᕙ(`▿´)ᕗ",
	"ヽ(^o^)丿",
	"\\(･ω･)/",
	"＼(^o^)／",
	"ｷﾀ━━━━(ﾟ∀ﾟ)━━━━ｯ!!",
	"ヽ(^。^)ノ",
	"(ﾟдﾟ)",
	"ヽ(´ω`*人*´ω`)ノ",
	" ﾟ+｡:.ﾟヽ(*´∀`)ﾉﾟ.:｡+ﾟ",
	"(゜ー゜＊）ネッ！",
	" ♪d(´▽｀)b♪オールオッケィ♪",
	"(ﾉ≧∀≦)ﾉ・‥…",
	"☆（ゝω・）vｷｬﾋﾟ",
	"ლ(╹◡╹ლ)",
	"ƪ(•̃͡ε•̃͡)∫ʃ",
	"(•_•)",
	"( ་ ⍸ ་ )",
	"(☉౪ ⊙)",
	"˙ ͜ʟ˙",
	"( ఠൠఠ )",
	"☆.｡.:*･ﾟ☆.｡.:*･ﾟ☆祝☆ﾟ･*:.｡.☆ﾟ･*:.｡.☆",
	"༼ꉺɷꉺ༽",
	"◉_◉",
	"ϵ( ‘Θ’ )϶",
	"ヾ(⌐■_■)ノ♪",
	"(◡‿◡✿)",
	"★.:ﾟ+｡☆ (●´v｀○)bｫﾒﾃﾞﾄd(○´v｀●)☆.:ﾟ+｡★",
	"(☆.☆)",
	"ｵﾒﾃﾞﾄｰ♪c(*ﾟｰ^)ﾉ*･'ﾟ☆｡.:*:･'☆'･:*:.",
	"☆.。.:*・°☆.。.:*・°☆",
	"ʕ •ᴥ•ʔ",
	"☼.☼",
	"⊂(・(ェ)・)⊃",
	"(ﾉ≧∇≦)ﾉ ﾐ ┸━┸",
	"¯\\_(ツ)_/¯",
	"UwU",
	"Paç fat!",
	"Sretno",
	"Hodně štěstí!",
	"Held og lykke!",
	"Veel geluk!",
	"Edu!",
	"lykkyä tykö",
	"Viel Glück!",
	"Καλή τύχη!",
	"Sok szerencsét kivánok!",
	"Gangi þér vel!",
	"Go n-éirí an t-ádh leat!",
	"Buona fortuna!",
	"Laimīgs gadījums!",
	"Sėkmės!",
	"Vill Gléck!",
	"Со среќа!",
	"Powodzenia!",
	"Boa sorte!",
	"Noroc!",
	"Срећно",
	"Veľa šťastia!",
	"Lycka till!",
	"Bona sort!",
	"Zorte on!",
	"Góða eydnu",
	"¡Boa fortuna!",
	"Bona fortuna!",
	"Xewqat sbieħ",
	"Aigh vie!",
	"Pob lwc!",
	" موفق باشيد",
	"İyi şanslar!",
	"Bonŝancon!",
	"祝你好运！",
	"祝你好運！",
	"頑張って！",
	"សំណាងល្អ ",
	"행운을 빌어요",
	"शुभ कामना ",
	"โชคดี!",
	"Chúc may mắn!",
	"بالتوفيق!",
	"Sterkte!",
	"Ke o lakaletsa mohlohonolo",
	"Uve nemhanza yakanaka",
	"Kila la kheri!",
	"Amathamsanqa",
	"Ngikufisela iwela!",
	"Bonne chance!",
	"¡Buena suerte!",
	"Good luck!",
	"Semoga Beruntung!",
	"Selamat Maju Jaya!",
	"Ia manuia",
	"Suwertehin ka sana",
	"Удачи!",
	"Հաջողությո'ւն",
	"Іске сәт",
	"Амжилт хүсье",
	"удачі!",
	"Da legst di nieda!",
	"Gell, da schaugst?",
	"Ois Guade",
	"शुभ कामना!",
	"நல் வாழ்த்துக்கள் ",
	"అంతా శుభం కలగాలి! ",
	":')",
	":'D",
	"Tree is open (^O^)",
	"Thượng lộ bình an",
	"Tree is open now (ง '̀͜ '́ )ง",
	"ヽ(^o^)ノ",
}

func randomMessage(c context.Context) string {
	message := messages[mathrand.Intn(c, len(messages))]
	if message[len(message)-1] == ')' {
		return message + " "
	}
	return message
}
