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
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// ApiKey manages a Twilio API key (standard or restricted).
type ApiKey struct{}

// ApiKeyArgs are the inputs of an ApiKey.
type ApiKeyArgs struct {
	FriendlyName string  `pulumi:"friendlyName"`
	AccountSid   *string `pulumi:"accountSid,optional" provider:"replaceOnChanges"`
	KeyType      *string `pulumi:"keyType,optional" provider:"replaceOnChanges"`
	Policy       *string `pulumi:"policy,optional"`
}

func (a *ApiKeyArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the key (up to 64 characters).")
	an.Describe(&a.AccountSid, "SID of the account or subaccount the key belongs to. Defaults to the provider account. "+
		"Changing it replaces the resource. To import a key owned by a subaccount use the ID <accountSid>/<keySid>.")
	an.Describe(&a.KeyType, "standard (default; full access to the account's REST API except account and key management) "+
		"or restricted (limited to the permissions in policy). Changing it replaces the resource.")
	an.Describe(&a.Policy, "JSON policy document for a restricted key, for example "+
		"{\"allow\":[\"/twilio/iam/api-keys/create\"]}. Twilio only accepts assertions of the form "+
		"/twilio/<product>/<resource>/<action>; a REST path such as /v1/Services/** is rejected with error 70002. "+
		"Required when keyType is restricted and not allowed otherwise. "+
		"Updated in place and read back on refresh, so a policy widened outside Pulumi is detected and reverted. "+
		"Creating a restricted key lists the account's keys first; if Twilio creates the key and then fails the call, the new key "+
		"with this friendlyName is deleted, so keep friendlyNames unique among keys created together.")
	an.SetDefault(&a.KeyType, "standard")
}

// ApiKeyState is the persisted state of an ApiKey.
type ApiKeyState struct {
	ApiKeyArgs

	// Sid is the key SID (SK...) and the resource ID.
	Sid string `pulumi:"sid"`
	// Secret is returned by Twilio exactly once, at creation.
	Secret string `pulumi:"secret" provider:"secret"`
	// DateCreated is the RFC 2822 creation time.
	DateCreated string `pulumi:"dateCreated"`
}

func (s *ApiKeyState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The API key SID (SK...); use it as the username with the secret.")
	a.Describe(&s.Secret, "The API key secret. Twilio returns it only once, at creation, so it cannot be recovered "+
		"later or after import.")
	a.Describe(&s.DateCreated, "When the key was created (RFC 2822).")
}

func (r *ApiKey) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio API key. The secret is available only in the creation response and is stored as a "+
		"Pulumi secret. Creating and deleting keys requires the account auth token (an API key cannot manage keys).")
}

var _ infer.CustomResource[ApiKeyArgs, ApiKeyState] = (*ApiKey)(nil)
var _ infer.CustomCheck[ApiKeyArgs] = (*ApiKey)(nil)
var _ infer.CustomRead[ApiKeyArgs, ApiKeyState] = (*ApiKey)(nil)
var _ infer.CustomUpdate[ApiKeyArgs, ApiKeyState] = (*ApiKey)(nil)
var _ infer.CustomDelete[ApiKeyState] = (*ApiKey)(nil)

func (r *ApiKey) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ApiKeyArgs], error) {
	args, failures, err := infer.DefaultCheck[ApiKeyArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ApiKeyArgs]{Inputs: args, Failures: failures}, err
	}
	unk := unknowns(req.NewInputs)
	if !unk("keyType") {
		switch deref(args.KeyType) {
		case "standard":
			if args.Policy != nil || unk("policy") {
				failures = append(failures, p.CheckFailure{Property: "policy", Reason: "policy is only valid when keyType is restricted"})
			}
		case "restricted":
			switch {
			case unk("policy"):
				// Only known at apply time.
			case args.Policy == nil:
				failures = append(failures, p.CheckFailure{Property: "policy", Reason: "policy is required when keyType is restricted"})
			case !json.Valid([]byte(*args.Policy)):
				failures = append(failures, p.CheckFailure{Property: "policy", Reason: "policy must be valid JSON"})
			}
		default:
			failures = append(failures, p.CheckFailure{Property: "keyType", Reason: "keyType must be standard or restricted"})
		}
	} else if args.Policy != nil && !unk("policy") && !json.Valid([]byte(*args.Policy)) {
		failures = append(failures, p.CheckFailure{Property: "policy", Reason: "policy must be valid JSON"})
	}
	if !unk("policy") && args.Policy != nil {
		for _, bad := range unsupportedAssertions(*args.Policy) {
			p.GetLogger(ctx).Warningf("policy assertion %q is not of the form /twilio/<product>/<resource>/<action>: Twilio rejects it with error 70002, "+
				"and the create fails", bad)
		}
	}
	if !unk("friendlyName") && (args.FriendlyName == "" || len(args.FriendlyName) > 64) {
		failures = append(failures, p.CheckFailure{Property: "friendlyName", Reason: "friendlyName must be 1-64 characters"})
	}
	return infer.CheckResponse[ApiKeyArgs]{Inputs: args, Failures: failures}, nil
}

