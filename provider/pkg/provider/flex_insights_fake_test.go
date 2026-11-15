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
	"net/http"
	"net/url"
)

// The Flex Insights quality management routes, served for the Flex Insights tests:
//
//	/flex-api/v1/Insights/QualityManagement/Categories[/{sid}]          GET list, POST create (Name), POST update (Name), DELETE
//	/flex-api/v1/Insights/QualityManagement/Questions[/{sid}]           GET list (CategorySid), POST create, POST update (AllowNa required), DELETE
//	/flex-api/v1/Insights/QualityManagement/Questionnaires[/{sid}]      GET list (IncludeInactive), GET, POST create, POST update (Active required), DELETE
//	/flex-api/v1/Insights/QualityManagement/Settings/{AnswerSets,CommentTags}   GET
//
// Categories and questions have no fetch by SID, as in twilio-oai spec/json/twilio_flex_v1.json.
// Objects are stored as kind "flex" keyed Insights/<collection>/<sid>.

var flexInsightsQuestionFields = []fakeField{{"Question", "question", "s"}, {"Description", "description", "s"}, {"AnswerSetId", "answer_set_id", "s"}, {"AllowNa", "allow_na", "b"}}

var flexInsightsQuestionnaireFields = []fakeField{{"Name", "name", "s"}, {"Description", "description", "s"}, {"Active", "active", "b"}}

func (f *fakeTwilio) flexInsights(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	notFoundErr := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	if len(parts) >= 2 && parts[0] == "Settings" && r.Method == "GET" {
		switch parts[1] {
		case "AnswerSets":
			fwrite(w, 200, map[string]any{"account_sid": testAccount, "url": "https://flex-api.twilio.com/v1/Insights/QualityManagement/Settings/AnswerSets",
				"answer_sets":           []any{map[string]any{"answer_set_id": "yes-no", "name": "Yes or no"}},
				"answer_set_categories": []any{map[string]any{"category": "binary"}},
				"not_applicable":        map[string]any{"answer_text": "N/A"}})
		case "CommentTags":
			fwrite(w, 200, map[string]any{"account_sid": testAccount, "url": "https://flex-api.twilio.com/v1/Insights/QualityManagement/Settings/CommentTags",
				"comments": map[string]any{"tags": []any{"good"}}})
		default:
			notFoundErr()
		}
		return
	}
	if len(parts) == 0 || len(parts) > 2 {
		notFoundErr()
		return
	}
	coll := parts[0]
	switch coll {
	case "Categories", "Questions", "Questionnaires":
	default:
		notFoundErr()
		return
	}
	itemsKey := map[string]string{"Categories": "categories", "Questions": "questions", "Questionnaires": "questionnaires"}[coll]
	sidKey := map[string]string{"Categories": "category_sid", "Questions": "question_sid", "Questionnaires": "questionnaire_sid"}[coll]
	prefix := map[string]string{"Categories": "FY", "Questions": "GE", "Questionnaires": "QQ"}[coll]
	base := "https://flex-api.twilio.com/v1/Insights/QualityManagement/" + coll
	storeKey := func(sid string) string { return "Insights/" + coll + "/" + sid }
	if len(parts) == 1 {
		switch r.Method {
		case "GET":
			q := r.URL.Query()
			keep := func(o map[string]any) bool {
				switch coll {
				case "Questions":
					if cs := q.Get("CategorySid"); cs != "" {
						c, _ := o["category"].(map[string]any)
						return c["category_sid"] == cs
					}
				case "Questionnaires":
					if q.Get("IncludeInactive") != "true" {
						return o["active"] == true
					}
				}
				return true
			}
			metaPage(w, r, itemsKey, f.flexList("Insights/"+coll, keep), 2)
		case "POST":
			required := map[string][]string{"Categories": {"Name"}, "Questions": {"CategorySid", "Question", "AnswerSetId", "AllowNa"}, "Questionnaires": {"Name"}}[coll]
			for _, req := range required {
				if form.Get(req) == "" {
					ferr(w, 400, "Missing required parameter "+req)
					return
				}
			}
			sid := f.sid(prefix)
			o := map[string]any{"account_sid": testAccount, sidKey: sid, "url": base + "/" + sid}
			if msg := f.flexInsightsApply(coll, o, form); msg != "" {
				ferr(w, 400, msg)
				return
			}
			switch coll {
			case "Questions":
				o["usage"] = float64(0)
			case "Questionnaires":
				if _, ok := o["active"]; !ok {
					o["active"] = true
				}
			}
			f.put("flex", testAccount, storeKey(sid), o)
			fwrite(w, 201, o)
		default:
			ferr(w, 405, "method not allowed")
		}
		return
	}
	sid := parts[1]
	o, ok := f.objs[key("flex", testAccount, storeKey(sid))]
	if !ok {
		notFoundErr()
		return
	}
	switch r.Method {
	case "GET":
		if coll != "Questionnaires" {
			ferr(w, 405, "method not allowed")
			return
		}
		fwrite(w, 200, o)
	case "POST":
		switch coll {
		case "Categories":
			if form.Get("Name") == "" {
				ferr(w, 400, "Missing required parameter Name")
				return
			}
		case "Questions":
			if form.Get("AllowNa") == "" {
				ferr(w, 400, "Missing required parameter AllowNa")
				return
			}
		case "Questionnaires":
			if form.Get("Active") == "" {
				ferr(w, 400, "Missing required parameter Active")
				return
			}
		}
		if msg := f.flexInsightsApply(coll, o, form); msg != "" {
			ferr(w, 400, msg)
			return
		}
		fwrite(w, 200, o)
	case "DELETE":
		f.del("flex", testAccount, storeKey(sid))
		w.WriteHeader(204)
	default:
		ferr(w, 405, "method not allowed")
	}
}

// flexInsightsApply applies the posted fields; the caller holds the lock.
func (f *fakeTwilio) flexInsightsApply(coll string, o map[string]any, form url.Values) string {
	switch coll {
	case "Categories":
		if v, ok := form["Name"]; ok {
			o["name"] = v[0]
		}
	case "Questions":
		if msg := applyForm(o, form, flexInsightsQuestionFields); msg != "" {
			return msg
		}
		if v, ok := form["CategorySid"]; ok {
			c, ok := f.objs[key("flex", testAccount, "Insights/Categories/"+v[0])]
			if !ok {
				return "Unknown CategorySid " + v[0]
			}
			o["category"] = map[string]any{"category_sid": v[0], "name": c["name"]}
		}
	case "Questionnaires":
		if msg := applyForm(o, form, flexInsightsQuestionnaireFields); msg != "" {
			return msg
		}
		if sids, ok := form["QuestionSids"]; ok {
			qs := []any{}
			for _, s := range sids {
				q, ok := f.objs[key("flex", testAccount, "Insights/Questions/"+s)]
				if !ok {
					return "Unknown QuestionSid " + s
				}
				qs = append(qs, map[string]any{"question_sid": s, "question": q["question"], "category": q["category"]})
			}
			o["questions"] = qs
		} else if _, ok := o["questions"]; !ok {
			o["questions"] = []any{}
		}
	}
	return ""
}
