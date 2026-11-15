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
	"net/url"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Lookups of Flex Insights quality management (flex-api.twilio.com /v1/Insights/QualityManagement).
// Categories and questions have no fetch by SID in the spec, so those lookups scan the list.

// jsonDocOf renders a nested value Twilio does not type as a JSON document ("" when absent).
func jsonDocOf(o obj, k string) string {
	v := o.get(k)
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// ------------------------------------------------------------ categories

// GetFlexInsightsCategory looks up a quality management category by SID.
type GetFlexInsightsCategory struct{}

// GetFlexInsightsCategoryArgs are the lookup arguments.
type GetFlexInsightsCategoryArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetFlexInsightsCategoryArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (FY...) of the category.")
}

// FlexInsightsCategoryInfo is one category as Twilio reports it.
type FlexInsightsCategoryInfo struct {
	Sid        string `pulumi:"sid"`
	AccountSid string `pulumi:"accountSid"`
	Name       string `pulumi:"name"`
	Url        string `pulumi:"url"`
}

func (a *FlexInsightsCategoryInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The category SID (FY...).")
	an.Describe(&a.AccountSid, "The SID (AC...) of the account that owns the category.")
	an.Describe(&a.Name, "The name of the category.")
	an.Describe(&a.Url, "The URL of the category resource.")
}

func flexInsightsCategoryInfo(o obj) FlexInsightsCategoryInfo {
	return FlexInsightsCategoryInfo{Sid: o.str("category_sid"), AccountSid: o.str("account_sid"), Name: o.str("name"), Url: o.str("url")}
}

func (f *GetFlexInsightsCategory) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Flex Insights quality management category by SID (found in GET /v1/Insights/QualityManagement/Categories on flex-api.twilio.com, which has no fetch by SID).")
}

func (f *GetFlexInsightsCategory) Invoke(ctx context.Context, req infer.FunctionRequest[GetFlexInsightsCategoryArgs]) (infer.FunctionResponse[FlexInsightsCategoryInfo], error) {
	const fn = "getFlexInsightsCategory"
	type resp = infer.FunctionResponse[FlexInsightsCategoryInfo]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := flexInsightsFind(ctx, c, fn, flexInsightsPath("Categories"), nil, "categories", "category_sid", req.Input.Sid)
	if err != nil {
		return resp{}, lookupFailure(fn, "Flex Insights category "+req.Input.Sid, err)
	}
	return resp{Output: flexInsightsCategoryInfo(o)}, nil
}

// GetFlexInsightsCategories lists the quality management categories.
type GetFlexInsightsCategories struct{}

// GetFlexInsightsCategoriesArgs are the lookup arguments (none).
type GetFlexInsightsCategoriesArgs struct{}

// GetFlexInsightsCategoriesResult is the list.
type GetFlexInsightsCategoriesResult struct {
	Categories []FlexInsightsCategoryInfo `pulumi:"categories"`
}

func (a *GetFlexInsightsCategoriesResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Categories, "The categories, in the order Twilio returns them.")
}

func (f *GetFlexInsightsCategories) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the Flex Insights quality management categories (GET /v1/Insights/QualityManagement/Categories on flex-api.twilio.com).")
}

func (f *GetFlexInsightsCategories) Invoke(ctx context.Context, _ infer.FunctionRequest[GetFlexInsightsCategoriesArgs]) (infer.FunctionResponse[GetFlexInsightsCategoriesResult], error) {
	const fn = "getFlexInsightsCategories"
	none := infer.FunctionResponse[GetFlexInsightsCategoriesResult]{}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	out := []FlexInsightsCategoryInfo{}
	err = listPages(ctx, c, fn, flexProduct, flexInsightsPath("Categories"), url.Values{"PageSize": {"1000"}}, "categories", func(o obj) bool {
		out = append(out, flexInsightsCategoryInfo(o))
		return true
	})
	if err != nil {
		return none, lookupFailure(fn, "Flex Insights categories", err)
	}
	return infer.FunctionResponse[GetFlexInsightsCategoriesResult]{Output: GetFlexInsightsCategoriesResult{Categories: out}}, nil
}

