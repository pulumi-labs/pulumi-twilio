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
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// readOnlyDefaults are Read defaults whose Diff does not go through a defaults table: nested
// objects and properties that are compared elsewhere. Every other Read default must have a Diff
// counterpart reading the same table entry (#1896), so an import that adopts a value cannot plan a
// change for a program that sets Twilio's default explicitly (#1880).
var readOnlyDefaults = map[string]bool{
	"conversationsAddressConfigurationDefaults/addressCountry":                      true,
	"conversationsAddressConfigurationDefaults/autoCreation.type":                   true,
	"conversationsAddressConfigurationDefaults/autoCreation.conversationServiceSid": true,
	"conversationsAddressConfigurationDefaults/autoCreation.webhookUrl":             true,
	"conversationsAddressConfigurationDefaults/autoCreation.webhookMethod":          true,
	"conversationsAddressConfigurationDefaults/autoCreation.studioFlowSid":          true,
	"conversationsAddressConfigurationDefaults/autoCreation.studioRetryCount":       true,
	"conversationsNotificationConfigurationDefaults/notification.enabled":           true,
	"conversationsNotificationConfigurationDefaults/notification.template":          true,
	"conversationsNotificationConfigurationDefaults/notification.sound":             true,
	"conversationsNotificationConfigurationDefaults/newMessage.badgeCountEnabled":   true,
	"conversationsNotificationConfigurationDefaults/newMessage.withMedia.enabled":   true,
	"conversationsNotificationConfigurationDefaults/newMessage.withMedia.template":  true,
	"taskrouterActivityDefaults/available":                                          true,
}

func TestDefaultsTablesAgree(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool { return !strings.HasSuffix(fi.Name(), "_test.go") }, 0)
	if err != nil {
		t.Fatal(err)
	}
	declared := map[string]bool{} // table/key
	read := map[string]bool{}
	diff := map[string]bool{}
	other := map[string]bool{}
	tableKey := func(e ast.Expr) (string, bool) {
		c, ok := e.(*ast.CallExpr)
		if !ok || len(c.Args) != 1 {
			return "", false
		}
		sel, ok := c.Fun.(*ast.SelectorExpr)
		if !ok || (sel.Sel.Name != "str" && sel.Sel.Name != "boolean" && sel.Sel.Name != "integer") {
			return "", false
		}
		id, ok := sel.X.(*ast.Ident)
		lit, ok2 := c.Args[0].(*ast.BasicLit)
		if !ok || !ok2 || !strings.HasSuffix(id.Name, "Defaults") {
			return "", false
		}
		k, _ := strconv.Unquote(lit.Value)
		return id.Name + "/" + k, true
	}
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.ValueSpec:
					for i, v := range x.Values {
						cl, ok := v.(*ast.CompositeLit)
						if !ok || i >= len(x.Names) {
							continue
						}
						if ty, ok := cl.Type.(*ast.Ident); !ok || ty.Name != "trDefaults" {
							continue
						}
						for _, el := range cl.Elts {
							kv := el.(*ast.KeyValueExpr)
							k, _ := strconv.Unquote(kv.Key.(*ast.BasicLit).Value)
							declared[x.Names[i].Name+"/"+k] = true
						}
					}
				case *ast.CallExpr:
					name := ""
					switch fn := x.Fun.(type) {
					case *ast.Ident:
						name = fn.Name
					case *ast.SelectorExpr:
						name = fn.Sel.Name
					}
					switch name {
					case "trStr", "trBool", "trInt":
						if len(x.Args) == 4 {
							if k, ok := tableKey(x.Args[2]); ok {
								read[k] = true
							}
						}
					case "strDef", "boolDef", "intDef":
						if len(x.Args) == 4 {
							id, ok := x.Args[0].(*ast.Ident)
							lit, ok2 := x.Args[1].(*ast.BasicLit)
							if ok && ok2 {
								k, _ := strconv.Unquote(lit.Value)
								diff[id.Name+"/"+k] = true
							}
						}
					default:
						// any other use of a table entry (for example derefOr in a Diff)
						for _, a := range x.Args {
							if k, ok := tableKey(a); ok {
								other[k] = true
							}
						}
					}
				}
				return true
			})
		}
	}
	var problems []string
	for k := range read {
		if !declared[k] {
			problems = append(problems, "Read uses undeclared default "+k)
		}
		if !diff[k] && !other[k] && !readOnlyDefaults[k] {
			problems = append(problems, "Read default "+k+" has no Diff counterpart: add the Diff default or list it in readOnlyDefaults")
		}
	}
	for k := range diff {
		if !declared[k] {
			problems = append(problems, "Diff uses undeclared default "+k)
		}
	}
	for k := range declared {
		if !read[k] && !diff[k] && !other[k] {
			problems = append(problems, "unused default "+k)
		}
	}
	for k := range readOnlyDefaults {
		if !read[k] || diff[k] || other[k] {
			problems = append(problems, "readOnlyDefaults entry "+k+" is stale (not read, or now has a Diff counterpart)")
		}
	}
	sort.Strings(problems)
	for _, p := range problems {
		t.Error(p)
	}
	if len(declared) < 80 {
		t.Errorf("found only %d table entries: the scan is not seeing the tables", len(declared))
	}
}
