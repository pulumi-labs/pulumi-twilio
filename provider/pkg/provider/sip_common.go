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
	"fmt"
	"net/url"
	"strings"
)

// sipPath is the path of a collection (sid empty) or instance of a resource under
// /Accounts/{AccountSid}/SIP. coll is relative to SIP, for example "CredentialLists".
func sipPath(account, coll, sid string) string {
	return acctPath(account, "SIP/"+coll, sid)
}

// sipChildPath is the path of a collection (sid empty) or instance of a resource that
// belongs to a SIP list, for example CredentialLists/{parent}/Credentials.
func sipChildPath(account, parentColl, parent, coll, sid string) string {
	return sipPath(account, parentColl+"/"+url.PathEscape(parent)+"/"+coll, sid)
}

// splitChildID splits the ID of an object that lives under a parent list. A bare
// <parentSid>/<sid> belongs to the provider account; <accountSid>/<parentSid>/<sid> names
// the owning account. The two forms are told apart by their number of segments.
func splitChildID(id string) (acct, parent, sid string, err error) {
	parts := strings.Split(id, "/")
	switch len(parts) {
	case 2:
		return "", parts[0], parts[1], nil
	case 3:
		return parts[0], parts[1], parts[2], nil
	}
	return "", "", "", fmt.Errorf("invalid ID %q: expected <parentSid>/<sid> or <accountSid>/<parentSid>/<sid>", id)
}

// childID is the resource ID of a child object: <parentSid>/<sid> for the provider's own
// account and <accountSid>/<parentSid>/<sid> for a subaccount, which makes it importable.
func childID(c *Client, acct, parent, sid string) string {
	if acct == "" || acct == c.accountSid {
		return parent + "/" + sid
	}
	return acct + "/" + parent + "/" + sid
}

// resolveChildScope finds the owning account, parent list and SID of an existing child. The
// owner recorded in state wins, then the account in the ID (import), then the provider account.
func resolveChildScope(c *Client, id, stateOwner string) (acct, parent, sid string, err error) {
	idAcct, parent, sid, err := splitChildID(id)
	if err != nil {
		return "", "", "", err
	}
	switch {
	case stateOwner != "":
		return stateOwner, parent, sid, nil
	case idAcct != "":
		return idAcct, parent, sid, nil
	}
	return c.accountSid, parent, sid, nil
}