// ------------------------------------------------------------ questions

// GetFlexInsightsQuestion looks up a quality management question by SID.
type GetFlexInsightsQuestion struct{}

// GetFlexInsightsQuestionArgs are the lookup arguments.
type GetFlexInsightsQuestionArgs struct {
	Sid         string  `pulumi:"sid"`
	CategorySid *string `pulumi:"categorySid,optional"`
}

func (a *GetFlexInsightsQuestionArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID (GE...) of the question.")
	an.Describe(&a.CategorySid, "SID (FY...) of the category the question is in. It narrows the search; without it the whole list is searched.")
}

// FlexInsightsQuestionInfo is one question as Twilio reports it.
type FlexInsightsQuestionInfo struct {
	Sid         string  `pulumi:"sid"`
	AccountSid  string  `pulumi:"accountSid"`
	Question    string  `pulumi:"question"`
	Description *string `pulumi:"description,optional"`
	CategorySid string  `pulumi:"categorySid"`
	AnswerSetId string  `pulumi:"answerSetId"`
	AllowNa     bool    `pulumi:"allowNa"`
	Usage       int     `pulumi:"usage"`
	Url         string  `pulumi:"url"`
}

func (a *FlexInsightsQuestionInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The question SID (GE...).")
	an.Describe(&a.AccountSid, "The SID (AC...) of the account that owns the question.")
	an.Describe(&a.Question, "The text of the question.")
	an.Describe(&a.Description, "The description of the question.")
	an.Describe(&a.CategorySid, "The SID (FY...) of the category of the question.")
	an.Describe(&a.AnswerSetId, "The ID of the answer set of the question.")
	an.Describe(&a.AllowNa, "Whether an evaluator may answer not applicable.")
	an.Describe(&a.Usage, "How many questionnaires use the question.")
	an.Describe(&a.Url, "The URL of the question resource.")
}

func flexInsightsQuestionInfo(o obj) FlexInsightsQuestionInfo {
	info := FlexInsightsQuestionInfo{Sid: o.str("question_sid"), AccountSid: o.str("account_sid"), Question: o.str("question"),
		Description: nonEmpty(o.optStr("description")), CategorySid: questionCategorySid(o), AnswerSetId: o.str("answer_set_id"), Url: o.str("url")}
	if v := o.optBool("allow_na"); v != nil {
		info.AllowNa = *v
	}
	if v := o.optInt("usage"); v != nil {
		info.Usage = *v
	}
	return info
}

func (f *GetFlexInsightsQuestion) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Flex Insights quality management question by SID (found in GET /v1/Insights/QualityManagement/Questions on flex-api.twilio.com, which has no fetch by SID).")
}

func (f *GetFlexInsightsQuestion) Invoke(ctx context.Context, req infer.FunctionRequest[GetFlexInsightsQuestionArgs]) (infer.FunctionResponse[FlexInsightsQuestionInfo], error) {
	const fn = "getFlexInsightsQuestion"
	type resp = infer.FunctionResponse[FlexInsightsQuestionInfo]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := flexInsightsQuestion(ctx, c, fn, req.Input.Sid, req.Input.CategorySid)
	if err != nil {
		return resp{}, lookupFailure(fn, "Flex Insights question "+req.Input.Sid, err)
	}
	return resp{Output: flexInsightsQuestionInfo(o)}, nil
}

// GetFlexInsightsQuestions lists the quality management questions.
type GetFlexInsightsQuestions struct{}

// GetFlexInsightsQuestionsArgs are the lookup arguments.
type GetFlexInsightsQuestionsArgs struct {
	CategorySid *string `pulumi:"categorySid,optional"`
}

func (a *GetFlexInsightsQuestionsArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.CategorySid, "Only the questions of the category with this SID (FY...).")
}

