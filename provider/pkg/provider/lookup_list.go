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
)

// maxListPages bounds how many pages a list lookup reads before it fails loudly.
const maxListPages = 100

// listPages reads a paginated Twilio list. itemsKey names the array that holds the
// items on each page. visit gets every item in order and returns false to stop early
// (a caller-supplied limit). The 2010 API continues through next_page_uri (a path
// with its query), the newer APIs through meta.next_page_url (a full URL); both are
// followed. A list that is still continuing after maxListPages pages is an error, so a
// result is never silently truncated.
func listPages(ctx context.Context, c *Client, fn, product, path string, query url.Values, itemsKey string, visit func(obj) bool) error {
	for page := 0; page < maxListPages; page++ {
		o, err := c.do(ctx, "GET", product, path, query, nil)
		if err != nil {
			return err
		}
		items, _ := o[itemsKey].([]any)
		for _, it := range items {
			m, ok := it.(map[string]any)
			if !ok {
				continue
			}
			if !visit(obj(m)) {
				return nil
			}
		}
		if next := o.str("next_page_uri"); next != "" {
			path, query = next, nil
			continue
		}
		next := o.str("meta.next_page_url")
		if next == "" {
			return nil
		}
		u, err := url.Parse(next)
		if err != nil {
			return fmt.Errorf("parse next page URL: %w", err)
		}
		query = u.Query()
	}
	return fmt.Errorf("%s: more than %d pages of results; narrow the query", fn, maxListPages)
}

// floatOf reads a JSON number that Twilio sometimes sends as a string.
func floatOf(o obj, k string) *float64 {
	switch v := o.get(k).(type) {
	case float64:
		return &v
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return &f
		}
	}
	return nil
}

// limitOrDefault validates an optional result limit.
func limitOrDefault(fn string, p *int, def, max int) (int, error) {
	if p == nil {
		return def, nil
	}
	if *p < 1 || *p > max {
		return 0, fmt.Errorf("%s: limit must be between 1 and %d", fn, max)
	}
	return *p, nil
}
