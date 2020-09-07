// Copyright 2018 The LUCI Authors.
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

// Package config implements validation and common manipulation of CQ config
// files.
package config

import (
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	"github.com/tetrafolium/luci-go/common/data/stringset"
	"github.com/tetrafolium/luci-go/config/validation"

	v2 "github.com/tetrafolium/luci-go/cv/api/config/v2"
)

// Config validation rules go here.

func init() {
	addRules(&validation.Rules)
}

func addRules(r *validation.RuleSet) {
	r.Add("regex:projects/[^/]+", "${appid}.cfg", validateProject)
}

// validateProject validates a project-level CQ config.
//
// Validation result is returned via validation ctx, while error returned
// directly implies only a bug in this code.
func validateProject(ctx *validation.Context, configSet, path string, content []byte) error {
	ctx.SetFile(path)
	cfg := v2.Config{}
	if err := proto.UnmarshalText(string(content), &cfg); err != nil {
		ctx.Error(err)
	} else {
		validateProjectConfig(ctx, &cfg)
	}
	return nil
}

func validateProjectConfig(ctx *validation.Context, cfg *v2.Config) {
	if cfg.ProjectScopedAccount != v2.Toggle_UNSET {
		ctx.Errorf("project_scoped_account for just CQ isn't supported. " +
			"Use project-wide config for all LUCI services in luci-config/projects.cfg")
	}
	if cfg.DrainingStartTime != "" {
		if _, err := time.Parse(time.RFC3339, cfg.DrainingStartTime); err != nil {
			ctx.Errorf("failed to parse draining_start_time %q as RFC3339 format: %s", cfg.DrainingStartTime, err)
		} else {
			// TODO(crbug/1102635): remove this check as only Python CQ code can't
			// handle strings without "Z".
			if !strings.HasSuffix(cfg.DrainingStartTime, "Z") {
				ctx.Errorf("draining_start_time %q should be in UTC timezone and end with 'Z'"+
					", for example '2020-07-06T21:00:30Z'", cfg.DrainingStartTime)
			}
		}
	}
	if cfg.CqStatusHost != "" {
		switch u, err := url.Parse("https://" + cfg.CqStatusHost); {
		case err != nil:
			ctx.Errorf("failed to parse cq_status_host %q: %s", cfg.CqStatusHost, err)
		case u.Host != cfg.CqStatusHost:
			ctx.Errorf("cq_status_host %q should be just a host %q", cfg.CqStatusHost, u.Host)
		}
	}
	if cfg.SubmitOptions != nil {
		ctx.Enter("submit_options")
		if cfg.SubmitOptions.MaxBurst < 0 {
			ctx.Errorf("max_burst must be >= 0")
		}
		if cfg.SubmitOptions.BurstDelay != nil {
			switch d, err := ptypes.Duration(cfg.SubmitOptions.BurstDelay); {
			case err != nil:
				ctx.Errorf("invalid burst_delay: %s", err)
			case d.Seconds() < 0.0:
				ctx.Errorf("burst_delay must be positive or 0")
			}
		}
		ctx.Exit()
	}
	if len(cfg.ConfigGroups) == 0 {
		ctx.Errorf("at least 1 config_group is required")
		return
	}

	fallbackGroupIdx := -1
	for i, g := range cfg.ConfigGroups {
		ctx.Enter("config_group #%d", i+1)
		validateConfigGroup(ctx, g)
		switch {
		case g.Fallback == v2.Toggle_YES && fallbackGroupIdx == -1:
			fallbackGroupIdx = i
		case g.Fallback == v2.Toggle_YES:
			ctx.Errorf("At most 1 config_group with fallback=YES allowed "+
				"(already declared in config_group #%d", fallbackGroupIdx+1)
		}
		ctx.Exit()
	}
	bestEffortDisjointGroups(ctx, cfg)
}

type refKey struct {
	url     string
	project string
	refStr  string
}

