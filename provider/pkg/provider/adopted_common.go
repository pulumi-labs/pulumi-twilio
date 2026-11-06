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
	"context"
	"fmt"
	"net/url"
)

// Short codes, Connect apps and verified caller IDs cannot be created through the Twilio REST
// API in a way a deployment could rely on (short codes are leased through Twilio support, a
// Connect app is created in the Console and a caller ID needs a verification call to the
// number). The three resources therefore adopt an existing object, manage its mutable
// settings, and never delete it on destroy.

// adoptedColl describes one account-scoped collection of adoptable objects.
type adoptedColl struct {
	coll    string // path segment under the account, for example SMS/ShortCodes
	listKey string // key of the list in a list response
	what    string // for error messages
}

var (
	shortCodeColl  = adoptedColl{"SMS/ShortCodes", "short_codes", "short code"}
	connectAppColl = adoptedColl{"ConnectApps", "connect_apps", "Connect app"}
	callerIDColl   = adoptedColl{"OutgoingCallerIds", "outgoing_caller_ids", "verified caller ID"}
)

// fetchAdopted reads one object of the collection by SID.
func fetchAdopted(ctx context.Context, c *Client, acct string, ac adoptedColl, sid string) (obj, error) {
	o, err := c.do(ctx, "GET", "api", acctPath(acct, ac.coll, sid), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("account %s has no %s with SID %s", acct, ac.what, sid)
		}
		return nil, err
	}
	return o, nil
}

// findAdopted lists the collection filtered by param and returns the object whose field equals
// want exactly.
func findAdopted(ctx context.Context, c *Client, acct string, ac adoptedColl, param, field, want string) (obj, error) {
	o, err := c.do(ctx, "GET", "api", acctPath(acct, ac.coll, ""), url.Values{param: {want}}, nil)
	if err != nil {
		return nil, err
	}
	list, _ := o[ac.listKey].([]any)
	for _, it := range list {
		if n, ok := it.(map[string]any); ok && obj(n).str(field) == want {
			return obj(n), nil
		}
	}
	return nil, fmt.Errorf("account %s has no %s %s (this resource adopts an existing %s, it cannot create one)", acct, ac.what, want, ac.what)
}
