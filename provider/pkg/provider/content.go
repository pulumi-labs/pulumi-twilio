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
	"strconv"
	"strings"
	"unicode"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// The Content API lives on content.twilio.com (/v1/Content). Unlike most of this provider it takes
// and returns JSON bodies, so it goes through doJSON. A content template is account-scoped by the
// credentials; nothing in its path names an account.
//
// The shapes below follow twilio-oai spec/json/twilio_content_v1.json (version 1.1.0) read
// 2026-11-12 and have not been checked against a live account. The WhatsApp approval submission
// (POST and PATCH /v1/Content/{Sid}/ApprovalRequests/whatsapp) is not a resource here, see
// research/coverage/twilio.md.

const contentProduct = "content"

func contentPath(sid string) string {
	path := "/v1/Content"
	if sid != "" {
		path += "/" + url.PathEscape(sid)
	}
	return path
}

// ------------------------------------------------------------- action and item types

// ContentListItem is one option of a twilio/list-picker.
type ContentListItem struct {
	Id          string  `pulumi:"id"`
	Item        string  `pulumi:"item"`
	Description *string `pulumi:"description,optional"`
}

func (a *ContentListItem) Annotate(an infer.Annotator) {
	an.Describe(&a.Id, "Identifier of the option, returned to you when the recipient picks it.")
	an.Describe(&a.Item, "Label of the option.")
	an.Describe(&a.Description, "Secondary text shown under the label.")
}

// ContentCallToActionAction is one button of a twilio/call-to-action.
type ContentCallToActionAction struct {
	Type  string  `pulumi:"type"`
	Title *string `pulumi:"title,optional"`
	Url   *string `pulumi:"url,optional"`
	Phone *string `pulumi:"phone,optional"`
	Code  *string `pulumi:"code,optional"`
	Id    *string `pulumi:"id,optional"`
}

func (a *ContentCallToActionAction) Annotate(an infer.Annotator) {
	an.Describe(&a.Type, "The button type: URL, PHONE_NUMBER, COPY_CODE, VOICE_CALL, VOICE_CALL_REQUEST or REQUEST_CONTACT_INFO.")
	an.Describe(&a.Title, "Label of the button.")
	an.Describe(&a.Url, "URL the button opens (type URL).")
	an.Describe(&a.Phone, "Phone number the button dials (type PHONE_NUMBER or VOICE_CALL).")
	an.Describe(&a.Code, "Code the button copies (type COPY_CODE).")
	an.Describe(&a.Id, "Identifier of the button.")
}

// ContentQuickReplyAction is one button of a twilio/quick-reply.
type ContentQuickReplyAction struct {
	Type  *string `pulumi:"type,optional"`
	Title string  `pulumi:"title"`
	Id    *string `pulumi:"id,optional"`
}

func (a *ContentQuickReplyAction) Annotate(an infer.Annotator) {
	an.Describe(&a.Type, "The button type. Only QUICK_REPLY exists.")
	an.Describe(&a.Title, "Label of the button.")
	an.Describe(&a.Id, "Identifier returned to you when the recipient taps the button.")
}

// ContentCardAction is one button of a twilio/card or whatsapp/card.
type ContentCardAction struct {
	Type          string  `pulumi:"type"`
	Title         *string `pulumi:"title,optional"`
	Url           *string `pulumi:"url,optional"`
	Phone         *string `pulumi:"phone,optional"`
	Id            *string `pulumi:"id,optional"`
	Code          *string `pulumi:"code,optional"`
	WebviewSize   *string `pulumi:"webviewSize,optional"`
	DigitableLine *string `pulumi:"digitableLine,optional"`
}

func (a *ContentCardAction) Annotate(an infer.Annotator) {
	an.Describe(&a.Type, "The button type: URL, PHONE_NUMBER, QUICK_REPLY, COPY_CODE, VOICE_CALL, PIX, BOLETO or PAYMENT_LINK.")
	an.Describe(&a.Title, "Label of the button.")
	an.Describe(&a.Url, "URL the button opens (type URL).")
	an.Describe(&a.Phone, "Phone number the button dials (type PHONE_NUMBER or VOICE_CALL).")
	an.Describe(&a.Id, "Identifier of the button.")
	an.Describe(&a.Code, "Code the button copies (type COPY_CODE).")
	an.Describe(&a.WebviewSize, "Size of the in-app web view a URL button opens: TALL, FULL, HALF or NONE.")
	an.Describe(&a.DigitableLine, "Digitable line of a BOLETO payment.")
}

