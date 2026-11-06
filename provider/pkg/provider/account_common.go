// Copyright 2026, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"net/url"
	"strconv"
	"strings"
)

// acctPath is the path of a collection (sid empty) or instance of an account-scoped
// 2010-04-01 resource, for example acctPath(acct, "Queues", sid).
func acctPath(account, coll, sid string) string {
	if sid == "" {
		return api2010(account) + "/" + coll + ".json"
	}
	return api2010(account) + "/" + coll + "/" + url.PathEscape(sid) + ".json"
}

func isTrue(b *bool) bool { return b != nil && *b }

func intOf(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// sameNumber reports whether two decimal strings denote the same number; strings that are
// not numbers are compared literally.
func sameNumber(a, b string) bool {
	x, err1 := strconv.ParseFloat(strings.TrimSpace(a), 64)
	y, err2 := strconv.ParseFloat(strings.TrimSpace(b), 64)
	if err1 != nil || err2 != nil {
		return a == b
	}
	return x == y
}
