package provider

import (
	"context"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// GetMessagingService looks up a Messaging Service by SID.
type GetMessagingService struct{}

// GetMessagingServiceArgs are the lookup arguments.
type GetMessagingServiceArgs struct {
	ServiceSid string `pulumi:"serviceSid"`
}

func (a *GetMessagingServiceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ServiceSid, "SID (MG...) of the Messaging Service. The Messaging API is scoped to the credentials' account.")
}

// GetMessagingServiceResult is the looked-up service: the configuration Twilio reports plus its SID.
type GetMessagingServiceResult struct {
	MessagingServiceEffective

	Sid string `pulumi:"sid"`
}

func (a *GetMessagingServiceResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The service SID (MG...).")
}

func (f *GetMessagingService) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Messaging Service by SID.")
}

func (f *GetMessagingService) Invoke(ctx context.Context, req infer.FunctionRequest[GetMessagingServiceArgs]) (infer.FunctionResponse[GetMessagingServiceResult], error) {
	const fn = "getMessagingService"
	if err := requireArg(fn, "serviceSid", req.Input.ServiceSid); err != nil {
		return infer.FunctionResponse[GetMessagingServiceResult]{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetMessagingServiceResult]{}, err
	}
	o, err := c.do(ctx, "GET", "messaging", serviceBase(req.Input.ServiceSid), nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetMessagingServiceResult]{}, lookupFailure(fn, "messaging service "+req.Input.ServiceSid, err)
	}
	return infer.FunctionResponse[GetMessagingServiceResult]{Output: GetMessagingServiceResult{
		MessagingServiceEffective: *messagingEffective(o), Sid: o.str("sid"),
	}}, nil
}