// ContentCarouselAction is one button of a twilio/carousel card.
type ContentCarouselAction struct {
	Type  string  `pulumi:"type"`
	Title string  `pulumi:"title"`
	Url   *string `pulumi:"url,optional"`
	Phone *string `pulumi:"phone,optional"`
	Id    *string `pulumi:"id,optional"`
}

func (a *ContentCarouselAction) Annotate(an infer.Annotator) {
	an.Describe(&a.Type, "The button type: URL, PHONE_NUMBER or QUICK_REPLY.")
	an.Describe(&a.Title, "Label of the button.")
	an.Describe(&a.Url, "URL the button opens (type URL).")
	an.Describe(&a.Phone, "Phone number the button dials (type PHONE_NUMBER).")
	an.Describe(&a.Id, "Identifier returned to you when the recipient taps the button (type QUICK_REPLY).")
}

// ContentAuthenticationAction is the one-time-password button of a whatsapp/authentication template.
type ContentAuthenticationAction struct {
	Type         string `pulumi:"type"`
	CopyCodeText string `pulumi:"copyCodeText"`
}

func (a *ContentAuthenticationAction) Annotate(an infer.Annotator) {
	an.Describe(&a.Type, "The button type. Only COPY_CODE exists.")
	an.Describe(&a.CopyCodeText, "Label of the copy-code button.")
}

// ContentCatalogItem is one product of a twilio/catalog.
type ContentCatalogItem struct {
	Id           *string  `pulumi:"id,optional"`
	SectionTitle *string  `pulumi:"sectionTitle,optional"`
	Name         *string  `pulumi:"name,optional"`
	MediaUrl     *string  `pulumi:"mediaUrl,optional"`
	Price        *float64 `pulumi:"price,optional"`
	Description  *string  `pulumi:"description,optional"`
}

func (a *ContentCatalogItem) Annotate(an infer.Annotator) {
	an.Describe(&a.Id, "Identifier of the product in the catalog.")
	an.Describe(&a.SectionTitle, "Title of the section the product is listed in.")
	an.Describe(&a.Name, "Name of the product.")
	an.Describe(&a.MediaUrl, "URL of the product image.")
	an.Describe(&a.Price, "Price of the product.")
	an.Describe(&a.Description, "Description of the product.")
}

// ContentCarouselCard is one card of a twilio/carousel.
type ContentCarouselCard struct {
	Title   *string                 `pulumi:"title,optional"`
	Body    string                  `pulumi:"body"`
	Media   string                  `pulumi:"media"`
	Actions []ContentCarouselAction `pulumi:"actions"`
}

func (a *ContentCarouselCard) Annotate(an infer.Annotator) {
	an.Describe(&a.Title, "Title of the card.")
	an.Describe(&a.Body, "Text of the card.")
	an.Describe(&a.Media, "URL of the card's media.")
	an.Describe(&a.Actions, "Buttons of the card.")
}

// ContentFlowsPageComponent is one layout component of a twilio/flows page.
type ContentFlowsPageComponent struct {
	Label *string `pulumi:"label,optional"`
	Type  string  `pulumi:"type"`
}

func (a *ContentFlowsPageComponent) Annotate(an infer.Annotator) {
	an.Describe(&a.Label, "Label of the component.")
	an.Describe(&a.Type, "The component type.")
}

// ContentFlowsPage is one page of a twilio/flows template.
type ContentFlowsPage struct {
	Id         string                      `pulumi:"id"`
	NextPageId *string                     `pulumi:"nextPageId,optional"`
	Title      string                      `pulumi:"title"`
	Subtitle   *string                     `pulumi:"subtitle,optional"`
	Layout     []ContentFlowsPageComponent `pulumi:"layout"`
}

func (a *ContentFlowsPage) Annotate(an infer.Annotator) {
	an.Describe(&a.Id, "Identifier of the page.")
	an.Describe(&a.NextPageId, "Identifier of the page that follows this one.")
	an.Describe(&a.Title, "Title of the page.")
	an.Describe(&a.Subtitle, "Subtitle of the page.")
	an.Describe(&a.Layout, "Components of the page.")
}

// ------------------------------------------------------------------ the content types

// ContentText is the twilio/text type: plain text.
type ContentText struct {
	Body string `pulumi:"body"`
}

func (a *ContentText) Annotate(an infer.Annotator) {
	an.Describe(&a.Body, "The text of the message.")
}

// ContentMedia is the twilio/media type: file attachments, or long text sent by MMS.
type ContentMedia struct {
	Body  *string  `pulumi:"body,optional"`
	Media []string `pulumi:"media"`
}