func keyPath(sid string) string { return "/v1/Keys/" + url.PathEscape(sid) }

// classicKeyPath is the account-explicit 2010-04-01 route of a key. The IAM
// route has no account in its path, so a key that belongs to a subaccount may
// not be visible through it with the parent's credentials.
func classicKeyPath(acct, sid string) string {
	return api2010(acct) + "/Keys/" + url.PathEscape(sid) + ".json"
}

// keyScope finds the account of an existing key from the ID, then the
// accountSid input in state, then the provider account.
func keyScope(c *Client, id string, in ApiKeyArgs) (acct, sid string) {
	return resolveScope(c, id, deref(in.AccountSid))
}

// fetchKey reads a key through the IAM route and, when that cannot see it and
// the key belongs to another account, through the account-explicit route.
// classic reports that the fallback answered (it does not return the policy).
func fetchKey(ctx context.Context, c *Client, acct, sid string) (o obj, classic bool, err error) {
	o, err = c.do(ctx, "GET", "iam", keyPath(sid), nil, nil)
	if err == nil || !isNotFound(err) || acct == c.accountSid {
		return o, false, err
	}
	o, err = c.do(ctx, "GET", "api", classicKeyPath(acct, sid), nil, nil)
	return o, true, err
}

// policyJSON renders the policy object of a key response ("" when none).
func policyJSON(o obj) string {
	if pol, ok := o["policy"].(map[string]any); ok && len(pol) > 0 {
		b, _ := json.Marshal(pol)
		return string(b)
	}
	return ""
}

func sameJSON(a, b string) bool {
	var x, y any
	if json.Unmarshal([]byte(a), &x) != nil || json.Unmarshal([]byte(b), &y) != nil {
		return a == b
	}
	return reflect.DeepEqual(x, y)
}

// policyCovered reports whether the policy Twilio returned still contains
// everything the program supplied. Only the keys the program wrote are
// compared, so extra fields Twilio echoes back are not permanent drift.
func policyCovered(supplied, returned string) bool {
	var x, y any
	if json.Unmarshal([]byte(supplied), &x) != nil || json.Unmarshal([]byte(returned), &y) != nil {
		return supplied == returned
	}
	return jsonCovers(x, y)
}

func jsonCovers(want, got any) bool {
	wm, ok := want.(map[string]any)
	if !ok {
		return reflect.DeepEqual(want, got)
	}
	gm, ok := got.(map[string]any)
	if !ok {
		return false
	}
	for k, wv := range wm {
		gv, present := gm[k]
		if !present || !jsonCovers(wv, gv) {
			return false
		}
	}
	return true
}