// bestEffortDisjointGroups errors out on easy to spot overlaps between
// configGroups.
//
// It is non-trivial if it all possible to ensure that regexp across
// config_groups don't overlap. But, we can catch typical copy-pasta mistakes
// early on by checking for equality of regexps.
func bestEffortDisjointGroups(ctx *validation.Context, cfg *v2.Config) {
	defaultRefRegexps := []string{"refs/heads/master"}
	// Multimap gerrit URL => project => refRegexp => config group index.
	seen := map[refKey]int{}

	for grIdx, gr := range cfg.ConfigGroups {
		if gr.Fallback == v2.Toggle_YES {
			continue
		}
		for gIdx, g := range gr.Gerrit {
			for pIdx, p := range g.Projects {
				refRegexps := p.RefRegexp
				if len(p.RefRegexp) == 0 {
					refRegexps = defaultRefRegexps
				}
				for rIdx, refRegexp := range refRegexps {
					if seenIdx, aliasing := seen[refKey{g.Url, p.Name, refRegexp}]; !aliasing {
						seen[refKey{g.Url, p.Name, refRegexp}] = grIdx
					} else if seenIdx != grIdx {
						// NOTE: we have already emitted error on duplicate gerrit URL,
						// project name, or ref_regexp within their own respective
						// container, so only error here is cases when these span multiple
						// config_groups.
						ctx.Enter("config_group #%d", grIdx+1)
						ctx.Enter("gerrit #%d", gIdx+1)
						ctx.Enter("project #%d", pIdx+1)
						ctx.Enter("ref_regexp #%d", rIdx+1)
						ctx.Errorf("aliases config_group #%d", seenIdx+1)
						ctx.Exit()
						ctx.Exit()
						ctx.Exit()
						ctx.Exit()
					}
				}
			}
		}
	}

	// Second type of heuristics: match individual refs which are typically in
	// use, and check if they match against >1 configs.
	plainRefs := []string{
		"refs/heads/master",
		"refs/heads/branch",
		"refs/heads/infra/config",
		"refs/branch-heads/1234",
	}
	// Multimap gerrit url => project => plainRef => list of config_group indexes
	// matching this plainRef.
	matchedBy := map[refKey][]int{}
	for ref, seenIdx := range seen {
		// Only check valid regexps here.
		if re, err := regexp.Compile("^" + ref.refStr + "$"); err == nil {
			for _, plainRef := range plainRefs {
				if re.MatchString(plainRef) {
					plainRefKey := refKey{ref.url, ref.project, plainRef}
					matchedBy[plainRefKey] = append(matchedBy[plainRefKey], seenIdx)
				}
			}
		}
	}
	for ref, matchedIdxs := range matchedBy {
		if len(matchedIdxs) > 1 {
			sort.Slice(matchedIdxs, func(i, j int) bool { return matchedIdxs[i] < matchedIdxs[j] })
			ctx.Errorf("Overlapping config_groups not allowed. Gerrit %q project %q ref %q matches config_groups %v",
				ref.url, ref.project, ref.refStr, matchedIdxs)
		}
	}
}

