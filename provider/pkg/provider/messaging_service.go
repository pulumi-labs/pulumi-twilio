package provider

import (
	"context"
	"fmt"
	"net/url"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// MessagingService manages a Twilio Messaging Service (a sender pool with shared routing).
type MessagingService struct{}

// MessagingServiceArgs are the inputs of a MessagingService.
type MessagingServiceArgs struct {
	FriendlyName              string  `pulumi:"friendlyName"`
	InboundRequestUrl         *string `pulumi:"inboundRequestUrl,optional"`
	InboundMethod             *string `pulumi:"inboundMethod,optional"`
	FallbackUrl               *string `pulumi:"fallbackUrl,optional"`
	FallbackMethod            *string `pulumi:"fallbackMethod,optional"`
	StatusCallback            *string `pulumi:"statusCallback,optional"`
	ScanMessageContent        *string `pulumi:"scanMessageContent,optional"`
	StickySender              *bool   `pulumi:"stickySender,optional"`
	MmsConverter              *bool   `pulumi:"mmsConverter,optional"`
	SmartEncoding             *bool   `pulumi:"smartEncoding,optional"`
	FallbackToLongCode        *bool   `pulumi:"fallbackToLongCode,optional"`
	AreaCodeGeomatch          *bool   `pulumi:"areaCodeGeomatch,optional"`
	SynchronousValidation     *bool   `pulumi:"synchronousValidation,optional"`
	UseInboundWebhookOnNumber *bool   `pulumi:"useInboundWebhookOnNumber,optional"`
	ValidityPeriod            *int    `pulumi:"validityPeriod,optional"`
}

func (a *MessagingServiceArgs) props() props {
	return props{
		strs: []strProp{
			{"InboundRequestUrl", "inbound_request_url", &a.InboundRequestUrl},
			{"InboundMethod", "inbound_method", &a.InboundMethod},
			{"FallbackUrl", "fallback_url", &a.FallbackUrl}, {"FallbackMethod", "fallback_method", &a.FallbackMethod},
			{"StatusCallback", "status_callback", &a.StatusCallback},
			{"ScanMessageContent", "scan_message_content", &a.ScanMessageContent},
		},
		bools: []boolProp{
			{"StickySender", "sticky_sender", &a.StickySender}, {"MmsConverter", "mms_converter", &a.MmsConverter},
			{"SmartEncoding", "smart_encoding", &a.SmartEncoding},
			{"FallbackToLongCode", "fallback_to_long_code", &a.FallbackToLongCode},
			{"AreaCodeGeomatch", "area_code_geomatch", &a.AreaCodeGeomatch},
			{"SynchronousValidation", "synchronous_validation", &a.SynchronousValidation},
			{"UseInboundWebhookOnNumber", "use_inbound_webhook_on_number", &a.UseInboundWebhookOnNumber},
		},
		ints: []intProp{{"ValidityPeriod", "validity_period", &a.ValidityPeriod}},
	}
}

func (a *MessagingServiceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Human-readable name of the service (up to 64 characters).")
	an.Describe(&a.InboundRequestUrl, "URL Twilio requests for incoming messages to any sender in the pool.")
	an.Describe(&a.InboundMethod, "HTTP method for inboundRequestUrl: GET or POST.")
	an.Describe(&a.FallbackUrl, "URL requested if inboundRequestUrl fails.")
	an.Describe(&a.FallbackMethod, "HTTP method for fallbackUrl: GET or POST.")
	an.Describe(&a.StatusCallback, "URL that receives status events for messages sent through the service.")
	an.Describe(&a.ScanMessageContent, "Content inspection: inherit, enable or disable.")
	an.Describe(&a.StickySender, "Whether to keep using the same sender for a given recipient.")
	an.Describe(&a.MmsConverter, "Whether to convert unsupported MMS media for carriers.")
	an.Describe(&a.SmartEncoding, "Whether to replace non-GSM characters to keep messages in cheaper GSM-7 encoding.")
	an.Describe(&a.FallbackToLongCode, "Whether a long code may be used when no short code or alphanumeric sender fits.")
	an.Describe(&a.AreaCodeGeomatch, "Whether to prefer a sender with the recipient's area code.")
	an.Describe(&a.SynchronousValidation, "Whether to validate sender and recipient before accepting a send request.")
	an.Describe(&a.UseInboundWebhookOnNumber, "Whether a sender's own inbound webhook takes precedence over inboundRequestUrl.")
	an.Describe(&a.ValidityPeriod, "Seconds a message may wait for delivery before expiring (1-36000).")
}

// MessagingServiceState is the persisted state of a MessagingService.
type MessagingServiceState struct {
	MessagingServiceArgs

	// Sid is the service SID (MG...) and the resource ID.
	Sid string `pulumi:"sid"`
}

func (s *MessagingServiceState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The Messaging Service SID (MG...).")
}

func (r *MessagingService) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio Messaging Service: a pool of senders with shared inbound and status webhooks. "+
		"Add senders with MessagingServicePhoneNumber.")
}