func (a *ContentMedia) Annotate(an infer.Annotator) {
	an.Describe(&a.Body, "Text sent with the media.")
	an.Describe(&a.Media, "URLs of the media to send.")
}

// ContentLocation is the twilio/location type: a location pin with an optional label.
type ContentLocation struct {
	Latitude  float64 `pulumi:"latitude"`
	Longitude float64 `pulumi:"longitude"`
	Label     *string `pulumi:"label,optional"`
	Id        *string `pulumi:"id,optional"`
	Address   *string `pulumi:"address,optional"`
}

func (a *ContentLocation) Annotate(an infer.Annotator) {
	an.Describe(&a.Latitude, "Latitude of the pin.")
	an.Describe(&a.Longitude, "Longitude of the pin.")
	an.Describe(&a.Label, "Label shown with the pin.")
	an.Describe(&a.Id, "Identifier of the location.")
	an.Describe(&a.Address, "Address shown with the pin.")
}

// ContentListPicker is the twilio/list-picker type: a menu of up to 10 options.
type ContentListPicker struct {
	Body   string            `pulumi:"body"`
	Button string            `pulumi:"button"`
	Items  []ContentListItem `pulumi:"items"`
}

func (a *ContentListPicker) Annotate(an infer.Annotator) {
	an.Describe(&a.Body, "The text of the message.")
	an.Describe(&a.Button, "Label of the button that opens the menu.")
	an.Describe(&a.Items, "The options of the menu.")
}

// ContentCallToAction is the twilio/call-to-action type: buttons that open a URL or dial a number.
type ContentCallToAction struct {
	Body    string                      `pulumi:"body"`
	Actions []ContentCallToActionAction `pulumi:"actions"`
}

func (a *ContentCallToAction) Annotate(an infer.Annotator) {
	an.Describe(&a.Body, "The text of the message.")
	an.Describe(&a.Actions, "The buttons.")
}

// ContentQuickReply is the twilio/quick-reply type: buttons the recipient taps to answer.
type ContentQuickReply struct {
	Body    string                    `pulumi:"body"`
	Actions []ContentQuickReplyAction `pulumi:"actions"`
}

func (a *ContentQuickReply) Annotate(an infer.Annotator) {
	an.Describe(&a.Body, "The text of the message.")
	an.Describe(&a.Actions, "The reply buttons.")
}

// ContentCard is the twilio/card type: a structured message with media and buttons.
type ContentCard struct {
	Title    *string             `pulumi:"title,optional"`
	Subtitle *string             `pulumi:"subtitle,optional"`
	Media    []string            `pulumi:"media,optional"`
	Actions  []ContentCardAction `pulumi:"actions,optional"`
}

func (a *ContentCard) Annotate(an infer.Annotator) {
	an.Describe(&a.Title, "Title of the card.")
	an.Describe(&a.Subtitle, "Subtitle of the card.")
	an.Describe(&a.Media, "URLs of the card's media.")
	an.Describe(&a.Actions, "Buttons of the card.")
}

// ContentCatalog is the twilio/catalog type: a list of catalog products.
type ContentCatalog struct {
	Title        *string              `pulumi:"title,optional"`
	Body         string               `pulumi:"body"`
	Subtitle     *string              `pulumi:"subtitle,optional"`
	Id           *string              `pulumi:"id,optional"`
	Items        []ContentCatalogItem `pulumi:"items,optional"`
	DynamicItems *string              `pulumi:"dynamicItems,optional"`
}

func (a *ContentCatalog) Annotate(an infer.Annotator) {
	an.Describe(&a.Title, "Title of the message.")
	an.Describe(&a.Body, "The text of the message.")
	an.Describe(&a.Subtitle, "Subtitle of the message.")
	an.Describe(&a.Id, "Identifier of the catalog.")
	an.Describe(&a.Items, "The products to show.")
	an.Describe(&a.DynamicItems, "A variable placeholder that Twilio fills with the products when the message is sent.")
}

// ContentCarousel is the twilio/carousel type: a message with up to 10 cards.
type ContentCarousel struct {
	Body  string                `pulumi:"body"`
	Cards []ContentCarouselCard `pulumi:"cards"`
}

func (a *ContentCarousel) Annotate(an infer.Annotator) {
	an.Describe(&a.Body, "The text of the message.")
	an.Describe(&a.Cards, "The cards.")
}

