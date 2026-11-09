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
	"net/url"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// ------------------------------------------------------------------ getEventTypes

// EventType is one event type in the Event Streams catalogue.
type EventType struct {
	Type             string `pulumi:"type"`
	SchemaId         string `pulumi:"schemaId"`
	Description      string `pulumi:"description"`
	Status           string `pulumi:"status"`
	DocumentationUrl string `pulumi:"documentationUrl"`
	DateCreated      string `pulumi:"dateCreated"`
	DateUpdated      string `pulumi:"dateUpdated"`
}

func (a *EventType) Annotate(an infer.Annotator) {
	an.Describe(&a.Type, "The event type, for example com.twilio.messaging.message.delivered. Use it in EventSubscription.types.")
	an.Describe(&a.SchemaId, "ID of the schema the event type follows. See getEventSchema.")
	an.Describe(&a.Description, "Human-readable description of the event type.")
	an.Describe(&a.Status, "Whether the event type can be used, for example available or deprecated.")
	an.Describe(&a.DocumentationUrl, "Link to the documentation or changelog for the event type.")
	an.Describe(&a.DateCreated, "When the event type was created (ISO 8601).")
	an.Describe(&a.DateUpdated, "When the event type was last updated (ISO 8601).")
}

func eventType(o obj) EventType {
	return EventType{
		Type: o.str("type"), SchemaId: o.str("schema_id"), Description: o.str("description"), Status: o.str("status"),
		DocumentationUrl: o.str("documentation_url"), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"),
	}
}

// GetEventTypes lists the event types Event Streams can deliver.
type GetEventTypes struct{}

// GetEventTypesArgs are the lookup arguments.
type GetEventTypesArgs struct {
	SchemaId *string `pulumi:"schemaId,optional"`
}

func (a *GetEventTypesArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.SchemaId, "Only event types that follow this schema.")
}

// GetEventTypesResult is the catalogue.
type GetEventTypesResult struct {
	Types []EventType `pulumi:"types"`
}

func (a *GetEventTypesResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Types, "The event types, in the order Twilio returns them.")
}

func (f *GetEventTypes) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the Event Streams event types, optionally those that follow one schema (GET /v1/Types on events.twilio.com).")
}

func (f *GetEventTypes) Invoke(ctx context.Context, req infer.FunctionRequest[GetEventTypesArgs]) (infer.FunctionResponse[GetEventTypesResult], error) {
	const fn = "getEventTypes"
	none := infer.FunctionResponse[GetEventTypesResult]{}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	q := url.Values{"PageSize": {"1000"}}
	if s := req.Input.SchemaId; s != nil && *s != "" {
		q.Set("SchemaId", *s)
	}
	out := []EventType{}
	err = listPages(ctx, c, fn, "events", "/v1/Types", q, "types", func(o obj) bool {
		out = append(out, eventType(o))
		return true
	})
	if err != nil {
		return none, lookupFailure(fn, "event types", err)
	}
	return infer.FunctionResponse[GetEventTypesResult]{Output: GetEventTypesResult{Types: out}}, nil
}

// GetEventType looks up one event type.
type GetEventType struct{}

// GetEventTypeArgs are the lookup arguments.
type GetEventTypeArgs struct {
	Type string `pulumi:"type"`
}

func (a *GetEventTypeArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Type, "The event type, for example com.twilio.messaging.message.delivered.")
}

func (f *GetEventType) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up one Event Streams event type (GET /v1/Types/{Type} on events.twilio.com).")
}

func (f *GetEventType) Invoke(ctx context.Context, req infer.FunctionRequest[GetEventTypeArgs]) (infer.FunctionResponse[EventType], error) {
	const fn = "getEventType"
	none := infer.FunctionResponse[EventType]{}
	if err := requireArg(fn, "type", req.Input.Type); err != nil {
		return none, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	o, err := c.do(ctx, "GET", "events", "/v1/Types/"+url.PathEscape(req.Input.Type), nil, nil)
	if err != nil {
		return none, lookupFailure(fn, "event type "+req.Input.Type, err)
	}
	return infer.FunctionResponse[EventType]{Output: eventType(o)}, nil
}

// ---------------------------------------------------------------- getEventSchema

// GetEventSchema looks up an Event Streams schema and its versions.
type GetEventSchema struct{}

// GetEventSchemaArgs are the lookup arguments.
type GetEventSchemaArgs struct {
	SchemaId string `pulumi:"schemaId"`
}

func (a *GetEventSchemaArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.SchemaId, "ID of the schema, as reported by an event type's schemaId.")
}

// EventSchemaVersion is one published version of a schema.
type EventSchemaVersion struct {
	SchemaVersion int    `pulumi:"schemaVersion"`
	DateCreated   string `pulumi:"dateCreated"`
	Raw           string `pulumi:"raw"`
}

func (a *EventSchemaVersion) Annotate(an infer.Annotator) {
	an.Describe(&a.SchemaVersion, "The version number.")
	an.Describe(&a.DateCreated, "When the version was published (ISO 8601).")
	an.Describe(&a.Raw, "URL of the raw JSON schema document for the version.")
}

// GetEventSchemaResult is the looked-up schema.
type GetEventSchemaResult struct {
	SchemaId                 string               `pulumi:"schemaId"`
	LatestVersion            int                  `pulumi:"latestVersion"`
	LatestVersionDateCreated string               `pulumi:"latestVersionDateCreated"`
	Versions                 []EventSchemaVersion `pulumi:"versions"`
}

func (a *GetEventSchemaResult) Annotate(an infer.Annotator) {
	an.Describe(&a.SchemaId, "ID of the schema.")
	an.Describe(&a.LatestVersion, "The latest published version number.")
	an.Describe(&a.LatestVersionDateCreated, "When the latest version was published (ISO 8601).")
	an.Describe(&a.Versions, "Every published version, in the order Twilio returns them.")
}

func (f *GetEventSchema) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up an Event Streams schema and its versions (GET /v1/Schemas/{Id} and /v1/Schemas/{Id}/Versions on events.twilio.com).")
}

func (f *GetEventSchema) Invoke(ctx context.Context, req infer.FunctionRequest[GetEventSchemaArgs]) (infer.FunctionResponse[GetEventSchemaResult], error) {
	const fn = "getEventSchema"
	none := infer.FunctionResponse[GetEventSchemaResult]{}
	id := req.Input.SchemaId
	if err := requireArg(fn, "schemaId", id); err != nil {
		return none, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	base := "/v1/Schemas/" + url.PathEscape(id)
	o, err := c.do(ctx, "GET", "events", base, nil, nil)
	if err != nil {
		return none, lookupFailure(fn, "event schema "+id, err)
	}
	versions := []EventSchemaVersion{}
	err = listPages(ctx, c, fn, "events", base+"/Versions", url.Values{"PageSize": {"1000"}}, "schema_versions", func(v obj) bool {
		n := 0
		if p := v.optInt("schema_version"); p != nil {
			n = *p
		}
		versions = append(versions, EventSchemaVersion{SchemaVersion: n, DateCreated: v.str("date_created"), Raw: v.str("raw")})
		return true
	})
	if err != nil {
		return none, lookupFailure(fn, "versions of event schema "+id, err)
	}
	latest := 0
	if p := o.optInt("latest_version"); p != nil {
		latest = *p
	}
	return infer.FunctionResponse[GetEventSchemaResult]{Output: GetEventSchemaResult{
		SchemaId: o.str("id"), LatestVersion: latest, LatestVersionDateCreated: o.str("latest_version_date_created"), Versions: versions,
	}}, nil
}
