package provider

import (
	"context"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// GetApplication looks up a TwiML application by SID.
type GetApplication struct{}

// GetApplicationArgs are the lookup arguments.
type GetApplicationArgs struct {
	ApplicationSid string  `pulumi:"applicationSid"`
	AccountSid     *string `pulumi:"accountSid,optional"`
}

func (a *GetApplicationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ApplicationSid, "SID (AP...) of the application.")
	an.Describe(&a.AccountSid, "SID of the account that owns the application. Defaults to the provider account.")
}

// GetApplicationResult is the looked-up application: the configuration Twilio reports plus its identity.
type GetApplicationResult struct {
	ApplicationEffective

	Sid             string `pulumi:"sid"`
	OwnerAccountSid string `pulumi:"ownerAccountSid"`
}

func (a *GetApplicationResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The application SID (AP...).")
	an.Describe(&a.OwnerAccountSid, "SID of the account that owns the application.")
}

func (f *GetApplication) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a TwiML application by SID.")
}

func (f *GetApplication) Invoke(ctx context.Context, req infer.FunctionRequest[GetApplicationArgs]) (infer.FunctionResponse[GetApplicationResult], error) {
	const fn = "getApplication"
	if err := requireArg(fn, "applicationSid", req.Input.ApplicationSid); err != nil {
		return infer.FunctionResponse[GetApplicationResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetApplicationResult]{}, err
	}
	acct := c.acct(req.Input.AccountSid)
	o, err := c.do(ctx, "GET", "api", appPath(acct, req.Input.ApplicationSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetApplicationResult]{}, lookupFailure(fn, "application "+req.Input.ApplicationSid, err)
	}
	return infer.FunctionResponse[GetApplicationResult]{Output: GetApplicationResult{
		ApplicationEffective: *applicationEffective(o), Sid: o.str("sid"), OwnerAccountSid: acct,
	}}, nil
}