// ContentFlows is the twilio/flows type: a multi-page flow.
type ContentFlows struct {
	Body       string             `pulumi:"body"`
	ButtonText string             `pulumi:"buttonText"`
	Subtitle   *string            `pulumi:"subtitle,optional"`
	MediaUrl   *string            `pulumi:"mediaUrl,optional"`
	Pages      []ContentFlowsPage `pulumi:"pages"`
	Type       string             `pulumi:"type"`
}

func (a *ContentFlows) Annotate(an infer.Annotator) {
	an.Describe(&a.Body, "The text of the message.")
	an.Describe(&a.ButtonText, "Label of the button that opens the flow.")
	an.Describe(&a.Subtitle, "Subtitle of the message.")
	an.Describe(&a.MediaUrl, "URL of the media shown with the message.")
	an.Describe(&a.Pages, "The pages of the flow.")
	an.Describe(&a.Type, "The flow type.")
}

// ContentSchedule is the twilio/schedule type: a message offering time slots.
type ContentSchedule struct {
	Id        string `pulumi:"id"`
	Title     string `pulumi:"title"`
	TimeSlots string `pulumi:"timeSlots"`
}

func (a *ContentSchedule) Annotate(an infer.Annotator) {
	an.Describe(&a.Id, "Identifier of the schedule.")
	an.Describe(&a.Title, "Title of the message.")
	an.Describe(&a.TimeSlots, "The time slots, as Twilio's time_slots string (usually a variable placeholder filled when the message is sent).")
}

// ContentWhatsappCard is the whatsapp/card type: a WhatsApp card with header, footer and buttons.
type ContentWhatsappCard struct {
	Body       string              `pulumi:"body"`
	Footer     *string             `pulumi:"footer,optional"`
	Media      []string            `pulumi:"media,optional"`
	HeaderText *string             `pulumi:"headerText,optional"`
	Actions    []ContentCardAction `pulumi:"actions,optional"`
}

func (a *ContentWhatsappCard) Annotate(an infer.Annotator) {
	an.Describe(&a.Body, "The text of the message.")
	an.Describe(&a.Footer, "Footer text.")
	an.Describe(&a.Media, "URLs of the card's media.")
	an.Describe(&a.HeaderText, "Header text.")
	an.Describe(&a.Actions, "Buttons of the card.")
}

// ContentWhatsappAuthentication is the whatsapp/authentication type: a one-time-password message.
type ContentWhatsappAuthentication struct {
	AddSecurityRecommendation *bool                         `pulumi:"addSecurityRecommendation,optional"`
	CodeExpirationMinutes     *float64                      `pulumi:"codeExpirationMinutes,optional"`
	Actions                   []ContentAuthenticationAction `pulumi:"actions"`
}

func (a *ContentWhatsappAuthentication) Annotate(an infer.Annotator) {
	an.Describe(&a.AddSecurityRecommendation, "Whether WhatsApp adds its security recommendation to the message.")
	an.Describe(&a.CodeExpirationMinutes, "Minutes until the code expires.")
	an.Describe(&a.Actions, "The copy-code button.")
}

// ContentWhatsappFlows is the whatsapp/flows type: a message that opens a WhatsApp Flow.
type ContentWhatsappFlows struct {
	Body                    string  `pulumi:"body"`
	ButtonText              string  `pulumi:"buttonText"`
	Subtitle                *string `pulumi:"subtitle,optional"`
	MediaUrl                *string `pulumi:"mediaUrl,optional"`
	FlowId                  string  `pulumi:"flowId"`
	FlowToken               *string `pulumi:"flowToken,optional"`
	FlowFirstPageId         *string `pulumi:"flowFirstPageId,optional"`
	IsFlowFirstPageEndpoint *bool   `pulumi:"isFlowFirstPageEndpoint,optional"`
}

func (a *ContentWhatsappFlows) Annotate(an infer.Annotator) {
	an.Describe(&a.Body, "The text of the message.")
	an.Describe(&a.ButtonText, "Label of the button that opens the flow.")
	an.Describe(&a.Subtitle, "Subtitle of the message.")
	an.Describe(&a.MediaUrl, "URL of the media shown with the message.")
	an.Describe(&a.FlowId, "ID of the WhatsApp Flow to open.")
	an.Describe(&a.FlowToken, "Token passed to the flow.")
	an.Describe(&a.FlowFirstPageId, "ID of the first page of the flow.")
	an.Describe(&a.IsFlowFirstPageEndpoint, "Whether the first page of the flow is served by an endpoint.")
}