// GetFlexInsightsQuestionsResult is the list.
type GetFlexInsightsQuestionsResult struct {
	Questions []FlexInsightsQuestionInfo `pulumi:"questions"`
}

func (a *GetFlexInsightsQuestionsResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Questions, "The questions, in the order Twilio returns them.")
}

func (f *GetFlexInsightsQuestions) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the Flex Insights quality management questions, optionally of one category (GET /v1/Insights/QualityManagement/Questions on flex-api.twilio.com).")
}

func (f *GetFlexInsightsQuestions) Invoke(ctx context.Context, req infer.FunctionRequest[GetFlexInsightsQuestionsArgs]) (infer.FunctionResponse[GetFlexInsightsQuestionsResult], error) {
	const fn = "getFlexInsightsQuestions"
	none := infer.FunctionResponse[GetFlexInsightsQuestionsResult]{}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	q := url.Values{"PageSize": {"1000"}}
	setS(q, "CategorySid", req.Input.CategorySid)
	out := []FlexInsightsQuestionInfo{}
	err = listPages(ctx, c, fn, flexProduct, flexInsightsPath("Questions"), q, "questions", func(o obj) bool {
		out = append(out, flexInsightsQuestionInfo(o))
		return true
	})
	if err != nil {
		return none, lookupFailure(fn, "Flex Insights questions", err)
	}
	return infer.FunctionResponse[GetFlexInsightsQuestionsResult]{Output: GetFlexInsightsQuestionsResult{Questions: out}}, nil
}

// ------------------------------------------------------------ questionnaires

// GetFlexInsightsQuestionnaire looks up a quality management questionnaire by SID.
type GetFlexInsightsQuestionnaire struct{}

// GetFlexInsightsQuestionnaireArgs are the lookup arguments.
type GetFlexInsightsQuestionnaireArgs struct {
	Sid string `pulumi:"sid"`
}

func (a *GetFlexInsightsQuestionnaireArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "SID of the questionnaire.")
}

// FlexInsightsQuestionnaireInfo is one questionnaire as Twilio reports it.
type FlexInsightsQuestionnaireInfo struct {
	Sid          string   `pulumi:"sid"`
	AccountSid   string   `pulumi:"accountSid"`
	Name         string   `pulumi:"name"`
	Description  *string  `pulumi:"description,optional"`
	Active       bool     `pulumi:"active"`
	QuestionSids []string `pulumi:"questionSids"`
	Url          string   `pulumi:"url"`
}

func (a *FlexInsightsQuestionnaireInfo) Annotate(an infer.Annotator) {
	an.Describe(&a.Sid, "The questionnaire SID.")
	an.Describe(&a.AccountSid, "The SID (AC...) of the account that owns the questionnaire.")
	an.Describe(&a.Name, "The name of the questionnaire.")
	an.Describe(&a.Description, "The description of the questionnaire.")
	an.Describe(&a.Active, "Whether the questionnaire is active.")
	an.Describe(&a.QuestionSids, "The SIDs (GE...) of the questions in the questionnaire.")
	an.Describe(&a.Url, "The URL of the questionnaire resource.")
}

func flexInsightsQuestionnaireInfo(o obj) FlexInsightsQuestionnaireInfo {
	info := FlexInsightsQuestionnaireInfo{Sid: o.str("questionnaire_sid"), AccountSid: o.str("account_sid"), Name: o.str("name"),
		Description: nonEmpty(o.optStr("description")), QuestionSids: *questionnaireQuestionSids(o), Url: o.str("url")}
	if v := o.optBool("active"); v != nil {
		info.Active = *v
	}
	return info
}

func (f *GetFlexInsightsQuestionnaire) Annotate(a infer.Annotator) {
	a.Describe(&f, "Look up a Flex Insights quality management questionnaire by SID (GET /v1/Insights/QualityManagement/Questionnaires/{Sid} on flex-api.twilio.com).")
}

