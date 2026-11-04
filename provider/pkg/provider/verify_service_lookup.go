package provider

import (
	"context"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// GetVerifyService looks up a Verify service by SID.
type GetVerifyService struct{}

// GetVerifyServiceArgs are the lookup arguments.
type GetVerifyServiceArgs struct {
	ServiceSid string `pulumi:"serviceSid"`
}

func (a *GetVerifyServiceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (VA...) of the Verify service. The Verify API is scoped to the credentials' account.")
}

// GetVerifyServiceResult is the looked-up service: the configuration Twilio reports plus its SID.
type GetVerifyServiceResult struct {
	VerifyServiceEffective

	Sid string `pulumi:"sid"`
}

func (a *GetVerifyServiceResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The Verify service SID (VA...).")
}

func (f *GetVerifyService) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Verify service by SID.")
}

func (f *GetVerifyService) Invoke(ctx context.Context, req infer.FunctionRequest[GetVerifyServiceArgs]) (infer.FunctionResponse[GetVerifyServiceResult], error) {
	const fn = "getVerifyService"
	if err := requireArg(fn, "serviceSid", req.Input.ServiceSid); err != nil {
		return infer.FunctionResponse[GetVerifyServiceResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetVerifyServiceResult]{}, err
	}
	o, err := c.do(ctx, "GET", "verify", verifyPath(req.Input.ServiceSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetVerifyServiceResult]{}, lookupFailure(fn, "verify service "+req.Input.ServiceSid, err)
	}
	return infer.FunctionResponse[GetVerifyServiceResult]{Output: GetVerifyServiceResult{
		VerifyServiceEffective: *verifyEffective(o), Sid: o.str("sid"),
	}}, nil
}
