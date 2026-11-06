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

	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

// A resource's optional wire fields are described once as a props table; the
// same table drives the create/update form, the refresh from an API response,
// the import of every returned field and the reset of fields removed from the
// program (where Twilio documents a default to reset to).

type strProp struct {
	param, key string
	p          **string
}

type boolProp struct {
	param, key string
	p          **bool
}

type intProp struct {
	param, key string
	p          **int
}

// listProp is a list of strings sent as one comma-separated form value.
type listProp struct {
	param, key string
	p          **[]string
}

type props struct {
	strs  []strProp
	bools []boolProp
	ints  []intProp
	lists []listProp
}

// noStrReset names the string fields that are never reset when removed from the
// program: Twilio states no default for them and "" is not a valid value, so a
// removed input leaves the setting as it is on Twilio.
var noStrReset = map[string]bool{
	"VoiceReceiveMode": true, "EmergencyStatus": true, "ApiVersion": true,
	"Usecase": true, "DefaultTemplateSid": true,
	// The TOTP issuer: Twilio defaults it to the friendly name at creation and answers HTTP 400 to an empty
	// value, so a removed input leaves the issuer as it is (#1633).
	"Totp.Issuer": true,
	// Connect app listing details: the app form requires them, so a removed input leaves them as they are.
	"CompanyName": true, "Description": true, "HomepageUrl": true, "AuthorizeRedirectUrl": true,
}

// noListReset names the list fields that are never reset when removed from the program (the
// permissions of a Connect app: Twilio states no empty value).
var noListReset = map[string]bool{"Permissions": true}

// noAdopt names the string fields that an import never turns into an input
// because Twilio always returns a value for them (the API version in use).
var noAdopt = map[string]bool{"ApiVersion": true}

// strDefault is the value an unset string field has on Twilio, which is what a
// removed input is reset to. Every HTTP method defaults to POST. VoiceReceiveMode
// and EmergencyStatus are in noStrReset, so for them the value only decides what an
// import adopts: Twilio's default is not turned into an input.
func strDefault(param string) string {
	switch {
	case strings.HasSuffix(param, "Method"):
		return "POST"
	case param == "ScanMessageContent":
		return "inherit"
	case param == "VoiceReceiveMode":
		return "voice"
	case param == "EmergencyStatus":
		return "Inactive"
	}
	return ""
}

// boolDefaults and intDefaults hold only the defaults that Twilio's own OpenAPI
// definitions (twilio-oai, spec/json) state for the field. A removed input with
// an entry here is reset to that value; a removed input without one is left
// untouched on Twilio (not sent), because a reset to a guess is worse than
// doing nothing. Sources:
//   - ValidityPeriod 36000: twilio_messaging_v1 POST /v1/Services, "Default
//     value is 36,000" (the Messaging Service field this provider manages).
//   - DoNotShareWarningEnabled false: twilio_verify_v2 POST /v2/Services,
//     "Disabled by default".
//   - MaxSize 1000: twilio_api_v2010 POST Queues, "The default is 1000".
//   - CidrPrefixLength 32: twilio_api_v2010 POST SIP IpAddresses, "By default the entire IP
//     address is used" (IPv4 only, so a /32).
//
// Deliberately absent (the spec states no default, or the field is obsolete):
// StickySender, MmsConverter, SmartEncoding, FallbackToLongCode (obsolete),
// AreaCodeGeomatch, SynchronousValidation (reserved), UseInboundWebhookOnNumber,
// VoiceCallerIdLookup (the Application spec is silent), LookupEnabled,
// SkipSmsToLandlines, DtmfInputRequired, Psd2Enabled, CustomCodeEnabled and
// CodeLength.
var boolDefaults = map[string]bool{"DoNotShareWarningEnabled": false}

var intDefaults = map[string]int{"ValidityPeriod": 36000, "MaxSize": 1000, "CidrPrefixLength": 32}