func (f *GetFlexInsightsQuestionnaire) Invoke(ctx context.Context, req infer.FunctionRequest[GetFlexInsightsQuestionnaireArgs]) (infer.FunctionResponse[FlexInsightsQuestionnaireInfo], error) {
	const fn = "getFlexInsightsQuestionnaire"
	type resp = infer.FunctionResponse[FlexInsightsQuestionnaireInfo]
	if err := requireArg(fn, "sid", req.Input.Sid); err != nil {
		return resp{}, err
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", flexProduct, flexInsightsPath("Questionnaires", req.Input.Sid), nil, nil)
	if err != nil {
		return resp{}, lookupFailure(fn, "Flex Insights questionnaire "+req.Input.Sid, err)
	}
	return resp{Output: flexInsightsQuestionnaireInfo(o)}, nil
}

// GetFlexInsightsQuestionnaires lists the quality management questionnaires.
type GetFlexInsightsQuestionnaires struct{}

// GetFlexInsightsQuestionnairesArgs are the lookup arguments.
type GetFlexInsightsQuestionnairesArgs struct {
	IncludeInactive *bool `pulumi:"includeInactive,optional"`
}

func (a *GetFlexInsightsQuestionnairesArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.IncludeInactive, "Whether to include inactive questionnaires (Twilio lists active ones only unless this is true).")
}

// GetFlexInsightsQuestionnairesResult is the list.
type GetFlexInsightsQuestionnairesResult struct {
	Questionnaires []FlexInsightsQuestionnaireInfo `pulumi:"questionnaires"`
}

func (a *GetFlexInsightsQuestionnairesResult) Annotate(an infer.Annotator) {
	an.Describe(&a.Questionnaires, "The questionnaires, in the order Twilio returns them.")
}

func (f *GetFlexInsightsQuestionnaires) Annotate(a infer.Annotator) {
	a.Describe(&f, "List the Flex Insights quality management questionnaires (GET /v1/Insights/QualityManagement/Questionnaires on flex-api.twilio.com).")
}

func (f *GetFlexInsightsQuestionnaires) Invoke(ctx context.Context, req infer.FunctionRequest[GetFlexInsightsQuestionnairesArgs]) (infer.FunctionResponse[GetFlexInsightsQuestionnairesResult], error) {
	const fn = "getFlexInsightsQuestionnaires"
	none := infer.FunctionResponse[GetFlexInsightsQuestionnairesResult]{}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	q := url.Values{"PageSize": {"1000"}}
	setB(q, "IncludeInactive", req.Input.IncludeInactive)
	out := []FlexInsightsQuestionnaireInfo{}
	err = listPages(ctx, c, fn, flexProduct, flexInsightsPath("Questionnaires"), q, "questionnaires", func(o obj) bool {
		out = append(out, flexInsightsQuestionnaireInfo(o))
		return true
	})
	if err != nil {
		return none, lookupFailure(fn, "Flex Insights questionnaires", err)
	}
	return infer.FunctionResponse[GetFlexInsightsQuestionnairesResult]{Output: GetFlexInsightsQuestionnairesResult{Questionnaires: out}}, nil
}

// ------------------------------------------------------------ settings

// GetFlexInsightsAnswerSets reads the quality management answer sets.
type GetFlexInsightsAnswerSets struct{}

// GetFlexInsightsAnswerSetsArgs are the lookup arguments (none).
type GetFlexInsightsAnswerSetsArgs struct{}

// GetFlexInsightsAnswerSetsResult is the answer set settings. Twilio does not type the nested
// values, so each is returned as the JSON document Twilio sends.
type GetFlexInsightsAnswerSetsResult struct {
	AccountSid          string `pulumi:"accountSid"`
	AnswerSets          string `pulumi:"answerSets"`
	AnswerSetCategories string `pulumi:"answerSetCategories"`
	NotApplicable       string `pulumi:"notApplicable"`
}

