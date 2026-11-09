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
	"fmt"
	"net/url"
	"strconv"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// ------------------------------------------------------- getAvailablePhoneNumbers

// GetAvailablePhoneNumbers searches the phone numbers Twilio can sell in a country.
type GetAvailablePhoneNumbers struct{}

// GetAvailablePhoneNumbersArgs are the search arguments. Every filter is optional.
type GetAvailablePhoneNumbersArgs struct {
	AccountSid                    *string `pulumi:"accountSid,optional"`
	CountryCode                   string  `pulumi:"countryCode"`
	NumberType                    *string `pulumi:"numberType,optional"`
	AreaCode                      *int    `pulumi:"areaCode,optional"`
	Contains                      *string `pulumi:"contains,optional"`
	SmsEnabled                    *bool   `pulumi:"smsEnabled,optional"`
	MmsEnabled                    *bool   `pulumi:"mmsEnabled,optional"`
	VoiceEnabled                  *bool   `pulumi:"voiceEnabled,optional"`
	FaxEnabled                    *bool   `pulumi:"faxEnabled,optional"`
	Beta                          *bool   `pulumi:"beta,optional"`
	ExcludeAllAddressRequired     *bool   `pulumi:"excludeAllAddressRequired,optional"`
	ExcludeLocalAddressRequired   *bool   `pulumi:"excludeLocalAddressRequired,optional"`
	ExcludeForeignAddressRequired *bool   `pulumi:"excludeForeignAddressRequired,optional"`
	InRegion                      *string `pulumi:"inRegion,optional"`
	InPostalCode                  *string `pulumi:"inPostalCode,optional"`
	InLocality                    *string `pulumi:"inLocality,optional"`
	InRateCenter                  *string `pulumi:"inRateCenter,optional"`
	InLata                        *string `pulumi:"inLata,optional"`
	NearNumber                    *string `pulumi:"nearNumber,optional"`
	NearLatLong                   *string `pulumi:"nearLatLong,optional"`
	Distance                      *int    `pulumi:"distance,optional"`
	Limit                         *int    `pulumi:"limit,optional"`
}

func (a *GetAvailablePhoneNumbersArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.AccountSid, "SID of the account to search as. Defaults to the provider account.")
	an.Describe(&a.CountryCode, "ISO 3166-1 alpha-2 country code of the numbers, for example US.")
	an.Describe(&a.NumberType, "Kind of number: local, tollFree or mobile. Defaults to local.")
	an.Describe(&a.AreaCode, "Only numbers in this area code (US and Canada).")
	an.Describe(&a.Contains, "Only numbers matching this pattern: digits, or letters and * wildcards, for example 415***1234.")
	an.Describe(&a.SmsEnabled, "Only numbers that can send and receive SMS.")
	an.Describe(&a.MmsEnabled, "Only numbers that can send and receive MMS.")
	an.Describe(&a.VoiceEnabled, "Only numbers that can make and receive calls.")
	an.Describe(&a.FaxEnabled, "Only numbers that can send and receive faxes.")
	an.Describe(&a.Beta, "Only numbers that are new to the Twilio platform.")
	an.Describe(&a.ExcludeAllAddressRequired, "Leave out numbers that need any address on file.")
	an.Describe(&a.ExcludeLocalAddressRequired, "Leave out numbers that need a local address on file.")
	an.Describe(&a.ExcludeForeignAddressRequired, "Leave out numbers that need a foreign address on file.")
	an.Describe(&a.InRegion, "Only numbers in this two-letter state or province (US and Canada).")
	an.Describe(&a.InPostalCode, "Only numbers in this postal or ZIP code (US and Canada).")
	an.Describe(&a.InLocality, "Only numbers in this city or locality (US and Canada).")
	an.Describe(&a.InRateCenter, "Only numbers in this rate center (US and Canada; set inRegion too).")
	an.Describe(&a.InLata, "Only numbers in this LATA (US and Canada).")
	an.Describe(&a.NearNumber, "Only numbers near this number (E.164).")
	an.Describe(&a.NearLatLong, "Only numbers near this latitude,longitude pair, for example 37.8,-122.4.")
	an.Describe(&a.Distance, "Search radius in miles for nearNumber or nearLatLong (at most 500).")
	an.Describe(&a.Limit, "Most numbers to return, 1 to 1000. Defaults to 50.")
}