// importBoolDefaults and importIntDefaults are the values a freshly created resource reports
// for fields whose default Twilio's OpenAPI does not state (#1595). They are used by adopt
// only: an import adopts a returned value that differs from them and leaves one that equals
// them out of the inputs (#1572). They are deliberately separate from boolDefaults and
// intDefaults, which also decide what an update sends when an input is removed, so an
// observed value must never be added there. The entries are the defaults reported by the
// live API in the Mirror report behind #1572 (pulumi-labs/pulumi-twilio#2); they have not
// been re-probed per account, and Twilio may roll out different defaults. A wrong entry
// either brings back the #1572 spurious update or drops a real setting, so extend this map
// only from a probe of a freshly created resource.
var importBoolDefaults = map[string]bool{
	"VoiceCallerIdLookup": false, "StickySender": true, "AreaCodeGeomatch": true, "MmsConverter": true,
	"SmartEncoding": true, "SynchronousValidation": false, "UseInboundWebhookOnNumber": false,
	"CustomCodeEnabled": false, "DtmfInputRequired": true, "Psd2Enabled": false, "SkipSmsToLandlines": false,
}

var importIntDefaults = map[string]int{"CodeLength": 6}

// importStrDefaults are the string values a freshly created resource reports (#1634): an import leaves
// an equal value out of the inputs, like importBoolDefaults. MessagingService usecase is "undeclared"
// until set. Like the other import maps this is never used to reset a removed input; the observed
// values come from the Mirror report behind #1634 (pulumi-labs/pulumi-twilio#2). VerifyService
// totpIssuer is a default too, but it is the friendly name, so adopt compares it with that.
var importStrDefaults = map[string]string{"Usecase": "undeclared"}

// importBoolDefault returns the default an import compares a returned bool with.
func importBoolDefault(param string) (bool, bool) {
	if d, ok := boolDefaults[param]; ok {
		return d, true
	}
	d, ok := importBoolDefaults[param]
	return d, ok
}

// importIntDefault returns the default an import compares a returned number with.
func importIntDefault(param string) (int, bool) {
	if d, ok := intDefaults[param]; ok {
		return d, true
	}
	d, ok := importIntDefaults[param]
	return d, ok
}

// encode builds the form for a create or update. When prev is non-nil (an
// update), every field that was set before but is now unset is reset to its
// Twilio default (strings to "" or their default enum value), so removing an
// input reaches the API. A bool or int is reset only when Twilio's OpenAPI
// states its default (boolDefaults, intDefaults); otherwise it is left
// untouched. The FriendlyName of a phone number has no default to reset to and
// is left alone.
func (ps props) encode(v url.Values, prev *props) {
	for i, f := range ps.strs {
		switch {
		case *f.p != nil:
			v.Set(f.param, **f.p)
		case prev != nil && *prev.strs[i].p != nil && f.param != "FriendlyName" && !noStrReset[f.param]:
			if d := strDefault(f.param); **prev.strs[i].p != d {
				v.Set(f.param, d)
			}
		}
	}
	for i, f := range ps.bools {
		switch {
		case *f.p != nil:
			v.Set(f.param, strconv.FormatBool(**f.p))
		case prev != nil && *prev.bools[i].p != nil:
			if d, ok := boolDefaults[f.param]; ok {
				v.Set(f.param, strconv.FormatBool(d))
			}
		}
	}
	for i, f := range ps.ints {
		switch {
		case *f.p != nil:
			v.Set(f.param, strconv.Itoa(**f.p))
		case prev != nil && *prev.ints[i].p != nil:
			if d, ok := intDefaults[f.param]; ok {
				v.Set(f.param, strconv.Itoa(d))
			}
		}
	}
	for i, f := range ps.lists {
		switch {
		case *f.p != nil:
			v.Set(f.param, strings.Join(**f.p, ","))
		case prev != nil && *prev.lists[i].p != nil && !noListReset[f.param]:
			v.Set(f.param, "")
		}
	}
}

// nonEmpty returns nil for a missing or empty string.
func nonEmpty(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}

// refreshSet copies API values into the fields the program set (drift
// detection on the inputs without turning server defaults into permanent
// diffs). A field the program left unset is never touched; its server value is
// exposed through the resource's effective output instead.
func (ps props) refreshSet(o obj) {
	for _, f := range ps.strs {
		if *f.p == nil {
			continue
		}
		was := **f.p
		v := nonEmpty(o.optStr(f.key))
		if v == nil && was == "" {
			continue // an explicit empty string stays as the program wrote it
		}
		*f.p = v
	}
	for _, f := range ps.bools {
		if *f.p != nil {
			*f.p = o.optBool(f.key)
		}
	}
	for _, f := range ps.ints {
		if *f.p != nil {
			*f.p = o.optInt(f.key)
		}
	}
	for _, f := range ps.lists {
		if *f.p != nil {
			was := **f.p
			v := nonEmptyList(o.optList(f.key))
			if v == nil && len(was) == 0 {
				continue // an explicit empty list stays as the program wrote it
			}
			*f.p = v
		}
	}
}