func (a *GetFlexInsightsAnswerSetsResult) Annotate(an infer.Annotator) {
	an.Describe(&a.AccountSid, "The SID (AC...) of the account.")
	an.Describe(&a.AnswerSets, "The answer sets, as a JSON document (empty when Twilio sends none). Use an answer set's ID as answerSetId of a FlexInsightsQuestion.")
	an.Describe(&a.AnswerSetCategories, "The answer set categories, as a JSON document (empty when Twilio sends none).")
	an.Describe(&a.NotApplicable, "The details of the not applicable answer set, as a JSON document (empty when Twilio sends none).")
}

func (f *GetFlexInsightsAnswerSets) Annotate(a infer.Annotator) {
	a.Describe(&f, "Read the Flex Insights quality management answer sets (GET /v1/Insights/QualityManagement/Settings/AnswerSets on flex-api.twilio.com).")
}

func (f *GetFlexInsightsAnswerSets) Invoke(ctx context.Context, _ infer.FunctionRequest[GetFlexInsightsAnswerSetsArgs]) (infer.FunctionResponse[GetFlexInsightsAnswerSetsResult], error) {
	const fn = "getFlexInsightsAnswerSets"
	none := infer.FunctionResponse[GetFlexInsightsAnswerSetsResult]{}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	o, err := c.do(ctx, "GET", flexProduct, flexInsightsPath("Settings/AnswerSets"), nil, nil)
	if err != nil {
		return none, lookupFailure(fn, "the Flex Insights answer sets", err)
	}
	return infer.FunctionResponse[GetFlexInsightsAnswerSetsResult]{Output: GetFlexInsightsAnswerSetsResult{AccountSid: o.str("account_sid"),
		AnswerSets: jsonDocOf(o, "answer_sets"), AnswerSetCategories: jsonDocOf(o, "answer_set_categories"), NotApplicable: jsonDocOf(o, "not_applicable")}}, nil
}

// GetFlexInsightsCommentTags reads the quality management comment tags.
type GetFlexInsightsCommentTags struct{}

// GetFlexInsightsCommentTagsArgs are the lookup arguments (none).
type GetFlexInsightsCommentTagsArgs struct{}

// GetFlexInsightsCommentTagsResult is the comment tag settings. Twilio does not type the comments,
// so they are returned as the JSON document Twilio sends.
type GetFlexInsightsCommentTagsResult struct {
	AccountSid string `pulumi:"accountSid"`
	Comments   string `pulumi:"comments"`
}

func (a *GetFlexInsightsCommentTagsResult) Annotate(an infer.Annotator) {
	an.Describe(&a.AccountSid, "The SID (AC...) of the account.")
	an.Describe(&a.Comments, "The comment tags, as a JSON document (empty when Twilio sends none).")
}

func (f *GetFlexInsightsCommentTags) Annotate(a infer.Annotator) {
	a.Describe(&f, "Read the Flex Insights quality management comment tags (GET /v1/Insights/QualityManagement/Settings/CommentTags on flex-api.twilio.com).")
}

func (f *GetFlexInsightsCommentTags) Invoke(ctx context.Context, _ infer.FunctionRequest[GetFlexInsightsCommentTagsArgs]) (infer.FunctionResponse[GetFlexInsightsCommentTagsResult], error) {
	const fn = "getFlexInsightsCommentTags"
	none := infer.FunctionResponse[GetFlexInsightsCommentTagsResult]{}
	c, err := clientFor(ctx)
	if err != nil {
		return none, err
	}
	o, err := c.do(ctx, "GET", flexProduct, flexInsightsPath("Settings/CommentTags"), nil, nil)
	if err != nil {
		return none, lookupFailure(fn, "the Flex Insights comment tags", err)
	}
	return infer.FunctionResponse[GetFlexInsightsCommentTagsResult]{Output: GetFlexInsightsCommentTagsResult{AccountSid: o.str("account_sid"), Comments: jsonDocOf(o, "comments")}}, nil
}