// ContentTypes holds the content types of a template. A template carries one or more; Twilio
// chooses the one that fits the channel when a message is sent. The Twilio keys are the slash
// names (twilio/text, whatsapp/card, ...), kept in the JSON tags.
type ContentTypes struct {
	Text                   *ContentText                   `pulumi:"text,optional"`
	Media                  *ContentMedia                  `pulumi:"media,optional"`
	Location               *ContentLocation               `pulumi:"location,optional"`
	ListPicker             *ContentListPicker             `pulumi:"listPicker,optional"`
	CallToAction           *ContentCallToAction           `pulumi:"callToAction,optional"`
	QuickReply             *ContentQuickReply             `pulumi:"quickReply,optional"`
	Card                   *ContentCard                   `pulumi:"card,optional"`
	Catalog                *ContentCatalog                `pulumi:"catalog,optional"`
	Carousel               *ContentCarousel               `pulumi:"carousel,optional"`
	Flows                  *ContentFlows                  `pulumi:"flows,optional"`
	Schedule               *ContentSchedule               `pulumi:"schedule,optional"`
	WhatsappCard           *ContentWhatsappCard           `pulumi:"whatsappCard,optional"`
	WhatsappAuthentication *ContentWhatsappAuthentication `pulumi:"whatsappAuthentication,optional"`
	WhatsappFlows          *ContentWhatsappFlows          `pulumi:"whatsappFlows,optional"`
}

func (a *ContentTypes) Annotate(an infer.Annotator) {
	an.Describe(&a.Text, "twilio/text: plain text.")
	an.Describe(&a.Media, "twilio/media: file attachments, or long text sent by MMS in the US and Canada.")
	an.Describe(&a.Location, "twilio/location: a location pin with an optional label.")
	an.Describe(&a.ListPicker, "twilio/list-picker: a menu of up to 10 options.")
	an.Describe(&a.CallToAction, "twilio/call-to-action: buttons that open a URL or dial a number.")
	an.Describe(&a.QuickReply, "twilio/quick-reply: buttons the recipient taps to answer.")
	an.Describe(&a.Card, "twilio/card: a structured message with media and buttons.")
	an.Describe(&a.Catalog, "twilio/catalog: a list of catalog products.")
	an.Describe(&a.Carousel, "twilio/carousel: a message with up to 10 cards.")
	an.Describe(&a.Flows, "twilio/flows: a multi-page flow.")
	an.Describe(&a.Schedule, "twilio/schedule: a message offering time slots.")
	an.Describe(&a.WhatsappCard, "whatsapp/card: a WhatsApp card with header, footer and buttons.")
	an.Describe(&a.WhatsappAuthentication, "whatsapp/authentication: a WhatsApp one-time-password message.")
	an.Describe(&a.WhatsappFlows, "whatsapp/flows: a message that opens a WhatsApp Flow.")
}

// count is the number of content types set.
func (t ContentTypes) count() int {
	n := 0
	for _, set := range []bool{t.Text != nil, t.Media != nil, t.Location != nil, t.ListPicker != nil, t.CallToAction != nil,
		t.QuickReply != nil, t.Card != nil, t.Catalog != nil, t.Carousel != nil, t.Flows != nil, t.Schedule != nil,
		t.WhatsappCard != nil, t.WhatsappAuthentication != nil, t.WhatsappFlows != nil} {
		if set {
			n++
		}
	}
	return n
}

// ------------------------------------------------------------------- the wire format
//
// Twilio's JSON uses snake_case names, and slash names (twilio/text) for the content types. The
// provider's own property names are camelCase. The nested types carry only pulumi tags, because the
// framework would otherwise write both spellings into the state; these functions translate by
// reflection instead, so the two sides cannot drift apart.

// contentTypeKeys maps a ContentTypes property to its Twilio key.
var contentTypeKeys = map[string]string{
	"text": "twilio/text", "media": "twilio/media", "location": "twilio/location", "listPicker": "twilio/list-picker",
	"callToAction": "twilio/call-to-action", "quickReply": "twilio/quick-reply", "card": "twilio/card",
	"catalog": "twilio/catalog", "carousel": "twilio/carousel", "flows": "twilio/flows", "schedule": "twilio/schedule",
	"whatsappCard": "whatsapp/card", "whatsappAuthentication": "whatsapp/authentication", "whatsappFlows": "whatsapp/flows",
}