// PhoneNumberCapabilities says what an available number can do.
type PhoneNumberCapabilities struct {
	Voice bool `pulumi:"voice"`
	Sms   bool `pulumi:"sms"`
	Mms   bool `pulumi:"mms"`
	Fax   bool `pulumi:"fax"`
}

func (a *PhoneNumberCapabilities) Annotate(an infer.Annotator) {
	an.Describe(&a.Voice, "Whether the number supports voice calls.")
	an.Describe(&a.Sms, "Whether the number supports SMS.")
	an.Describe(&a.Mms, "Whether the number supports MMS.")
	an.Describe(&a.Fax, "Whether the number supports fax.")
}

// AvailablePhoneNumber is one purchasable number.
type AvailablePhoneNumber struct {
	PhoneNumber         string                  `pulumi:"phoneNumber"`
	FriendlyName        string                  `pulumi:"friendlyName"`
	IsoCountry          string                  `pulumi:"isoCountry"`
	Locality            *string                 `pulumi:"locality,optional"`
	Region              *string                 `pulumi:"region,optional"`
	PostalCode          *string                 `pulumi:"postalCode,optional"`
	RateCenter          *string                 `pulumi:"rateCenter,optional"`
	Lata                *string                 `pulumi:"lata,optional"`
	Latitude            *float64                `pulumi:"latitude,optional"`
	Longitude           *float64                `pulumi:"longitude,optional"`
	AddressRequirements string                  `pulumi:"addressRequirements"`
	Beta                bool                    `pulumi:"beta"`
	Capabilities        PhoneNumberCapabilities `pulumi:"capabilities"`
}

func (a *AvailablePhoneNumber) Annotate(an infer.Annotator) {
	an.Describe(&a.PhoneNumber, "The number in E.164 format. Use it as IncomingPhoneNumber.phoneNumber to buy it.")
	an.Describe(&a.FriendlyName, "A formatted version of the number.")
	an.Describe(&a.IsoCountry, "ISO country code of the number.")
	an.Describe(&a.Locality, "City or locality of the number, when Twilio reports one.")
	an.Describe(&a.Region, "State or province abbreviation of the number, when Twilio reports one.")
	an.Describe(&a.PostalCode, "Postal or ZIP code of the number, when Twilio reports one.")
	an.Describe(&a.RateCenter, "Rate center of the number, when Twilio reports one.")
	an.Describe(&a.Lata, "LATA of the number, when Twilio reports one.")
	an.Describe(&a.Latitude, "Latitude of the number's location, when Twilio reports one.")
	an.Describe(&a.Longitude, "Longitude of the number's location, when Twilio reports one.")
	an.Describe(&a.AddressRequirements, "Address Twilio needs on file to buy the number: none, any, local or foreign. Empty when not reported.")
	an.Describe(&a.Beta, "Whether the number is new to the Twilio platform.")
	an.Describe(&a.Capabilities, "What the number can do.")
}

// GetAvailablePhoneNumbersResult is the search result.
type GetAvailablePhoneNumbersResult struct {
	CountryCode string                 `pulumi:"countryCode"`
	NumberType  string                 `pulumi:"numberType"`
	Numbers     []AvailablePhoneNumber `pulumi:"numbers"`
}

func (a *GetAvailablePhoneNumbersResult) Annotate(an infer.Annotator) {
	an.Describe(&a.CountryCode, "The country searched.")
	an.Describe(&a.NumberType, "The kind of number searched: local, tollFree or mobile.")
	an.Describe(&a.Numbers, "The matching numbers, in the order Twilio returns them, at most limit.")
}

func (f *GetAvailablePhoneNumbers) Annotate(a infer.Annotator) {
	a.Describe(&f, "Search the phone numbers available to buy (GET /2010-04-01/Accounts/{AccountSid}/AvailablePhoneNumbers/{CountryCode}/{Local|TollFree|Mobile}.json). Searching buys nothing; pass a result to IncomingPhoneNumber to purchase it.")
}