func validateConfigGroup(ctx *validation.Context, group *v2.ConfigGroup) {
	re, _ := regexp.Compile("^[a-zA-Z][a-zA-Z0-9_-]*$")
	switch {
	case group.Name == "":
		// TODO(crbug/1063508): make this an error.
		ctx.Warningf("please, specify `name` for monitoring and analytics")
	case !re.MatchString(group.Name):
		// TODO(crbug/1063508): make this an error.
		ctx.Warningf("`name` must match '^[a-zA-Z][a-zA-Z0-9 _.-]*$': %q", group.Name)
	}

	if len(group.Gerrit) == 0 {
		ctx.Errorf("at least 1 gerrit is required")
	}
	gerritURLs := stringset.Set{}
	for i, g := range group.Gerrit {
		ctx.Enter("gerrit #%d", i+1)
		validateGerrit(ctx, g)
		if g.Url != "" && !gerritURLs.Add(g.Url) {
			ctx.Errorf("duplicate gerrit url in the same config_group: %q", g.Url)
		}
		ctx.Exit()
	}

	if group.CombineCls != nil {
		ctx.Enter("combine_cls")
		if group.CombineCls.StabilizationDelay == nil {
			ctx.Errorf("stabilization_delay is required to enable cl_grouping")
		} else {
			switch d, err := ptypes.Duration(group.CombineCls.StabilizationDelay); {
			case err != nil:
				ctx.Errorf("invalid stabilization_delay: %s", err)
			case d.Seconds() < 10.0:
				ctx.Errorf("stabilization_delay must be at least 10 seconds")
			}
		}
		if group.GetVerifiers().GetGerritCqAbility().GetAllowSubmitWithOpenDeps() {
			ctx.Errorf("combine_cls can not be used with gerrit_cq_ability.allow_submit_with_open_deps=true.")
		}
		ctx.Exit()
	}

	if group.Verifiers == nil {
		ctx.Errorf("verifiers are required")
	} else {
		ctx.Enter("verifiers")
		validateVerifiers(ctx, group.Verifiers)
		ctx.Exit()
	}
}

func validateGerrit(ctx *validation.Context, g *v2.ConfigGroup_Gerrit) {
	validateGerritURL(ctx, g.Url)
	if len(g.Projects) == 0 {
		ctx.Errorf("at least 1 project is required")
	}
	nameToIndex := make(map[string]int, len(g.Projects))
	for i, p := range g.Projects {
		ctx.Enter("projects #%d", i+1)
		validateGerritProject(ctx, p)
		if p.Name != "" {
			if _, dup := nameToIndex[p.Name]; !dup {
				nameToIndex[p.Name] = i
			} else {
				ctx.Errorf("duplicate project in the same gerrit: %q", p.Name)
			}
		}
		ctx.Exit()
	}
}

func validateGerritURL(ctx *validation.Context, gURL string) {
	if gURL == "" {
		ctx.Errorf("url is required")
		return
	}
	u, err := url.Parse(gURL)
	if err != nil {
		ctx.Errorf("failed to parse url %q: %s", gURL, err)
		return
	}
	if u.Path != "" {
		ctx.Errorf("path component not yet allowed in url (%q specified)", u.Path)
	}
	if u.RawQuery != "" {
		ctx.Errorf("query component not allowed in url (%q specified)", u.RawQuery)
	}
	if u.Fragment != "" {
		ctx.Errorf("fragment component not allowed in url (%q specified)", u.Fragment)
	}
	if u.Scheme != "https" {
		ctx.Errorf("only 'https' scheme supported for now (%q specified)", u.Scheme)
	}
	if !strings.HasSuffix(u.Host, ".googlesource.com") {
		// TODO(tandrii): relax this.
		ctx.Errorf("only *.googlesource.com hosts supported for now (%q specified)", u.Host)
	}
}

func validateGerritProject(ctx *validation.Context, gp *v2.ConfigGroup_Gerrit_Project) {
	if gp.Name == "" {
		ctx.Errorf("name is required")
	} else {
		if strings.HasPrefix(gp.Name, "/") || strings.HasPrefix(gp.Name, "a/") {
			ctx.Errorf("name must not start with '/' or 'a/'")
		}
		if strings.HasSuffix(gp.Name, "/") || strings.HasSuffix(gp.Name, ".git") {
			ctx.Errorf("name must not end with '.git' or '/'")
		}
	}

	regexps := stringset.Set{}
	for i, r := range gp.RefRegexp {
		ctx.Enter("ref_regexp #%d", i+1)
		if _, err := regexp.Compile(r); err != nil {
			ctx.Error(err)
		}
		if !regexps.Add(r) {
			ctx.Errorf("duplicate regexp: %q", r)
		}
		ctx.Exit()
	}
}

