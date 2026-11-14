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
	"encoding/json"
	"net/http"
	"net/url"
)

// The Flex Plugins service of the fake: Plugins, their Versions, Configurations (with the plugin
// versions they load) and Releases. Plugins, Versions and Configurations archive instead of delete.

var flexPluginFields = []fakeField{{"FriendlyName", "friendly_name", "s"}, {"Description", "description", "s"}}

func (f *fakeTwilio) flexPlugins(w http.ResponseWriter, r *http.Request, form url.Values, parts []string) {
	notFound := func() { ferr(w, 404, "The requested resource "+r.URL.Path+" was not found") }
	if len(parts) == 0 {
		notFound()
		return
	}
	coll := parts[0]
	base := "https://flex-api.twilio.com/v1/PluginService/"
	missing := func(names ...string) bool {
		for _, n := range names {
			if form.Get(n) == "" {
				ferr(w, 400, "Missing required parameter "+n)
				return true
			}
		}
		return false
	}
	archive := func(kind, sid string) {
		o, ok := f.objs[key("flex", testAccount, kind+"/"+sid)]
		if !ok {
			notFound()
			return
		}
		o["archived"] = true
		fwrite(w, 200, o)
	}
	switch {
	// Plugin versions: Plugins/{sid}/Versions[/{v}[/Archive]]
	case coll == "Plugins" && len(parts) >= 3 && parts[2] == "Versions":
		plugin := parts[1]
		if _, ok := f.objs[key("flex", testAccount, "Plugins/"+plugin)]; !ok {
			notFound()
			return
		}
		if len(parts) == 3 {
			switch r.Method {
			case "POST":
				if missing("Version", "PluginUrl") {
					return
				}
				id := f.sid("FV")
				o := map[string]any{"sid": id, "plugin_sid": plugin, "account_sid": testAccount, "version": form.Get("Version"),
					"plugin_url": form.Get("PluginUrl"), "changelog": nil, "private": false, "archived": false, "validated": false,
					"date_created": cvStamp, "url": base + "Plugins/" + plugin + "/Versions/" + id}
				if msg := applyForm(o, form, []fakeField{{"Changelog", "changelog", "s"}, {"Private", "private", "b"}}); msg != "" {
					ferr(w, 400, msg)
					return
				}
				f.put("flex", testAccount, "Versions/"+id, o)
				fwrite(w, 201, o)
			case "GET":
				metaPage(w, r, "plugin_versions", f.flexList("Versions", func(o map[string]any) bool { return o["plugin_sid"] == plugin }), 2)
			default:
				ferr(w, 405, "method not allowed")
			}
			return
		}
		o, ok := f.objs[key("flex", testAccount, "Versions/"+parts[3])]
		if !ok || o["plugin_sid"] != plugin {
			notFound()
			return
		}
		switch {
		case len(parts) == 4 && r.Method == "GET":
			fwrite(w, 200, o)
		case len(parts) == 5 && parts[4] == "Archive" && r.Method == "POST":
			archive("Versions", parts[3])
		default:
			notFound()
		}
	case coll == "Plugins" && len(parts) == 1:
		switch r.Method {
		case "POST":
			if missing("UniqueName") {
				return
			}
			for _, o := range f.flexList("Plugins", nil) {
				if o["unique_name"] == form.Get("UniqueName") {
					ferr(w, 409, "Plugin with the unique name already exists")
					return
				}
			}
			id := f.sid("FP")
			o := map[string]any{"sid": id, "account_sid": testAccount, "unique_name": form.Get("UniqueName"), "friendly_name": nil,
				"description": nil, "archived": false, "date_created": cvStamp, "date_updated": cvStamp, "url": base + "Plugins/" + id}
			if msg := applyForm(o, form, flexPluginFields); msg != "" {
				ferr(w, 400, msg)
				return
			}
			f.put("flex", testAccount, "Plugins/"+id, o)
			fwrite(w, 201, o)
		case "GET":
			metaPage(w, r, "plugins", f.flexList("Plugins", nil), 2)
		default:
			ferr(w, 405, "method not allowed")
		}
	case coll == "Plugins" && len(parts) >= 2:
		o, ok := f.objs[key("flex", testAccount, "Plugins/"+parts[1])]
		if !ok {
			notFound()
			return
		}
		switch {
		case len(parts) == 2 && r.Method == "GET":
			fwrite(w, 200, o)
		case len(parts) == 2 && r.Method == "POST":
			if msg := applyForm(o, form, flexPluginFields); msg != "" {
				ferr(w, 400, msg)
				return
			}
			o["date_updated"] = cvUpdatedStamp
			fwrite(w, 200, o)
		case len(parts) == 3 && parts[2] == "Archive" && r.Method == "POST":
			archive("Plugins", parts[1])
		default:
			notFound()
		}
	case coll == "Configurations" && len(parts) == 1:
		switch r.Method {
		case "POST":
			if missing("Name") {
				return
			}
			id := f.sid("FJ")
			o := map[string]any{"sid": id, "account_sid": testAccount, "name": form.Get("Name"), "description": nil, "archived": false,
				"date_created": cvStamp, "url": base + "Configurations/" + id}
			if msg := applyForm(o, form, []fakeField{{"Description", "description", "s"}}); msg != "" {
				ferr(w, 400, msg)
				return
			}
			var versions []any
			for i, raw := range form["Plugins"] {
				var p map[string]string
				if err := json.Unmarshal([]byte(raw), &p); err != nil || p["plugin_version"] == "" {
					ferr(w, 400, "Invalid Plugins entry "+raw)
					return
				}
				vo, ok := f.objs[key("flex", testAccount, "Versions/"+p["plugin_version"])]
				if !ok {
					ferr(w, 400, "Plugin version "+p["plugin_version"]+" does not exist")
					return
				}
				versions = append(versions, map[string]any{"configuration_sid": id, "plugin_sid": vo["plugin_sid"], "plugin_version_sid": p["plugin_version"],
					"phase": float64(i + 3), "version": vo["version"], "plugin_url": vo["plugin_url"]})
			}
			o["_plugins"] = versions
			f.put("flex", testAccount, "Configurations/"+id, o)
			fwrite(w, 201, o)
		case "GET":
			metaPage(w, r, "configurations", f.flexList("Configurations", nil), 2)
		default:
			ferr(w, 405, "method not allowed")
		}
	case coll == "Configurations" && len(parts) >= 2:
		o, ok := f.objs[key("flex", testAccount, "Configurations/"+parts[1])]
		if !ok {
			notFound()
			return
		}
		switch {
		case len(parts) == 2 && r.Method == "GET":
			fwrite(w, 200, o)
		case len(parts) == 3 && parts[2] == "Plugins" && r.Method == "GET":
			items, _ := o["_plugins"].([]any)
			var maps []map[string]any
			for _, it := range items {
				maps = append(maps, it.(map[string]any))
			}
			metaPage(w, r, "plugins", maps, 2)
		case len(parts) == 3 && parts[2] == "Archive" && r.Method == "POST":
			if f.flexReleased(parts[1]) {
				ferr(w, 400, "Configuration "+parts[1]+" is in use by a release")
				return
			}
			archive("Configurations", parts[1])
		default:
			notFound()
		}
	case coll == "Releases" && len(parts) == 1:
		switch r.Method {
		case "POST":
			if missing("ConfigurationId") {
				return
			}
			co, ok := f.objs[key("flex", testAccount, "Configurations/"+form.Get("ConfigurationId"))]
			if !ok {
				ferr(w, 400, "Configuration "+form.Get("ConfigurationId")+" does not exist")
				return
			}
			id := f.sid("FK")
			o := map[string]any{"sid": id, "account_sid": testAccount, "configuration_sid": co["sid"], "date_created": cvStamp, "url": base + "Releases/" + id}
			f.put("flex", testAccount, "Releases/"+id, o)
			fwrite(w, 201, o)
		case "GET":
			metaPage(w, r, "releases", f.flexList("Releases", nil), 2)
		default:
			ferr(w, 405, "method not allowed")
		}
	case coll == "Releases" && len(parts) == 2 && r.Method == "GET":
		o, ok := f.objs[key("flex", testAccount, "Releases/"+parts[1])]
		if !ok {
			notFound()
			return
		}
		fwrite(w, 200, o)
	default:
		notFound()
	}
}

// flexReleased reports whether a Release made the Configuration live.
func (f *fakeTwilio) flexReleased(configuration string) bool {
	for _, o := range f.flexList("Releases", nil) {
		if o["configuration_sid"] == configuration {
			return true
		}
	}
	return false
}
