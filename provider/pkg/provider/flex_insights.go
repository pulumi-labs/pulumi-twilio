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
	"regexp"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Flex Insights quality management (flex-api.twilio.com /v1/Insights/QualityManagement): the
// categories, questions and questionnaires evaluators score conversations with. The shapes follow
// twilio-oai spec/json/twilio_flex_v1.json (info.version 1.0.0). Assessments, assessment comments
// and segments are runtime data and are not managed.
//
// The spec gives every operation an optional Authorization header for a Flex Insights token. The
// resources do not expose it: they use the account credentials like every other Twilio resource.
// Unverified against a live account whether quality management also accepts those credentials.
//
// Categories and questions have no fetch by SID in the spec, so Read finds them in the list.

var (
	flexInsightsCategorySidRE = regexp.MustCompile(`^FY[0-9a-fA-F]{32}$`)
	flexInsightsQuestionSidRE = regexp.MustCompile(`^GE[0-9a-fA-F]{32}$`)
)

func flexInsightsPath(coll string, rest ...string) string {
	return flexPath("Insights/QualityManagement/"+coll, rest...)
}

// flexInsightsFind returns the item of a list whose sidKey is sid, or a 404 error when the list
// has none (the object was deleted, or never existed).
func flexInsightsFind(ctx context.Context, c *Client, fn, path string, query url.Values, itemsKey, sidKey, sid string) (obj, error) {
	if query == nil {
		query = url.Values{}
	}
	query.Set("PageSize", "1000")
	var found obj
	err := listPages(ctx, c, fn, flexProduct, path, query, itemsKey, func(o obj) bool {
		if o.str(sidKey) == sid {
			found = o
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, &apiError{Status: 404, Message: sid + " is not in the list"}
	}
	return found, nil
}

// flexInsightsQuestion finds a question by SID. A category narrows the list when it is known; a
// question that is not in it (moved to another category outside Pulumi) is looked up in the whole list.
func flexInsightsQuestion(ctx context.Context, c *Client, fn, sid string, categorySid *string) (obj, error) {
	path := flexInsightsPath("Questions")
	if categorySid != nil && *categorySid != "" {
		o, err := flexInsightsFind(ctx, c, fn, path, url.Values{"CategorySid": {*categorySid}}, "questions", "question_sid", sid)
		if err == nil || !isNotFound(err) {
			return o, err
		}
	}
	return flexInsightsFind(ctx, c, fn, path, nil, "questions", "question_sid", sid)
}

// questionnaireQuestionSids is the SIDs of the questions of a questionnaire object.
func questionnaireQuestionSids(o obj) *[]string {
	items, _ := o.get("questions").([]any)
	out := []string{}
	for _, it := range items {
		if m, ok := it.(map[string]any); ok {
			if s := obj(m).str("question_sid"); s != "" {
				out = append(out, s)
			}
		}
	}
	return &out
}

// ------------------------------------------------------------ FlexInsightsCategory

// FlexInsightsCategory manages a quality management category, the group a question belongs to.
type FlexInsightsCategory struct{}

// FlexInsightsCategoryArgs are the inputs of a FlexInsightsCategory.
type FlexInsightsCategoryArgs struct {
	Name string `pulumi:"name"`
}

func (a *FlexInsightsCategoryArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Name, "The name of the category.")
}

// FlexInsightsCategoryState is the persisted state of a FlexInsightsCategory.
type FlexInsightsCategoryState struct {
	FlexInsightsCategoryArgs

	Sid        string `pulumi:"sid"`
	AccountSid string `pulumi:"accountSid"`
	Url        string `pulumi:"url"`
}

func (s *FlexInsightsCategoryState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The category SID (FY...).")
	a.Describe(&s.AccountSid, "The SID (AC...) of the account that owns the category.")
	a.Describe(&s.Url, "The URL of the category resource.")
}

func (r *FlexInsightsCategory) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Flex Insights quality management category. The resource ID is the category SID. Twilio has no fetch by SID for categories, so a refresh finds the category in the list. Destroying the resource deletes the category. Unverified against a live account: whether the account credentials are accepted instead of a Flex Insights token.")
}