func validateVerifiers(ctx *validation.Context, v *v2.Verifiers) {
	if v.Cqlinter != nil {
		ctx.Errorf("cqlinter verifier is not allowed (internal use only)")
	}
	if v.Fake != nil {
		ctx.Errorf("fake verifier is not allowed (internal use only)")
	}
	if v.TreeStatus != nil {
		ctx.Enter("tree_status")
		if v.TreeStatus.Url == "" {
			ctx.Errorf("url is required")
		} else {
			switch u, err := url.Parse(v.TreeStatus.Url); {
			case err != nil:
				ctx.Errorf("failed to parse url %q: %s", v.TreeStatus.Url, err)
			case u.Scheme != "https":
				ctx.Errorf("url scheme must be 'https'")
			}
		}
		ctx.Exit()
	}
	if v.GerritCqAbility == nil {
		ctx.Errorf("gerrit_cq_ability verifier is required")
	} else {
		ctx.Enter("gerrit_cq_ability")
		if len(v.GerritCqAbility.CommitterList) == 0 {
			ctx.Errorf("committer_list is required")
		} else {
			for i, l := range v.GerritCqAbility.CommitterList {
				if l == "" {
					ctx.Enter("committer_list #%d", i+1)
					ctx.Errorf("must not be empty string")
					ctx.Exit()
				}
			}
		}
		for i, l := range v.GerritCqAbility.DryRunAccessList {
			if l == "" {
				ctx.Enter("dry_run_access_list #%d", i+1)
				ctx.Errorf("must not be empty string")
				ctx.Exit()
			}
		}
		ctx.Exit()
	}
	if v.Tryjob != nil {
		ctx.Enter("tryjob")
		validateTryjobVerifier(ctx, v.Tryjob)
		ctx.Exit()
	}
}