var _ infer.CustomResource[MessagingServiceArgs, MessagingServiceState] = (*MessagingService)(nil)
var _ infer.CustomCheck[MessagingServiceArgs] = (*MessagingService)(nil)
var _ infer.CustomRead[MessagingServiceArgs, MessagingServiceState] = (*MessagingService)(nil)
var _ infer.CustomUpdate[MessagingServiceArgs, MessagingServiceState] = (*MessagingService)(nil)
var _ infer.CustomDelete[MessagingServiceState] = (*MessagingService)(nil)

func (r *MessagingService) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[MessagingServiceArgs], error) {
	args, failures, err := infer.DefaultCheck[MessagingServiceArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[MessagingServiceArgs]{Inputs: args, Failures: failures}, err
	}
	if args.FriendlyName == "" || len(args.FriendlyName) > 64 {
		failures = append(failures, p.CheckFailure{Property: "friendlyName", Reason: "friendlyName must be 1-64 characters"})
	}
	failures = checkMethods(failures, map[string]*string{"inboundMethod": args.InboundMethod, "fallbackMethod": args.FallbackMethod})
	if s := deref(args.ScanMessageContent); args.ScanMessageContent != nil && s != "inherit" && s != "enable" && s != "disable" {
		failures = append(failures, p.CheckFailure{Property: "scanMessageContent", Reason: "scanMessageContent must be inherit, enable or disable"})
	}
	if v := args.ValidityPeriod; v != nil && (*v < 1 || *v > 36000) {
		failures = append(failures, p.CheckFailure{Property: "validityPeriod", Reason: "validityPeriod must be between 1 and 36000 seconds"})
	}
	return infer.CheckResponse[MessagingServiceArgs]{Inputs: args, Failures: failures}, nil
}

func serviceBase(sid string) string {
	if sid == "" {
		return "/v1/Services"
	}
	return "/v1/Services/" + url.PathEscape(sid)
}

func (r *MessagingService) Create(ctx context.Context, req infer.CreateRequest[MessagingServiceArgs]) (infer.CreateResponse[MessagingServiceState], error) {
	a := req.Inputs
	if req.DryRun {
		return infer.CreateResponse[MessagingServiceState]{ID: req.Name, Output: MessagingServiceState{MessagingServiceArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.CreateResponse[MessagingServiceState]{}, err
	}
	form := url.Values{"FriendlyName": {a.FriendlyName}}
	a.props().encode(form, nil)
	o, err := c.do(ctx, "POST", "messaging", serviceBase(""), nil, form)
	if err != nil {
		return infer.CreateResponse[MessagingServiceState]{}, fmt.Errorf("create messaging service: %w", err)
	}
	return infer.CreateResponse[MessagingServiceState]{ID: o.str("sid"), Output: MessagingServiceState{MessagingServiceArgs: a, Sid: o.str("sid")}}, nil
}

func (r *MessagingService) Read(ctx context.Context, req infer.ReadRequest[MessagingServiceArgs, MessagingServiceState]) (infer.ReadResponse[MessagingServiceArgs, MessagingServiceState], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.ReadResponse[MessagingServiceArgs, MessagingServiceState]{}, err
	}
	o, err := c.do(ctx, "GET", "messaging", serviceBase(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			return infer.ReadResponse[MessagingServiceArgs, MessagingServiceState]{}, nil
		}
		return infer.ReadResponse[MessagingServiceArgs, MessagingServiceState]{}, fmt.Errorf("read messaging service %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := in.FriendlyName == ""
	in.FriendlyName = o.str("friendly_name")
	in.props().refresh(o, imported)
	return infer.ReadResponse[MessagingServiceArgs, MessagingServiceState]{ID: req.ID, Inputs: in,
		State: MessagingServiceState{MessagingServiceArgs: in, Sid: req.ID}}, nil
}

func (r *MessagingService) Update(ctx context.Context, req infer.UpdateRequest[MessagingServiceArgs, MessagingServiceState]) (infer.UpdateResponse[MessagingServiceState], error) {
	a := req.Inputs
	st := req.State
	st.MessagingServiceArgs = a
	if req.DryRun {
		return infer.UpdateResponse[MessagingServiceState]{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return infer.UpdateResponse[MessagingServiceState]{}, err
	}
	form := url.Values{"FriendlyName": {a.FriendlyName}}
	prev := req.State.MessagingServiceArgs.props()
	a.props().encode(form, &prev)
	if _, err := c.do(ctx, "POST", "messaging", serviceBase(req.ID), nil, form); err != nil {
		return infer.UpdateResponse[MessagingServiceState]{}, fmt.Errorf("update messaging service %s: %w", req.ID, err)
	}
	return infer.UpdateResponse[MessagingServiceState]{Output: st}, nil
}

func (r *MessagingService) Delete(ctx context.Context, req infer.DeleteRequest[MessagingServiceState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", "messaging", serviceBase(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete messaging service %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