func (r *ApiKey) Create(ctx context.Context, req infer.CreateRequest[ApiKeyArgs]) (infer.CreateResponse[ApiKeyState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[ApiKeyState]{ID: "", Output: ApiKeyState{ApiKeyArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[ApiKeyState]{}, err
	}
	acct := c.acct(a.AccountSid)
	form := url.Values{"AccountSid": {acct}, "FriendlyName": {a.FriendlyName}}
	if deref(a.KeyType) == "restricted" { // omit for a standard key: the API documents no "standard" value
		form.Set("KeyType", "restricted")
	}
	setS(form, "Policy", a.Policy)
	// Twilio can create the key and still fail the call (error 70002 for a policy assertion it
	// rejects), and the error carries no SID. Note the account's keys first so a failed create
	// can find, and remove, the untracked credential it left behind (#1574).
	before, beforeErr := listKeyNames(ctx, c, acct)
	o, err := c.do(ctx, "POST", "iam", "/v1/Keys", nil, form)
	if err != nil {
		return infer.CreateResponse[ApiKeyState]{}, cleanupFailedKeyCreate(ctx, c, acct, a.FriendlyName, before, beforeErr, err)
	}
	return infer.CreateResponse[ApiKeyState]{ID: scopedID(c, acct, o.str("sid")), Output: ApiKeyState{
		ApiKeyArgs: a, Sid: o.str("sid"), Secret: o.str("secret"), DateCreated: o.str("date_created"),
	}}, nil
}

func (r *ApiKey) Read(ctx context.Context, req infer.ReadRequest[ApiKeyArgs, ApiKeyState]) (infer.ReadResponse[ApiKeyArgs, ApiKeyState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[ApiKeyArgs, ApiKeyState]{}, err
	}
	acct, sid := keyScope(c, req.ID, req.Inputs)
	if idAcct, _ := splitScoped(req.ID); idAcct != "" && req.State.Sid == "" {
		acct = idAcct // import: the ID names the account
	}
	o, classic, err := fetchKey(ctx, c, acct, sid)
	if err != nil {
		if isNotFound(err) {
			return infer.ReadResponse[ApiKeyArgs, ApiKeyState]{}, nil
		}
		return infer.ReadResponse[ApiKeyArgs, ApiKeyState]{}, fmt.Errorf("read API key %s: %w", req.ID, err)
	}
	in := req.Inputs
	st := req.State
	imported := req.Inputs.FriendlyName == ""
	in.FriendlyName = o.str("friendly_name")
	pol := policyJSON(o)
	switch {
	case imported: // infer the key type from the presence of a policy
		if acct != c.accountSid {
			in.AccountSid = &acct
		}
		if pol != "" {
			kt := "restricted"
			in.Policy, in.KeyType = &pol, &kt
		} else {
			kt := "standard"
			in.KeyType = &kt
		}
	case classic:
		// The classic route does not return the policy: keep what is in state.
	case pol != "":
		if in.Policy == nil || !policyCovered(*in.Policy, pol) {
			in.Policy = &pol // drift: the policy was changed outside Pulumi
		}
	default:
		in.Policy = nil // the key has no policy any more
	}
	st.ApiKeyArgs = in
	st.Sid = sid
	st.DateCreated = o.str("date_created")
	return infer.ReadResponse[ApiKeyArgs, ApiKeyState]{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *ApiKey) Update(ctx context.Context, req infer.UpdateRequest[ApiKeyArgs, ApiKeyState]) (infer.UpdateResponse[ApiKeyState], error) {
	a := req.Inputs
	st := req.State
	st.ApiKeyArgs = a
	if req.DryRun {
		return infer.UpdateResponse[ApiKeyState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[ApiKeyState]{}, err
	}
	acct, sid := keyScope(c, req.ID, req.State.ApiKeyArgs)
	form := url.Values{"FriendlyName": {a.FriendlyName}}
	setS(form, "Policy", a.Policy)
	_, err = c.do(ctx, "POST", "iam", keyPath(sid), nil, form)
	if err != nil && isNotFound(err) && acct != c.accountSid {
		// The IAM route cannot see a subaccount's key: rename it through the account-explicit route.
		if !sameJSON(deref(a.Policy), deref(req.State.Policy)) {
			return infer.UpdateResponse[ApiKeyState]{}, fmt.Errorf("update API key %s: the policy of a subaccount key cannot be changed through the account-explicit API: %w", req.ID, err)
		}
		_, err = c.do(ctx, "POST", "api", classicKeyPath(acct, sid), nil, url.Values{"FriendlyName": {a.FriendlyName}})
	}
	if err != nil {
		return infer.UpdateResponse[ApiKeyState]{}, fmt.Errorf("update API key %s: %w", req.ID, err)
	}
	return infer.UpdateResponse[ApiKeyState]{Output: st}, nil
}

func (r *ApiKey) Delete(ctx context.Context, req infer.DeleteRequest[ApiKeyState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	acct, sid := keyScope(c, req.ID, req.State.ApiKeyArgs)
	err = deleteKey(ctx, c, acct, sid)
	if err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete API key %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// deleteKey removes a key through the IAM route and, for a key of another account that route
// cannot see, through the account-explicit route, so a live credential is never left behind.
func deleteKey(ctx context.Context, c *Client, acct, sid string) error {
	_, err := c.do(ctx, "DELETE", "iam", keyPath(sid), nil, nil)
	if err != nil && isNotFound(err) && acct != c.accountSid {
		// A 404 from the IAM route does not prove the key is gone when it belongs to a subaccount.
		_, err = c.do(ctx, "DELETE", "api", classicKeyPath(acct, sid), nil, nil)
	}
	return err
}

// maxKeyListPages bounds the walk over an account's keys.
const maxKeyListPages = 10

// listKeyNames returns the SID and friendly name of every key of an account, read through the
// account-explicit route (the IAM route has no list that names the account).
func listKeyNames(ctx context.Context, c *Client, acct string) (map[string]string, error) {
	out := map[string]string{}
	path, query := api2010(acct)+"/Keys.json", url.Values{"PageSize": {"1000"}}
	for page := 0; page < maxKeyListPages && path != ""; page++ {
		o, err := c.do(ctx, "GET", "api", path, query, nil)
		if err != nil {
			return nil, err
		}
		items, _ := o["keys"].([]any)
		for _, it := range items {
			if k, ok := it.(map[string]any); ok {
				out[obj(k).str("sid")] = obj(k).str("friendly_name")
			}
		}
		path, query = o.str("next_page_uri"), nil
	}
	if path != "" {
		// More pages than we read: a key that existed before could look new, so refuse rather than guess.
		return nil, fmt.Errorf("the account has more than %d pages of keys", maxKeyListPages)
	}
	return out, nil
}

// cleanupFailedKeyCreate turns a failed key create into an error that says what became of any key
// Twilio created during the call. A key that is new since before and carries the friendly name of
// this create cannot belong to anyone else, so it is deleted; if that cannot be established or
// done, the error names what to look for, because the key is a live credential Pulumi does not track.
func cleanupFailedKeyCreate(ctx context.Context, c *Client, acct, friendlyName string, before map[string]string, beforeErr, createErr error) error {
	base := fmt.Errorf("create API key: %w", createErr)
	if beforeErr != nil && isNotFound(beforeErr) {
		return base // the account does not exist or is not visible: no key can have been created in it
	}
	if beforeErr != nil {
		return fmt.Errorf("%w; Twilio may have created a key named %q before failing and the account's keys could not be listed to check (%v): look for it in the Twilio console and delete it if it is not in use", base, friendlyName, beforeErr)
	}
	after, err := listKeyNames(ctx, c, acct)
	if err != nil {
		return fmt.Errorf("%w; Twilio may have created a key named %q before failing and the account's keys could not be listed to check (%v): look for it in the Twilio console and delete it if it is not in use", base, friendlyName, err)
	}
	var orphans []string
	for sid, name := range after {
		if _, existed := before[sid]; !existed && name == friendlyName {
			orphans = append(orphans, sid)
		}
	}
	sort.Strings(orphans)
	if len(orphans) == 0 {
		return base
	}
	var removed, stuck []string
	for _, sid := range orphans {
		if err := deleteKey(ctx, c, acct, sid); err != nil && !isNotFound(err) {
			stuck = append(stuck, fmt.Sprintf("%s (%v)", sid, err))
			continue
		}
		removed = append(removed, sid)
	}
	switch {
	case len(stuck) > 0:
		return fmt.Errorf("%w; Twilio created the key(s) %s during the failed call and they could not be deleted: delete them in the Twilio console, they are live credentials Pulumi does not track", base, strings.Join(stuck, ", "))
	default:
		return fmt.Errorf("%w; the key Twilio created during the failed call (%s) was deleted so no untracked credential is left behind", base, strings.Join(removed, ", "))
	}
}

// unsupportedAssertions returns the allow and deny entries of a restricted key's policy that do not
// start with /twilio/. Twilio rejects those (a REST path such as /v1/Services/** answers error 70002).
// The full grammar is not documented, so Check only warns about this one verified-invalid class.
func unsupportedAssertions(policy string) []string {
	var doc map[string]any
	if json.Unmarshal([]byte(policy), &doc) != nil {
		return nil
	}
	var bad []string
	for _, k := range []string{"allow", "deny"} {
		items, _ := doc[k].([]any)
		for _, it := range items {
			if a, ok := it.(string); ok && !strings.HasPrefix(a, "/twilio/") {
				bad = append(bad, a)
			}
		}
	}
	return bad
}