func validateTryjobVerifier(ctx *validation.Context, v *v2.Verifiers_Tryjob) {
	if v.RetryConfig != nil {
		ctx.Enter("retry_config")
		validateTryjobRetry(ctx, v.RetryConfig)
		ctx.Exit()
	}

	switch v.CancelStaleTryjobs {
	case v2.Toggle_YES:
		ctx.Errorf("`cancel_stale_tryjobs: YES` matches default CQ behavior now; please remove")
	case v2.Toggle_NO:
		ctx.Errorf("`cancel_stale_tryjobs: NO` is no longer supported, use per-builder `cancel_stale` instead")
	case v2.Toggle_UNSET:
		// OK
	}

	if len(v.Builders) == 0 {
		ctx.Errorf("at least 1 builder required")
		return
	}

	// Validation of builders is done in two passes: local and global.

	visitBuilders := func(cb func(b *v2.Verifiers_Tryjob_Builder)) {
		for i, b := range v.Builders {
			if b.Name != "" {
				ctx.Enter("builder %s", b.Name)
			} else {
				ctx.Enter("builder #%d", i+1)
			}
			cb(b)
			ctx.Exit()
		}
	}

	// Pass 1, local: verify each builder separately.
	// Also, populate data structures for second pass.
	names := stringset.Set{}
	equi := stringset.Set{} // equivalent_to builder names.
	// Subset of builders that can be triggered directly
	// and which can be relied upon to trigger other builders.
	canStartTriggeringTree := make([]string, 0, len(v.Builders))
	triggersMap := map[string][]string{} // who triggers whom.
	// Find config by name.
	cfgByName := make(map[string]*v2.Verifiers_Tryjob_Builder, len(v.Builders))

	visitBuilders(func(b *v2.Verifiers_Tryjob_Builder) {
		validateBuilderName(ctx, b.Name, names)
		cfgByName[b.Name] = b
		if b.TriggeredBy != "" {
			// Don't validate TriggeredBy as builder name, it should just match
			// another main builder name, which will be validated anyway.
			triggersMap[b.TriggeredBy] = append(triggersMap[b.TriggeredBy], b.Name)
			if b.ExperimentPercentage != 0 {
				ctx.Errorf("experiment_percentage is not combinable with triggered_by")
			}
			if b.EquivalentTo != nil {
				ctx.Errorf("equivalent_to is not combinable with triggered_by")
			}
		}
		if b.EquivalentTo != nil {
			validateEquivalentBuilder(ctx, b.EquivalentTo, equi)
			if b.ExperimentPercentage != 0 {
				ctx.Errorf("experiment_percentage is not combinable with equivalent_to")
			}
		}
		if b.ExperimentPercentage != 0 {
			if b.ExperimentPercentage < 0.0 || b.ExperimentPercentage > 100.0 {
				ctx.Errorf("experiment_percentage must between 0 and 100 (%f given)", b.ExperimentPercentage)
			}
			if b.IncludableOnly {
				ctx.Errorf("includable_only is not combinable with experiment_percentage")
			}
		}
		if len(b.LocationRegexp)+len(b.LocationRegexpExclude) > 0 {
			validateLocationRegexp(ctx, "location_regexp", b.LocationRegexp)
			validateLocationRegexp(ctx, "location_regexp_exclude", b.LocationRegexpExclude)
			if b.IncludableOnly {
				ctx.Errorf("includable_only is not combinable with location_regexp[_exclude]")
			}
		}
		if len(b.OwnerWhitelistGroup) > 0 {
			for i, g := range b.OwnerWhitelistGroup {
				if g == "" {
					ctx.Enter("owner_whitelist_group #%d", i+1)
					ctx.Errorf("must not be empty string")
					ctx.Exit()
				}
			}
		}
		if b.ExperimentPercentage == 0 && b.TriggeredBy == "" && b.EquivalentTo == nil {
			canStartTriggeringTree = append(canStartTriggeringTree, b.Name)
		}
	})

	// Between passes, do a depth-first search into triggers-whom DAG starting
	// with only those builders which can be triggered directly by CQ.
	q := canStartTriggeringTree
	canBeTriggered := stringset.NewFromSlice(q...)
	for len(q) > 0 {
		var b string
		q, b = q[:len(q)-1], q[len(q)-1]
		for _, whom := range triggersMap[b] {
			if canBeTriggered.Add(whom) {
				q = append(q, whom)
			} else {
				panic("IMPOSSIBLE: builder |b| starting at |canStartTriggeringTree| " +
					"isn't triggered by anyone, so it can't be equal to |whom|, which had triggered_by.")
			}
		}
	}
	// Corollary: all builders with triggered_by but not in canBeTriggered set
	// are not properly configured, either referring to non-existing builder OR
	// forming a loop.

	// Pass 2, global: verify builder relationships.
	visitBuilders(func(b *v2.Verifiers_Tryjob_Builder) {
		switch {
		case b.EquivalentTo != nil && b.EquivalentTo.Name != "" && names.Has(b.EquivalentTo.Name):
			ctx.Errorf("equivalent_to.name must not refer to already defined %q builder", b.EquivalentTo.Name)
		case b.TriggeredBy != "" && !names.Has(b.TriggeredBy):
			ctx.Errorf("triggered_by must refer to an existing builder, but %q given", b.TriggeredBy)
		case b.TriggeredBy != "" && !canBeTriggered.Has(b.TriggeredBy):
			// Although we can detect actual loops and emit better errors,
			// this happens so rarely, it's not yet worth the time.
			ctx.Errorf("triggered_by must refer to an existing builder without "+
				" equivalent_to, location_regexp, or experiment_percentage options. "+
				"triggered_by relationships must also not form a loop (given: %q)",
				b.TriggeredBy)
		case b.TriggeredBy != "":
			// Reaching here means parent exists in config.
			parent, _ := cfgByName[b.TriggeredBy]
			validateParentLocationRegexp(ctx, b, parent)
		}
	})
}

