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

type props struct {
	strs  []strProp
	bools []boolProp
	ints  []intProp
}

// strDefault is the value an unset string field has on Twilio, which is what a
// removed input is reset to. Every HTTP method defaults to POST.
func strDefault(param string) string {
	switch {
	case strings.HasSuffix(param, "Method"):
		return "POST"
	case param == "ScanMessageContent":
		return "inherit"
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
//
// Deliberately absent (the spec states no default, or the field is obsolete):
// StickySender, MmsConverter, SmartEncoding, FallbackToLongCode (obsolete),
// AreaCodeGeomatch, SynchronousValidation (reserved), UseInboundWebhookOnNumber,
// VoiceCallerIdLookup (the Application spec is silent), LookupEnabled,
// SkipSmsToLandlines, DtmfInputRequired, Psd2Enabled, CustomCodeEnabled and
// CodeLength.
var boolDefaults = map[string]bool{"DoNotShareWarningEnabled": false}

var intDefaults = map[string]int{"ValidityPeriod": 36000}

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
		case prev != nil && *prev.strs[i].p != nil && f.param != "FriendlyName":
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
}

// adopt fills the fields on import: every returned value that differs from the
// Twilio default becomes an input, so importing and then writing a program that
// does not mention the defaults plans no change. A bool or int without a
// verified default cannot be told from a setting, so every returned value is
// adopted as an input; removing it later leaves the field untouched on Twilio.
func (ps props) adopt(o obj) {
	for _, f := range ps.strs {
		if v := nonEmpty(o.optStr(f.key)); v != nil && *v != strDefault(f.param) {
			*f.p = v
		}
	}
	for _, f := range ps.bools {
		if v := o.optBool(f.key); v != nil {
			if d, ok := boolDefaults[f.param]; !ok || *v != d {
				*f.p = v
			}
		}
	}
	for _, f := range ps.ints {
		if v := o.optInt(f.key); v != nil {
			if d, ok := intDefaults[f.param]; !ok || *v != d {
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
