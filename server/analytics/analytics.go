// Copyright 2016 The LUCI Authors.
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

package analytics

// analytics.go is contains public functions for getting
// the Google Analytics ID and javascript snippets out from the admin settings.

import (
	"context"
	"fmt"
	"html/template"

	"github.com/tetrafolium/luci-go/common/logging"
)

// ID returns the Google Analytics ID if it's set, and "" otherwise.
func ID(c context.Context) string {
	return fetchCachedSettings(c).AnalyticsID
}

// Snippet returns the html snippet for Google Analytics, including the
// <script> tag and ID, if ID is set.
func Snippet(c context.Context) template.HTML {
	id := ID(c)
	if id == "" {
		return ""
	}
	if !rAllowed.MatchString(id) {
		logging.Errorf(c, "Analytics ID %s does not match UA-\\d+-\\d+", id)
		return ""
	}
	return template.HTML(fmt.Sprintf(`
<script>
setTimeout(function() {
	(function(i,s,o,g,r,a,m){i['GoogleAnalyticsObject']=r;i[r]=i[r]||function(){
	(i[r].q=i[r].q||[]).push(arguments)},i[r].l=1*new Date();a=s.createElement(o),
	m=s.getElementsByTagName(o)[0];a.async=1;a.src=g;m.parentNode.insertBefore(a,m)
	})(window,document,'script','https://www.google-analytics.com/analytics.js','ga');

	ga('create', '%s', 'auto');
	ga('send', 'pageview');
}, 0);
</script>
`, template.JSEscapeString(id)))
}