// snakeCase turns buttonText into button_text.
func snakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// camelCase turns button_text into buttonText.
func camelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// toWire converts a value built from pulumi-tagged structs to the JSON shape Twilio takes: unset
// optional values are left out and property names become Twilio's.
func toWire(v reflect.Value) any {
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		return toWire(v.Elem())
	case reflect.Struct:
		out := map[string]any{}
		isTypes := v.Type() == reflect.TypeOf(ContentTypes{})
		for i := 0; i < v.NumField(); i++ {
			name, _, _ := strings.Cut(v.Type().Field(i).Tag.Get("pulumi"), ",")
			f := v.Field(i)
			if f.Kind() == reflect.Ptr && f.IsNil() {
				continue
			}
			if (f.Kind() == reflect.Slice || f.Kind() == reflect.Map) && f.IsNil() && strings.Contains(v.Type().Field(i).Tag.Get("pulumi"), "optional") {
				continue
			}
			key := snakeCase(name)
			if isTypes {
				key = contentTypeKeys[name]
			}
			out[key] = toWire(f)
		}
		return out
	case reflect.Slice:
		out := make([]any, v.Len())
		for i := range out {
			out[i] = toWire(v.Index(i))
		}
		return out
	}
	return v.Interface()
}

// fromWireKeys renames the keys of a decoded Twilio object to the provider's property names, all
// the way down, so that it decodes into the pulumi-tagged structs. The top level of the content types
// has the slash names to undo.
func fromWireKeys(v any, types bool) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			if types {
				for prop, slash := range contentTypeKeys {
					if slash == k {
						out[prop] = fromWireKeys(val, false)
					}
				}
				continue
			}
			out[camelCase(k)] = fromWireKeys(val, false)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = fromWireKeys(x[i], false)
		}
		return out
	}
	return v
}

// ------------------------------------------------------------------ ContentTemplate

// ContentTemplate manages a Twilio Content template: a reusable, channel-independent message
// definition (text, media, buttons, cards, lists, WhatsApp authentication and more) that
// messages refer to by its SID (HX...).
type ContentTemplate struct{}

// ContentTemplateArgs are the inputs of a ContentTemplate. Every input updates in place; the whole
// template is sent on each update.
type ContentTemplateArgs struct {
	FriendlyName *string           `pulumi:"friendlyName,optional"`
	Variables    map[string]string `pulumi:"variables,optional"`
	Language     string            `pulumi:"language"`
	Types        ContentTypes      `pulumi:"types"`
}

func (a *ContentTemplateArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.FriendlyName, "Name of the template, not shown to the recipient. Removing it leaves the name as it is on Twilio.")
	an.Describe(&a.Variables, "Default values of the placeholders used in the template, keyed by variable name (for example \"1\" for {{1}}). Removing all of them leaves the defaults as they are on Twilio.")
	an.Describe(&a.Language, "Language code of the template, for example en.")
	an.Describe(&a.Types, "The content types of the template. Set at least one.")
}

// ContentTemplateState is the persisted state of a ContentTemplate.
type ContentTemplateState struct {
	ContentTemplateArgs

	Sid         string `pulumi:"sid"`
	AccountSid  string `pulumi:"accountSid"`
	DateCreated string `pulumi:"dateCreated"`
	DateUpdated string `pulumi:"dateUpdated"`
	Url         string `pulumi:"url"`
}

func (s *ContentTemplateState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The template SID (HX...).")
	a.Describe(&s.AccountSid, "SID of the account that owns the template.")
	a.Describe(&s.DateCreated, "When the template was created, as Twilio reports it.")
	a.Describe(&s.DateUpdated, "When the template was last updated, as Twilio reports it.")
	a.Describe(&s.Url, "The URL of the template, relative to https://content.twilio.com.")
}

func (r *ContentTemplate) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Twilio Content template. The resource ID is the template SID (HX...). Optional settings that are removed leave the setting as it is on Twilio. "+
		"Submitting the template to WhatsApp for approval is not managed here; read the result with the getContentApprovalStatus function.")
}

var _ infer.CustomResource[ContentTemplateArgs, ContentTemplateState] = (*ContentTemplate)(nil)
var _ infer.CustomCheck[ContentTemplateArgs] = (*ContentTemplate)(nil)
var _ infer.CustomRead[ContentTemplateArgs, ContentTemplateState] = (*ContentTemplate)(nil)
var _ infer.CustomUpdate[ContentTemplateArgs, ContentTemplateState] = (*ContentTemplate)(nil)
var _ infer.CustomDelete[ContentTemplateState] = (*ContentTemplate)(nil)

var (
	contentCallToActionTypes = []string{"URL", "PHONE_NUMBER", "COPY_CODE", "VOICE_CALL", "VOICE_CALL_REQUEST", "REQUEST_CONTACT_INFO"}
	contentCardActionTypes   = []string{"URL", "PHONE_NUMBER", "QUICK_REPLY", "COPY_CODE", "VOICE_CALL", "PIX", "BOLETO", "PAYMENT_LINK"}
	contentCarouselTypes     = []string{"URL", "PHONE_NUMBER", "QUICK_REPLY"}
	contentWebviewSizes      = []string{"TALL", "FULL", "HALF", "NONE"}
)