func validateBuilderName(ctx *validation.Context, name string, knownNames stringset.Set) {
	if name == "" {
		ctx.Errorf("name is required")
		return
	}
	if !knownNames.Add(name) {
		ctx.Errorf("duplicate name %q", name)
	}
	parts := strings.Split(name, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		ctx.Errorf("name %q doesn't match required format project/short-bucket-name/builder, e.g. 'v8/try/linux'", name)
	}
	for _, part := range parts {
		subs := strings.Split(part, ".")
		if len(subs) >= 3 && subs[0] == "luci" {
			// Technically, this is allowed. However, practically, this is
			// extremely likely to be misunderstanding of project or bucket is.
			ctx.Errorf("name %q is highly likely malformed; it should be project/short-bucket-name/builder, e.g. 'v8/try/linux'", name)
			return
		}
	}
	if parts[0] == "*" {
		ctx.Errorf("Buildbot builders are no longer allowed in CQ")
		return
	}
}

func validateEquivalentBuilder(ctx *validation.Context, b *v2.Verifiers_Tryjob_EquivalentBuilder, equiNames stringset.Set) {
	ctx.Enter("equivalent_to")
	defer ctx.Exit()
	validateBuilderName(ctx, b.Name, equiNames)
	if b.Percentage < 0 || b.Percentage > 100 {
		ctx.Errorf("percentage must be between 0 and 100 (%f given)", b.Percentage)
	}
}

func validateLocationRegexp(ctx *validation.Context, field string, values []string) {
	valid := stringset.New(len(values))
	for i, v := range values {
		if v == "" {
			ctx.Errorf("%s #%d: must not be empty", field, i+1)
		} else if _, err := regexp.Compile(v); err != nil {
			ctx.Errorf("%s %q: %s", field, v, err)
		} else if !valid.Add(v) {
			ctx.Errorf("duplicate %s: %q", field, v)
		}
	}
}

func validateParentLocationRegexp(ctx *validation.Context, child, parent *v2.Verifiers_Tryjob_Builder) {
	// Child's regexps shouldn't be less restrictive than parent.
	// While general check is not possible, in known so far use-cases, ensuring
	// the regexps are exact same expressions suffices and will prevent
	// accidentally incorrect configs.
	c := stringset.NewFromSlice(child.LocationRegexp...)
	p := stringset.NewFromSlice(parent.LocationRegexp...)
	if !p.Contains(c) {
		// This func is called in the context of a child.
		ctx.Errorf("location_regexp of a triggered builder must be a subset of its parent %q,"+
			" but these are not in parent: %s",
			parent.Name, strings.Join(c.Difference(p).ToSortedSlice(), ", "))
	}
	c = stringset.NewFromSlice(child.LocationRegexpExclude...)
	p = stringset.NewFromSlice(parent.LocationRegexpExclude...)
	if !c.Contains(p) {
		// This func is called in the context of a child.
		ctx.Errorf("location_regexp_exclude of a triggered builder must contain all those of its parent %q,"+
			" but these are only in parent: %s",
			parent.Name, strings.Join(p.Difference(c).ToSortedSlice(), ", "))
	}
}

func validateTryjobRetry(ctx *validation.Context, r *v2.Verifiers_Tryjob_RetryConfig) {
	if r.SingleQuota < 0 {
		ctx.Errorf("negative single_quota not allowed (%d given)", r.SingleQuota)
	}
	if r.GlobalQuota < 0 {
		ctx.Errorf("negative global_quota not allowed (%d given)", r.GlobalQuota)
	}
	if r.FailureWeight < 0 {
		ctx.Errorf("negative failure_weight not allowed (%d given)", r.FailureWeight)
	}
	if r.TransientFailureWeight < 0 {
		ctx.Errorf("negative transitive_failure_weight not allowed (%d given)", r.TransientFailureWeight)
	}
	if r.TimeoutWeight < 0 {
		ctx.Errorf("negative timeout_weight not allowed (%d given)", r.TimeoutWeight)
	}
}