var availableNumberTypes = map[string]string{"local": "Local", "tollFree": "TollFree", "mobile": "Mobile"}

func (f *GetAvailablePhoneNumbers) Invoke(ctx context.Context, req infer.FunctionRequest[GetAvailablePhoneNumbersArgs]) (infer.FunctionResponse[GetAvailablePhoneNumbersResult], error) {
	const fn = "getAvailablePhoneNumbers"
	in := req.Input
	none := infer.FunctionResponse[GetAvailablePhoneNumbersResult]{}
	if err := requireArg(fn, "countryCode", in.CountryCode); err != nil {
		return none, err
	}
	kind := derefOr(in.NumberType, "local")
	seg, ok := availableNumberTypes[kind]
	if !ok {
		return none, fmt.Errorf("%s: numberType must be local, tollFree or mobile, got %q", fn, kind)
	}
	limit, err := limitOrDefault(fn, in.Limit, 50, 1000)
	if err != nil {
		return none, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	q := url.Values{"PageSize": {strconv.Itoa(min(limit, 1000))}}
	setI(q, "AreaCode", in.AreaCode)
	setS(q, "Contains", in.Contains)
	setB(q, "SmsEnabled", in.SmsEnabled)
	setB(q, "MmsEnabled", in.MmsEnabled)
	setB(q, "VoiceEnabled", in.VoiceEnabled)
	setB(q, "FaxEnabled", in.FaxEnabled)
	setB(q, "Beta", in.Beta)
	setB(q, "ExcludeAllAddressRequired", in.ExcludeAllAddressRequired)
	setB(q, "ExcludeLocalAddressRequired", in.ExcludeLocalAddressRequired)
	setB(q, "ExcludeForeignAddressRequired", in.ExcludeForeignAddressRequired)
	setS(q, "InRegion", in.InRegion)
	setS(q, "InPostalCode", in.InPostalCode)
	setS(q, "InLocality", in.InLocality)
	setS(q, "InRateCenter", in.InRateCenter)
	setS(q, "InLata", in.InLata)
	setS(q, "NearNumber", in.NearNumber)
	setS(q, "NearLatLong", in.NearLatLong)
	setI(q, "Distance", in.Distance)
	acct := c.acct(in.AccountSid)
	path := api2010(acct) + "/AvailablePhoneNumbers/" + url.PathEscape(in.CountryCode) + "/" + seg + ".json"
	out := []AvailablePhoneNumber{}
	err = listPages(ctx, c, fn, "api", path, q, "available_phone_numbers", func(o obj) bool {
		out = append(out, availableNumber(o))
		return len(out) < limit
	})
	if err != nil {
		return none, lookupFailure(fn, "available "+kind+" numbers in "+in.CountryCode, err)
	}
	return infer.FunctionResponse[GetAvailablePhoneNumbersResult]{Output: GetAvailablePhoneNumbersResult{
		CountryCode: in.CountryCode, NumberType: kind, Numbers: out,
	}}, nil
}

func availableNumber(o obj) AvailablePhoneNumber {
	capBool := func(names ...string) bool {
		for _, n := range names {
			if b := o.optBool("capabilities." + n); b != nil {
				return *b
			}
		}
		return false
	}
	return AvailablePhoneNumber{
		PhoneNumber: o.str("phone_number"), FriendlyName: o.str("friendly_name"), IsoCountry: o.str("iso_country"),
		Locality: nonEmpty(o.optStr("locality")), Region: nonEmpty(o.optStr("region")), PostalCode: nonEmpty(o.optStr("postal_code")),
		RateCenter: nonEmpty(o.optStr("rate_center")), Lata: nonEmpty(o.optStr("lata")),
		Latitude: floatOf(o, "latitude"), Longitude: floatOf(o, "longitude"),
		AddressRequirements: o.str("address_requirements"), Beta: o.optBool("beta") != nil && *o.optBool("beta"),
		Capabilities: PhoneNumberCapabilities{
			Voice: capBool("voice", "Voice"), Sms: capBool("SMS", "sms"), Mms: capBool("MMS", "mms"), Fax: capBool("fax", "Fax"),
		},
	}
}

// ------------------------------------------------------------- getUsageRecords

// GetUsageRecords reads the account's usage records.
type GetUsageRecords struct{}

// GetUsageRecordsArgs are the lookup arguments.
type GetUsageRecordsArgs struct {
	AccountSid         *string `pulumi:"accountSid,optional"`
	Period             *string `pulumi:"period,optional"`
	Category           *string `pulumi:"category,optional"`
	StartDate          *string `pulumi:"startDate,optional"`
	EndDate            *string `pulumi:"endDate,optional"`
	IncludeSubaccounts *bool   `pulumi:"includeSubaccounts,optional"`
	Limit              *int    `pulumi:"limit,optional"`
}

func (a *GetUsageRecordsArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.AccountSid, "SID of the account. Defaults to the provider account.")
	an.Describe(&a.Period, "How usage is bucketed: allTime (the default), daily, monthly, yearly, today, yesterday, thisMonth or lastMonth.")
	an.Describe(&a.Category, "Only this usage category, for example calls or sms.")
	an.Describe(&a.StartDate, "First date to include, YYYY-MM-DD. Also accepts offsets such as -30days.")
	an.Describe(&a.EndDate, "Last date to include, YYYY-MM-DD. Also accepts offsets such as +30days.")
	an.Describe(&a.IncludeSubaccounts, "Include usage of the account's subaccounts.")
	an.Describe(&a.Limit, "Most records to return, 1 to 10000. Defaults to 1000.")
}