var _ infer.CustomResource[FlexInsightsCategoryArgs, FlexInsightsCategoryState] = (*FlexInsightsCategory)(nil)
var _ infer.CustomCheck[FlexInsightsCategoryArgs] = (*FlexInsightsCategory)(nil)
var _ infer.CustomDiff[FlexInsightsCategoryArgs, FlexInsightsCategoryState] = (*FlexInsightsCategory)(nil)
var _ infer.CustomRead[FlexInsightsCategoryArgs, FlexInsightsCategoryState] = (*FlexInsightsCategory)(nil)
var _ infer.CustomUpdate[FlexInsightsCategoryArgs, FlexInsightsCategoryState] = (*FlexInsightsCategory)(nil)
var _ infer.CustomDelete[FlexInsightsCategoryState] = (*FlexInsightsCategory)(nil)

func (r *FlexInsightsCategory) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[FlexInsightsCategoryArgs], error) {
	args, failures, err := infer.DefaultCheck[FlexInsightsCategoryArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[FlexInsightsCategoryArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	if !c.unk("name") {
		c.required("name", args.Name)
	}
	return infer.CheckResponse[FlexInsightsCategoryArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *FlexInsightsCategory) Diff(_ context.Context, req infer.DiffRequest[FlexInsightsCategoryArgs, FlexInsightsCategoryState]) (infer.DiffResponse, error) {
	d := trDiffSet{}
	d.requiredStr("name", req.State.Name, req.Inputs.Name)
	return d.response(), nil
}

func (a FlexInsightsCategoryArgs) form() url.Values {
	return url.Values{"Name": {a.Name}}
}

func (s *FlexInsightsCategoryState) fill(o obj) {
	s.Sid = o.str("category_sid")
	s.AccountSid = o.str("account_sid")
	s.Url = o.str("url")
}

func (r *FlexInsightsCategory) Create(ctx context.Context, req infer.CreateRequest[FlexInsightsCategoryArgs]) (infer.CreateResponse[FlexInsightsCategoryState], error) {
	type resp = infer.CreateResponse[FlexInsightsCategoryState]
	st := FlexInsightsCategoryState{FlexInsightsCategoryArgs: req.Inputs}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", flexProduct, flexInsightsPath("Categories"), nil, req.Inputs.form())
	if err != nil {
		return resp{}, fmt.Errorf("create Flex Insights category: %w", err)
	}
	st.fill(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *FlexInsightsCategory) Read(ctx context.Context, req infer.ReadRequest[FlexInsightsCategoryArgs, FlexInsightsCategoryState]) (infer.ReadResponse[FlexInsightsCategoryArgs, FlexInsightsCategoryState], error) {
	type resp = infer.ReadResponse[FlexInsightsCategoryArgs, FlexInsightsCategoryState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := flexInsightsFind(ctx, c, "read Flex Insights category", flexInsightsPath("Categories"), nil, "categories", "category_sid", req.ID)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Flex Insights category", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Flex Insights category %s: %w", req.ID, err)
	}
	in := req.Inputs
	if v := o.str("name"); v != "" {
		in.Name = v
	}
	st := FlexInsightsCategoryState{FlexInsightsCategoryArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *FlexInsightsCategory) Update(ctx context.Context, req infer.UpdateRequest[FlexInsightsCategoryArgs, FlexInsightsCategoryState]) (infer.UpdateResponse[FlexInsightsCategoryState], error) {
	type resp = infer.UpdateResponse[FlexInsightsCategoryState]
	st := req.State
	st.FlexInsightsCategoryArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", flexProduct, flexInsightsPath("Categories", req.ID), nil, req.Inputs.form())
	if err != nil {
		return resp{}, fmt.Errorf("update Flex Insights category %s: %w", req.ID, err)
	}
	st.fill(o)
	return resp{Output: st}, nil
}

func (r *FlexInsightsCategory) Delete(ctx context.Context, req infer.DeleteRequest[FlexInsightsCategoryState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", flexProduct, flexInsightsPath("Categories", req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete Flex Insights category %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------------------------ FlexInsightsQuestion

// FlexInsightsQuestion manages a quality management question that evaluators answer.
type FlexInsightsQuestion struct{}

// FlexInsightsQuestionArgs are the inputs of a FlexInsightsQuestion.
type FlexInsightsQuestionArgs struct {
	CategorySid string  `pulumi:"categorySid"`
	Question    string  `pulumi:"question"`
	AnswerSetId string  `pulumi:"answerSetId"`
	AllowNa     bool    `pulumi:"allowNa"`
	Description *string `pulumi:"description,optional"`
}

func (a *FlexInsightsQuestionArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.CategorySid, "The SID (FY...) of the category the question belongs to.")
	an.Describe(&a.Question, "The text of the question.")
	an.Describe(&a.AnswerSetId, "The ID of the answer set the question is answered from. The getFlexInsightsAnswerSets function lists the available ones.")
	an.Describe(&a.AllowNa, "Whether an evaluator may answer not applicable.")
	an.Describe(&a.Description, "A description of the question. Removing it leaves the description unchanged.")
}

// FlexInsightsQuestionState is the persisted state of a FlexInsightsQuestion.
type FlexInsightsQuestionState struct {
	FlexInsightsQuestionArgs

	Sid        string `pulumi:"sid"`
	AccountSid string `pulumi:"accountSid"`
	Usage      int    `pulumi:"usage"`
	Url        string `pulumi:"url"`
}

func (s *FlexInsightsQuestionState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The question SID (GE...).")
	a.Describe(&s.AccountSid, "The SID (AC...) of the account that owns the question.")
	a.Describe(&s.Usage, "How many questionnaires use the question.")
	a.Describe(&s.Url, "The URL of the question resource.")
}

func (r *FlexInsightsQuestion) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Flex Insights quality management question. The resource ID is the question SID. Twilio has no fetch by SID for questions, so a refresh finds the question in the list. Destroying the resource deletes the question. Unverified against a live account: whether the account credentials are accepted instead of a Flex Insights token.")
}

var _ infer.CustomResource[FlexInsightsQuestionArgs, FlexInsightsQuestionState] = (*FlexInsightsQuestion)(nil)
var _ infer.CustomCheck[FlexInsightsQuestionArgs] = (*FlexInsightsQuestion)(nil)
var _ infer.CustomDiff[FlexInsightsQuestionArgs, FlexInsightsQuestionState] = (*FlexInsightsQuestion)(nil)
var _ infer.CustomRead[FlexInsightsQuestionArgs, FlexInsightsQuestionState] = (*FlexInsightsQuestion)(nil)
var _ infer.CustomUpdate[FlexInsightsQuestionArgs, FlexInsightsQuestionState] = (*FlexInsightsQuestion)(nil)
var _ infer.CustomDelete[FlexInsightsQuestionState] = (*FlexInsightsQuestion)(nil)

func (r *FlexInsightsQuestion) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[FlexInsightsQuestionArgs], error) {
	args, failures, err := infer.DefaultCheck[FlexInsightsQuestionArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[FlexInsightsQuestionArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	c.sid("categorySid", args.CategorySid, flexInsightsCategorySidRE, "Flex Insights category (FY followed by 32 hex characters)")
	if !c.unk("question") {
		c.required("question", args.Question)
	}
	if !c.unk("answerSetId") {
		c.required("answerSetId", args.AnswerSetId)
	}
	return infer.CheckResponse[FlexInsightsQuestionArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *FlexInsightsQuestion) Diff(_ context.Context, req infer.DiffRequest[FlexInsightsQuestionArgs, FlexInsightsQuestionState]) (infer.DiffResponse, error) {
	o, n := req.State.FlexInsightsQuestionArgs, req.Inputs
	d := trDiffSet{}
	d.requiredStr("categorySid", o.CategorySid, n.CategorySid)
	d.requiredStr("question", o.Question, n.Question)
	d.requiredStr("answerSetId", o.AnswerSetId, n.AnswerSetId)
	d.boolean("allowNa", &o.AllowNa, &n.AllowNa)
	d.str("description", o.Description, n.Description)
	return d.response(), nil
}

// form is the create and update body: the update needs AllowNa, so both always send it.
func (a FlexInsightsQuestionArgs) form() url.Values {
	form := url.Values{}
	form.Set("CategorySid", a.CategorySid)
	form.Set("Question", a.Question)
	form.Set("AnswerSetId", a.AnswerSetId)
	setB(form, "AllowNa", &a.AllowNa)
	setS(form, "Description", a.Description)
	return form
}

func (s *FlexInsightsQuestionState) fill(o obj) {
	s.Sid = o.str("question_sid")
	s.AccountSid = o.str("account_sid")
	if v := o.optInt("usage"); v != nil {
		s.Usage = *v
	} else {
		s.Usage = 0
	}
	s.Url = o.str("url")
}

// questionCategorySid reads the SID of the category object of a question.
func questionCategorySid(o obj) string {
	if v := o.str("category.category_sid"); v != "" {
		return v
	}
	return o.str("category.sid")
}

func (r *FlexInsightsQuestion) Create(ctx context.Context, req infer.CreateRequest[FlexInsightsQuestionArgs]) (infer.CreateResponse[FlexInsightsQuestionState], error) {
	type resp = infer.CreateResponse[FlexInsightsQuestionState]
	st := FlexInsightsQuestionState{FlexInsightsQuestionArgs: req.Inputs}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", flexProduct, flexInsightsPath("Questions"), nil, req.Inputs.form())
	if err != nil {
		return resp{}, fmt.Errorf("create Flex Insights question: %w", err)
	}
	st.fill(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *FlexInsightsQuestion) Read(ctx context.Context, req infer.ReadRequest[FlexInsightsQuestionArgs, FlexInsightsQuestionState]) (infer.ReadResponse[FlexInsightsQuestionArgs, FlexInsightsQuestionState], error) {
	type resp = infer.ReadResponse[FlexInsightsQuestionArgs, FlexInsightsQuestionState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	var known *string
	if req.State.CategorySid != "" {
		known = &req.State.CategorySid
	}
	o, err := flexInsightsQuestion(ctx, c, "read Flex Insights question", req.ID, known)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Flex Insights question", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Flex Insights question %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := req.State.Sid == ""
	if v := questionCategorySid(o); v != "" {
		in.CategorySid = v
	}
	if v := o.str("question"); v != "" {
		in.Question = v
	}
	if v := o.str("answer_set_id"); v != "" {
		in.AnswerSetId = v
	}
	if v := o.optBool("allow_na"); v != nil {
		in.AllowNa = *v
	}
	in.Description = trStr(in.Description, o.optStr("description"), "", imported)
	st := FlexInsightsQuestionState{FlexInsightsQuestionArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

func (r *FlexInsightsQuestion) Update(ctx context.Context, req infer.UpdateRequest[FlexInsightsQuestionArgs, FlexInsightsQuestionState]) (infer.UpdateResponse[FlexInsightsQuestionState], error) {
	type resp = infer.UpdateResponse[FlexInsightsQuestionState]
	st := req.State
	st.FlexInsightsQuestionArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", flexProduct, flexInsightsPath("Questions", req.ID), nil, req.Inputs.form())
	if err != nil {
		return resp{}, fmt.Errorf("update Flex Insights question %s: %w", req.ID, err)
	}
	st.fill(o)
	return resp{Output: st}, nil
}

func (r *FlexInsightsQuestion) Delete(ctx context.Context, req infer.DeleteRequest[FlexInsightsQuestionState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", flexProduct, flexInsightsPath("Questions", req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete Flex Insights question %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}

// ------------------------------------------------------------ FlexInsightsQuestionnaire

// FlexInsightsQuestionnaire manages a quality management questionnaire: a named set of questions.
type FlexInsightsQuestionnaire struct{}

// FlexInsightsQuestionnaireArgs are the inputs of a FlexInsightsQuestionnaire.
type FlexInsightsQuestionnaireArgs struct {
	Name         string    `pulumi:"name"`
	Description  *string   `pulumi:"description,optional"`
	Active       *bool     `pulumi:"active,optional"`
	QuestionSids *[]string `pulumi:"questionSids,optional"`
}

func (a *FlexInsightsQuestionnaireArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Name, "The name of the questionnaire.")
	an.Describe(&a.Description, "A description of the questionnaire. Removing it leaves the description unchanged.")
	an.Describe(&a.Active, "Whether the questionnaire is active and offered to evaluators. Twilio does not document its default, so an import does not adopt it. Removing it leaves the setting unchanged.")
	an.Describe(&a.QuestionSids, "The SIDs (GE...) of the questions in the questionnaire. The order is not significant. Removing the property leaves the questions unchanged. Unverified against a live account: that an update replaces the whole set.")
}

// FlexInsightsQuestionnaireState is the persisted state of a FlexInsightsQuestionnaire.
type FlexInsightsQuestionnaireState struct {
	FlexInsightsQuestionnaireArgs

	Sid        string `pulumi:"sid"`
	AccountSid string `pulumi:"accountSid"`
	Url        string `pulumi:"url"`
}

func (s *FlexInsightsQuestionnaireState) Annotate(a infer.Annotator) {
	a.Describe(&s.Sid, "The questionnaire SID.")
	a.Describe(&s.AccountSid, "The SID (AC...) of the account that owns the questionnaire.")
	a.Describe(&s.Url, "The URL of the questionnaire resource.")
}

func (r *FlexInsightsQuestionnaire) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Flex Insights quality management questionnaire. The resource ID is the questionnaire SID. Destroying the resource deletes the questionnaire. Unverified against a live account: whether the account credentials are accepted instead of a Flex Insights token.")
}

var _ infer.CustomResource[FlexInsightsQuestionnaireArgs, FlexInsightsQuestionnaireState] = (*FlexInsightsQuestionnaire)(nil)
var _ infer.CustomCheck[FlexInsightsQuestionnaireArgs] = (*FlexInsightsQuestionnaire)(nil)
var _ infer.CustomDiff[FlexInsightsQuestionnaireArgs, FlexInsightsQuestionnaireState] = (*FlexInsightsQuestionnaire)(nil)
var _ infer.CustomRead[FlexInsightsQuestionnaireArgs, FlexInsightsQuestionnaireState] = (*FlexInsightsQuestionnaire)(nil)
var _ infer.CustomUpdate[FlexInsightsQuestionnaireArgs, FlexInsightsQuestionnaireState] = (*FlexInsightsQuestionnaire)(nil)
var _ infer.CustomDelete[FlexInsightsQuestionnaireState] = (*FlexInsightsQuestionnaire)(nil)

func (r *FlexInsightsQuestionnaire) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[FlexInsightsQuestionnaireArgs], error) {
	args, failures, err := infer.DefaultCheck[FlexInsightsQuestionnaireArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[FlexInsightsQuestionnaireArgs]{Inputs: args, Failures: failures}, err
	}
	c := &checker{unk: unknowns(req.NewInputs)}
	if !c.unk("name") {
		c.required("name", args.Name)
	}
	if args.QuestionSids != nil && !c.unk("questionSids") {
		for i, s := range *args.QuestionSids {
			if unknownAt(req.NewInputs, "questionSids", i) {
				continue
			}
			if !flexInsightsQuestionSidRE.MatchString(s) {
				c.fail("questionSids", "questionSids must hold the SIDs of Flex Insights questions (GE followed by 32 hex characters), got "+s)
			}
		}
	}
	return infer.CheckResponse[FlexInsightsQuestionnaireArgs]{Inputs: args, Failures: c.failures}, nil
}

func (r *FlexInsightsQuestionnaire) Diff(_ context.Context, req infer.DiffRequest[FlexInsightsQuestionnaireArgs, FlexInsightsQuestionnaireState]) (infer.DiffResponse, error) {
	o, n := req.State.FlexInsightsQuestionnaireArgs, req.Inputs
	d := trDiffSet{}
	d.requiredStr("name", o.Name, n.Name)
	d.str("description", o.Description, n.Description)
	d.boolean("active", o.Active, n.Active)
	d.strList("questionSids", o.QuestionSids, n.QuestionSids)
	return d.response(), nil
}

func (a FlexInsightsQuestionnaireArgs) form() url.Values {
	form := url.Values{"Name": {a.Name}}
	setS(form, "Description", a.Description)
	setB(form, "Active", a.Active)
	cvAddList(form, "QuestionSids", a.QuestionSids)
	return form
}

func (s *FlexInsightsQuestionnaireState) fill(o obj) {
	s.Sid = o.str("questionnaire_sid")
	s.AccountSid = o.str("account_sid")
	s.Url = o.str("url")
}

func (r *FlexInsightsQuestionnaire) Create(ctx context.Context, req infer.CreateRequest[FlexInsightsQuestionnaireArgs]) (infer.CreateResponse[FlexInsightsQuestionnaireState], error) {
	type resp = infer.CreateResponse[FlexInsightsQuestionnaireState]
	st := FlexInsightsQuestionnaireState{FlexInsightsQuestionnaireArgs: req.Inputs}
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "POST", flexProduct, flexInsightsPath("Questionnaires"), nil, req.Inputs.form())
	if err != nil {
		return resp{}, fmt.Errorf("create Flex Insights questionnaire: %w", err)
	}
	st.fill(o)
	return resp{ID: st.Sid, Output: st}, nil
}

func (r *FlexInsightsQuestionnaire) Read(ctx context.Context, req infer.ReadRequest[FlexInsightsQuestionnaireArgs, FlexInsightsQuestionnaireState]) (infer.ReadResponse[FlexInsightsQuestionnaireArgs, FlexInsightsQuestionnaireState], error) {
	type resp = infer.ReadResponse[FlexInsightsQuestionnaireArgs, FlexInsightsQuestionnaireState]
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	o, err := c.do(ctx, "GET", flexProduct, flexInsightsPath("Questionnaires", req.ID), nil, nil)
	if err != nil {
		if isNotFound(err) {
			if err := notFoundOnImport("Flex Insights questionnaire", req.ID, req.Inputs, req.State); err != nil {
				return resp{}, err
			}
			return resp{}, nil
		}
		return resp{}, fmt.Errorf("read Flex Insights questionnaire %s: %w", req.ID, err)
	}
	in := req.Inputs
	imported := req.State.Sid == ""
	if v := o.str("name"); v != "" {
		in.Name = v
	}
	in.Description = trStr(in.Description, o.optStr("description"), "", imported)
	in.Active = flexKeep(in.Active, o.optBool("active"), imported)
	in.QuestionSids = cvList(in.QuestionSids, questionnaireQuestionSids(o), imported)
	st := FlexInsightsQuestionnaireState{FlexInsightsQuestionnaireArgs: in}
	st.fill(o)
	return resp{ID: req.ID, Inputs: in, State: st}, nil
}

// Update sends the stated settings. Twilio requires Active on every update, so when the program
// does not state it the current value is read and sent back unchanged.
func (r *FlexInsightsQuestionnaire) Update(ctx context.Context, req infer.UpdateRequest[FlexInsightsQuestionnaireArgs, FlexInsightsQuestionnaireState]) (infer.UpdateResponse[FlexInsightsQuestionnaireState], error) {
	type resp = infer.UpdateResponse[FlexInsightsQuestionnaireState]
	st := req.State
	st.FlexInsightsQuestionnaireArgs = req.Inputs
	if req.DryRun {
		return resp{Output: st}, nil
	}
	c, err := clientFor(ctx)
	if err != nil {
		return resp{}, err
	}
	path := flexInsightsPath("Questionnaires", req.ID)
	form := req.Inputs.form()
	if req.Inputs.Active == nil {
		cur, err := c.do(ctx, "GET", flexProduct, path, nil, nil)
		if err != nil {
			return resp{}, fmt.Errorf("update Flex Insights questionnaire %s: %w", req.ID, err)
		}
		active := true
		if v := cur.optBool("active"); v != nil {
			active = *v
		}
		setB(form, "Active", &active)
	}
	o, err := c.do(ctx, "POST", flexProduct, path, nil, form)
	if err != nil {
		return resp{}, fmt.Errorf("update Flex Insights questionnaire %s: %w", req.ID, err)
	}
	st.fill(o)
	return resp{Output: st}, nil
}

func (r *FlexInsightsQuestionnaire) Delete(ctx context.Context, req infer.DeleteRequest[FlexInsightsQuestionnaireState]) (infer.DeleteResponse, error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if _, err := c.do(ctx, "DELETE", flexProduct, flexInsightsPath("Questionnaires", req.ID), nil, nil); err != nil && !isNotFound(err) {
		return infer.DeleteResponse{}, fmt.Errorf("delete Flex Insights questionnaire %s: %w", req.ID, err)
	}
	return infer.DeleteResponse{}, nil
}
