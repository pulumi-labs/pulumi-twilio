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

import "fmt"

// lookupFailure turns a client error of a lookup function into the message the
// user sees: a 404 becomes "<what> was not found".
func lookupFailure(fn, what string, err error) error {
	if isNotFound(err) {
		return fmt.Errorf("%s: %s was not found", fn, what)
	}
	return fmt.Errorf("%s: %s: %w", fn, what, err)
}

// requireArg rejects an empty required lookup argument.
func requireArg(fn, name, v string) error {
	if v == "" {
		return fmt.Errorf("%s: %s must be set", fn, name)
	}
	return nil
}