// UsageRecord is one usage record.
type UsageRecord struct {
	AccountSid  string `pulumi:"accountSid"`
	Category    string `pulumi:"category"`
	Description string `pulumi:"description"`
	StartDate   string `pulumi:"startDate"`
	EndDate     string `pulumi:"endDate"`
	Count       string `pulumi:"count"`
	CountUnit   string `pulumi:"countUnit"`
	Usage       string `pulumi:"usage"`
	UsageUnit   string `pulumi:"usageUnit"`
	Price       string `pulumi:"price"`
	PriceUnit   string `pulumi:"priceUnit"`
	AsOf        string `pulumi:"asOf"`
}

func (a *UsageRecord) Annotate(an infer.Annotator) {
	an.Describe(&a.AccountSid, "SID of the account the usage belongs to.")
	an.Describe(&a.Category, "The usage category, for example calls.")
	an.Describe(&a.Description, "Plain-language description of the category.")
	an.Describe(&a.StartDate, "First date the record covers (YYYY-MM-DD).")
	an.Describe(&a.EndDate, "Last date the record covers (YYYY-MM-DD).")
	an.Describe(&a.Count, "Number of usage events, as a decimal string.")
	an.Describe(&a.CountUnit, "Unit of count, for example calls.")
	an.Describe(&a.Usage, "Amount used, as a decimal string.")
	an.Describe(&a.UsageUnit, "Unit of usage, for example minutes.")
	an.Describe(&a.Price, "Total price as a decimal string, kept as Twilio sends it.")
	an.Describe(&a.PriceUnit, "ISO 4217 currency of price.")
	an.Describe(&a.AsOf, "Timestamp the records are up to date as of.")
}

// GetUsageRecordsResult is the list of records.
type GetUsageRecordsResult struct {
	AccountSid string        `pulumi:"accountSid"`
	Period     string        `pulumi:"period"`
	Records    []UsageRecord `pulumi:"records"`
}

func (a *GetUsageRecordsResult) Annotate(an infer.Annotator) {
	an.Describe(&a.AccountSid, "SID of the account queried.")
	an.Describe(&a.Period, "The period bucketing used.")
	an.Describe(&a.Records, "The usage records, in the order Twilio returns them, at most limit.")
}

func (f *GetUsageRecords) Annotate(a infer.Annotator) {
	a.Describe(&f, "Read usage records (GET /2010-04-01/Accounts/{AccountSid}/Usage/Records[/{Period}].json), optionally for one category and date range.")
}

