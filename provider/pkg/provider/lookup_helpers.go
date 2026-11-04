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