func (r *ContentTemplate) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ContentTemplateArgs], error) {
	args, failures, err := infer.DefaultCheck[ContentTemplateArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[ContentTemplateArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.required("language", args.Language)
	if !c.unk("types") {
		if args.Types.count() == 0 {
			c.fail("types", "types must set at least one content type")
		}
		checkContentTypes(c, args.Types)
	}
	if args.FriendlyName != nil && !c.unk("friendlyName") && *args.FriendlyName == "" {
		c.fail("friendlyName", "friendlyName must not be empty")
	}
	return infer.CheckResponse[ContentTemplateArgs]{Inputs: args, Failures: c.failures}, nil
}

// checkContentTypes validates the enumerated action and webview values of every content type.
func checkContentTypes(c *checker, t ContentTypes) {
	enum := func(prop, v string, allowed []string) {
		c.failures = oneOf(c.failures, prop, &v, allowed)
	}
	for i, a := range orEmpty(t.CallToAction, func(x *ContentCallToAction) []ContentCallToActionAction { return x.Actions }) {
		enum(fmt.Sprintf("types.callToAction.actions[%d].type", i), a.Type, contentCallToActionTypes)
	}
	for i, a := range orEmpty(t.Card, func(x *ContentCard) []ContentCardAction { return x.Actions }) {
		checkContentCardAction(enum, fmt.Sprintf("types.card.actions[%d]", i), a)
	}
	for i, a := range orEmpty(t.WhatsappCard, func(x *ContentWhatsappCard) []ContentCardAction { return x.Actions }) {
		checkContentCardAction(enum, fmt.Sprintf("types.whatsappCard.actions[%d]", i), a)
	}
	if t.Carousel != nil {
		for i, card := range t.Carousel.Cards {
			for j, a := range card.Actions {
				enum(fmt.Sprintf("types.carousel.cards[%d].actions[%d].type", i, j), a.Type, contentCarouselTypes)
			}
		}
	}
	if t.Media != nil && len(t.Media.Media) == 0 {
		c.fail("types.media.media", "types.media.media must list at least one URL")
	}
	if len(orEmpty(t.WhatsappAuthentication, func(x *ContentWhatsappAuthentication) []ContentAuthenticationAction { return x.Actions })) == 0 && t.WhatsappAuthentication != nil {
		c.fail("types.whatsappAuthentication.actions", "types.whatsappAuthentication.actions must list at least one action")
	}
	for i, a := range orEmpty(t.WhatsappAuthentication, func(x *ContentWhatsappAuthentication) []ContentAuthenticationAction { return x.Actions }) {
		enum(fmt.Sprintf("types.whatsappAuthentication.actions[%d].type", i), a.Type, []string{"COPY_CODE"})
	}
	for i, a := range orEmpty(t.QuickReply, func(x *ContentQuickReply) []ContentQuickReplyAction { return x.Actions }) {
		if a.Type != nil {
			enum(fmt.Sprintf("types.quickReply.actions[%d].type", i), *a.Type, []string{"QUICK_REPLY"})
		}
	}
}

func checkContentCardAction(enum func(string, string, []string), prop string, a ContentCardAction) {
	enum(prop+".type", a.Type, contentCardActionTypes)
	if a.WebviewSize != nil {
		enum(prop+".webviewSize", *a.WebviewSize, contentWebviewSizes)
	}
}

// orEmpty returns pick(v), or nil when v is nil.
func orEmpty[T, E any](v *T, pick func(*T) []E) []E {
	if v == nil {
		return nil
	}
	return pick(v)
}

// body is the JSON request body of a create or an update.
func (a ContentTemplateArgs) body() map[string]any {
	b := map[string]any{"language": a.Language, "types": toWire(reflect.ValueOf(a.Types))}
	if a.FriendlyName != nil {
		b["friendly_name"] = *a.FriendlyName
	}
	if len(a.Variables) > 0 {
		b["variables"] = a.Variables
	}
	return b
}

// contentResponse is a decoded Content response.
type contentResponse struct {
	Sid          string
	AccountSid   string
	FriendlyName *string
	Language     string
	Variables    map[string]string
	Types        ContentTypes
	DateCreated  string
	DateUpdated  string
	Url          string
}

func decodeContent(o obj) (contentResponse, error) {
	r := contentResponse{Sid: o.str("sid"), AccountSid: o.str("account_sid"), FriendlyName: nonEmpty(o.optStr("friendly_name")),
		Language: o.str("language"), DateCreated: o.str("date_created"), DateUpdated: o.str("date_updated"), Url: o.str("url")}
	if vars, ok := o["variables"].(map[string]any); ok && len(vars) > 0 {
		r.Variables = map[string]string{}
		for k, v := range vars {
			switch x := v.(type) {
			case string:
				r.Variables[k] = x
			case float64:
				r.Variables[k] = strconv.FormatFloat(x, 'f', -1, 64)
			default:
				raw, _ := json.Marshal(x)
				r.Variables[k] = string(raw)
			}
		}
	}
	if t, ok := o["types"]; ok && t != nil {
		raw, err := json.Marshal(fromWireKeys(t, true))
		if err != nil {
			return r, err
		}
		if err := json.Unmarshal(raw, &r.Types); err != nil {
			return r, fmt.Errorf("twilio: decode content types: %w", err)
		}
	}
	return r, nil
}

func contentStateFrom(in ContentTemplateArgs, r contentResponse) ContentTemplateState {
	return ContentTemplateState{ContentTemplateArgs: in, Sid: r.Sid, AccountSid: r.AccountSid,
		DateCreated: r.DateCreated, DateUpdated: r.DateUpdated, Url: r.Url}
}

func (r *ContentTemplate) Create(ctx context.Context, req infer.CreateRequest[ContentTemplateArgs]) (infer.CreateResponse[ContentTemplateState], error) {
	type resp = infer.CreateResponse[ContentTemplateState]
	a := req.Inputs
	if req.DryRun {
		return resp{Output: ContentTemplateState{ContentTemplateArgs: a}}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.doJSON(ctx, "POST", contentProduct, contentPath(""), nil, a.body())
	if err != nil {
		return resp{}, fmt.Errorf("create content template: %w", err)
	}
	got, err := decodeContent(o)
	if err != nil {
		// Twilio created the template, but its answer is unreadable: record it so the next
		// update or refresh can pick it up instead of leaking a template.
		if sid := o.str("sid"); sid != "" {
			return resp{ID: sid, Output: ContentTemplateState{ContentTemplateArgs: a, Sid: sid}},
				infer.ResourceInitFailedError{Reasons: []string{fmt.Sprintf("read back content template %s: %v", sid, err)}}
		}
		return resp{}, err
	}
	if got.Sid == "" {
		return resp{}, fmt.Errorf("create content template: Twilio returned no sid")
	}
	return resp{ID: got.Sid, Output: contentStateFrom(a, got)}, nil
}

func (r *ContentTemplate) Read(ctx context.Context, req infer.ReadRequest[ContentTemplateArgs, ContentTemplateState]) (infer.ReadResponse[ContentTemplateArgs, ContentTemplateState], error) {
	type resp = infer.ReadResponse[ContentTemplateArgs, ContentTemplateState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.doJSON(ctx, "GET", contentProduct, contentPath(req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("content template", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read content template %s: %w", req.ID, err)
	}
	got, err := decodeContent(o)
	if err != nil {
		return resp{}, err
	}
	in := req.Inputs
	// An import arrives with no prior state; the program's own settings are the only record of what it manages.
	imported := req.State.Sid == ""
	in.Language = got.Language
	in.Types = got.Types
	if imported || in.FriendlyName != nil {
		in.FriendlyName = got.FriendlyName
	}
	if imported || len(in.Variables) > 0 {
		in.Variables = got.Variables
	}
	return resp{ID: req.ID, Inputs: in, State: contentStateFrom(in, got)}, nil
}

func (r *ContentTemplate) Update(ctx context.Context, req infer.UpdateRequest[ContentTemplateArgs, ContentTemplateState]) (infer.UpdateResponse[ContentTemplateState], error) {
	type resp = infer.UpdateResponse[ContentTemplateState]
	st := req.State
	st.ContentTemplateArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.doJSON(ctx, "PUT", contentProduct, contentPath(req.ID), nil, req.Inputs.body())
	if err != nil {
		return resp{}, fmt.Errorf("update content template %s: %w", req.ID, err)
	}
	got, err := decodeContent(o)
	if err != nil {
		return resp{}, err
	}
	st.DateUpdated = got.DateUpdated
	return resp{Output: st}, nil
}

func (r *ContentTemplate) Delete(ctx context.Context, req infer.DeleteRequest[ContentTemplateState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.doJSON(ctx, "DELETE", contentProduct, contentPath(req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete content template %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