var usagePeriods = map[string]string{
	"allTime": "/AllTime", "daily": "/Daily", "monthly": "/Monthly", "yearly": "/Yearly",
	"today": "/Today", "yesterday": "/Yesterday", "thisMonth": "/ThisMonth", "lastMonth": "/LastMonth",
}

func (f *GetUsageRecords) Invoke(ctx context.Context, req infer.FunctionRequest[GetUsageRecordsArgs]) (infer.FunctionResponse[GetUsageRecordsResult], error) {
	const fn = "getUsageRecords"
	in := req.Input
	none := infer.FunctionResponse[GetUsageRecordsResult]{}
	period := derefOr(in.Period, "allTime")
	seg, ok := usagePeriods[period]
	if !ok {
		return none, fmt.Errorf("%s: period must be one of allTime, daily, monthly, yearly, today, yesterday, thisMonth or lastMonth, got %q", fn, period)
	}
	limit, err := limitOrDefault(fn, in.Limit, 1000, 10000)
	if err != nil {
		return none, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	q := url.Values{"PageSize": {strconv.Itoa(min(limit, 1000))}}
	setS(q, "Category", in.Category)
	setS(q, "StartDate", in.StartDate)
	setS(q, "EndDate", in.EndDate)
	setB(q, "IncludeSubaccounts", in.IncludeSubaccounts)
	acct := c.acct(in.AccountSid)
	out := []UsageRecord{}
	err = listPages(ctx, c, fn, "api", api2010(acct)+"/Usage/Records"+seg+".json", q, "usage_records", func(o obj) bool {
		out = append(out, UsageRecord{
			AccountSid: o.str("account_sid"), Category: o.str("category"), Description: o.str("description"),
			StartDate: o.str("start_date"), EndDate: o.str("end_date"), Count: o.str("count"), CountUnit: o.str("count_unit"),
			Usage: o.str("usage"), UsageUnit: o.str("usage_unit"), Price: o.str("price"), PriceUnit: o.str("price_unit"), AsOf: o.str("as_of"),
		})
		return len(out) < limit
	})
	if err != nil {
		return none, lookupFailure(fn, "usage records of account "+acct, err)
	}
	return infer.FunctionResponse[GetUsageRecordsResult]{Output: GetUsageRecordsResult{AccountSid: acct, Period: period, Records: out}}, nil
}

// ------------------------------------------------------ getAuthorizedConnectApp(s)

// AuthorizedConnectApp is a Connect App the account has authorized.
type AuthorizedConnectApp struct {
	ConnectAppSid string   `pulumi:"connectAppSid"`
	AccountSid    string   `pulumi:"accountSid"`
	FriendlyName  string   `pulumi:"friendlyName"`
	CompanyName   string   `pulumi:"companyName"`
	Description   string   `pulumi:"description"`
	HomepageUrl   string   `pulumi:"homepageUrl"`
	Permissions   []string `pulumi:"permissions"`
}

func (a *AuthorizedConnectApp) Annotate(an infer.Annotator) {
	an.Describe(&a.ConnectAppSid, "SID (CN...) of the Connect App.")
	an.Describe(&a.AccountSid, "SID of the account that authorized the app.")
	an.Describe(&a.FriendlyName, "Name of the Connect App.")
	an.Describe(&a.CompanyName, "Company name set for the Connect App.")
	an.Describe(&a.Description, "Description of the Connect App.")
	an.Describe(&a.HomepageUrl, "Public URL of the Connect App.")
	an.Describe(&a.Permissions, "Permissions the account granted: get-all or post-all.")
}

func authorizedConnectApp(o obj) AuthorizedConnectApp {
	perms := []string{}
	if l := o.optList("permissions"); l != nil {
		perms = *l
	}
	return AuthorizedConnectApp{
		ConnectAppSid: o.str("connect_app_sid"), AccountSid: o.str("account_sid"), FriendlyName: o.str("connect_app_friendly_name"),
		CompanyName: o.str("connect_app_company_name"), Description: o.str("connect_app_description"),
		HomepageUrl: o.str("connect_app_homepage_url"), Permissions: perms,
	}
}

// GetAuthorizedConnectApps lists the Connect Apps an account has authorized.
type GetAuthorizedConnectApps struct{}

// GetAuthorizedConnectAppsArgs are the lookup arguments.
type GetAuthorizedConnectAppsArgs struct {
	AccountSid *string `pulumi:"accountSid,optional"`
}

func (a *GetAuthorizedConnectAppsArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.AccountSid, "SID of the account. Defaults to the provider account.")
}