// nonEmptyList returns nil for a missing or empty list.
func nonEmptyList(l *[]string) *[]string {
	if l == nil || len(*l) == 0 {
		return nil
	}
	return l
}

// adopt fills the fields on import: every returned value that differs from the
// Twilio default becomes an input, so importing and then writing a program that
// does not mention the defaults plans no change. A bool or int is adopted only
// when a default is known (boolDefaults, intDefaults, or the import-only
// importBoolDefaults, importIntDefaults) and the returned value differs from it. Without a verified default a returned value
// cannot be told from the server's own default, and adopting it would make a
// program that sets only the required fields plan to remove it on the next
// preview (#1572); such a field is left unset, and declaring it later is an
// ordinary update.
func (ps props) adopt(o obj) {
	for _, f := range ps.strs {
		v := nonEmpty(o.optStr(f.key))
		if v == nil || *v == strDefault(f.param) || noAdopt[f.param] {
			continue
		}
		if d, ok := importStrDefaults[f.param]; ok && *v == d {
			continue
		}
		if f.param == "Totp.Issuer" && *v == o.str("friendly_name") {
			continue // Twilio sets the issuer to the friendly name at creation
		}
		*f.p = v
	}
	for _, f := range ps.lists {
		if v := nonEmptyList(o.optList(f.key)); v != nil {
			*f.p = v
		}
	}
	for _, f := range ps.bools {
		if v := o.optBool(f.key); v != nil {
			if d, ok := importBoolDefault(f.param); ok && *v != d {
				*f.p = v
			}
		}
	}
	for _, f := range ps.ints {
		if v := o.optInt(f.key); v != nil {
			if d, ok := importIntDefault(f.param); ok && *v != d {
				*f.p = v
			}
		}
	}
}

// fillAll sets every returned field (the effective server-side configuration).
func (ps props) fillAll(o obj) {
	for _, f := range ps.strs {
		*f.p = nonEmpty(o.optStr(f.key))
	}
	for _, f := range ps.bools {
		*f.p = o.optBool(f.key)
	}
	for _, f := range ps.ints {
		*f.p = o.optInt(f.key)
	}
	for _, f := range ps.lists {
		*f.p = nonEmptyList(o.optList(f.key))
	}
}

// unknowns reports whether an input is a preview-time unknown. infer decodes an
// unknown into its zero value, so a Check that validated that value would
// reject programs that wire in a computed value.
func unknowns(m property.Map) func(keys ...string) bool {
	return func(keys ...string) bool {
		for _, k := range keys {
			if v, ok := m.GetOk(k); ok && v.IsComputed() {
				return true
			}
		}
		return false
	}
}

// splitScoped splits an ID of the form <accountSid>/<sid>; a bare SID has no
// account.
func splitScoped(id string) (acct, sid string) {
	if i := strings.IndexByte(id, '/'); i >= 0 {
		return id[:i], id[i+1:]
	}
	return "", id
}

// scopedID is the resource ID of an object owned by acct: the bare SID for the
// provider's own account (the original format) and <accountSid>/<sid> for an
// object in a subaccount, which makes it importable.
func scopedID(c *Client, acct, sid string) string {
	if acct == "" || acct == c.accountSid {
		return sid
	}
	return acct + "/" + sid
}

// resolveScope finds the owning account and SID of an existing object. The
// owner recorded in state wins (a number can move between accounts without its
// ID changing), then the account in the ID (import), then the provider account.
func resolveScope(c *Client, id, stateOwner string) (acct, sid string) {
	idAcct, sid := splitScoped(id)
	switch {
	case stateOwner != "":
		return stateOwner, sid
	case idAcct != "":
		return idAcct, sid
	}
	return c.accountSid, sid
}