// GetAuthorizedConnectAppsResult is the list of authorized apps.
type GetAuthorizedConnectAppsResult struct {
	AccountSid string                 `pulumi:"accountSid"`
	Apps       []AuthorizedConnectApp `pulumi:"apps"`
}

func (a *GetAuthorizedConnectAppsResult) Annotate(an infer.Annotator) {
	an.Describe(&a.AccountSid, "SID of the account queried.")
	an.Describe(&a.Apps, "The authorized Connect Apps, in the order Twilio returns them.")
}

func (f *GetAuthorizedConnectApps) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the Connect Apps an account has authorized (GET /2010-04-01/Accounts/{AccountSid}/AuthorizedConnectApps.json).")
}

func (f *GetAuthorizedConnectApps) Invoke(ctx context.Context, req infer.FunctionRequest[GetAuthorizedConnectAppsArgs]) (infer.FunctionResponse[GetAuthorizedConnectAppsResult], error) {
	const fn = "getAuthorizedConnectApps"
	none := infer.FunctionResponse[GetAuthorizedConnectAppsResult]{}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	acct := c.acct(req.Input.AccountSid)
	out := []AuthorizedConnectApp{}
	err = listPages(ctx, c, fn, "api", api2010(acct)+"/AuthorizedConnectApps.json", url.Values{"PageSize": {"1000"}}, "authorized_connect_apps", func(o obj) bool {
		out = append(out, authorizedConnectApp(o))
		return true
	})
	if err != nil {
		return none, lookupFailure(fn, "authorized connect apps of account "+acct, err)
	}
	return infer.FunctionResponse[GetAuthorizedConnectAppsResult]{Output: GetAuthorizedConnectAppsResult{AccountSid: acct, Apps: out}}, nil
}

// GetAuthorizedConnectApp looks up one authorized Connect App.
type GetAuthorizedConnectApp struct{}

// GetAuthorizedConnectAppArgs are the lookup arguments.
type GetAuthorizedConnectAppArgs struct {
	AccountSid    *string `pulumi:"accountSid,optional"`
	ConnectAppSid string  `pulumi:"connectAppSid"`
}

func (a *GetAuthorizedConnectAppArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.AccountSid, "SID of the account. Defaults to the provider account.")
	an.Describe(&a.ConnectAppSid, "SID (CN...) of the Connect App.")
}

func (f *GetAuthorizedConnectApp) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up one Connect App an account has authorized (GET /2010-04-01/Accounts/{AccountSid}/AuthorizedConnectApps/{ConnectAppSid}.json).")
}

func (f *GetAuthorizedConnectApp) Invoke(ctx context.Context, req infer.FunctionRequest[GetAuthorizedConnectAppArgs]) (infer.FunctionResponse[AuthorizedConnectApp], error) {
	const fn = "getAuthorizedConnectApp"
	none := infer.FunctionResponse[AuthorizedConnectApp]{}
	if err := requireArg(fn, "connectAppSid", req.Input.ConnectAppSid); err != nil {
		return none, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	acct := c.acct(req.Input.AccountSid)
	o, err := c.do(ctx, "GET", "api", api2010(acct)+"/AuthorizedConnectApps/"+url.PathEscape(req.Input.ConnectAppSid)+".json", nil, nil)
	if err != nil {
		return none, lookupFailure(fn, "authorized connect app "+req.Input.ConnectAppSid, err)
	}
	return infer.FunctionResponse[AuthorizedConnectApp]{Output: authorizedConnectApp(o)}, nil
}
