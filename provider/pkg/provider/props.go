package provider

import (
	"net/url"
	"strconv"
)

// A resource's optional wire fields are described once as a props table; the
// same table drives the create/update form, the refresh from an API response
// and the clearing of fields removed from the program.

type strProp struct {
	param, key string
	p          **string
}

type boolProp struct {
	param, key string
	p          **bool
}

type intProp struct {
	param, key string
	p          **int
}

type props struct {
	strs  []strProp
	bools []boolProp
	ints  []intProp
}

// encode builds the form for a create or update. When prev is non-nil (an
// update), string fields that were set before but are now unset are sent as
// empty strings, which is how Twilio clears a URL or SID.
func (ps props) encode(v url.Values, prev *props) {
	for i, f := range ps.strs {
		switch {
		case *f.p != nil:
			v.Set(f.param, **f.p)
		case prev != nil && *prev.strs[i].p != nil && **prev.strs[i].p != "":
			v.Set(f.param, "")
		}
	}
	for _, f := range ps.bools {
		if *f.p != nil {
			v.Set(f.param, strconv.FormatBool(**f.p))
		}
	}
	for _, f := range ps.ints {
		if *f.p != nil {
			v.Set(f.param, strconv.Itoa(**f.p))
		}
	}
}

// refresh copies API values into the fields. With all=false only fields the
// program set are refreshed (drift detection without turning server defaults
// into permanent diffs); with all=true (import) every returned field is set.
func (ps props) refresh(o obj, all bool) {
	for _, f := range ps.strs {
		if all || *f.p != nil {
			*f.p = o.optStr(f.key)
		}
	}
	for _, f := range ps.bools {
		if all || *f.p != nil {
			*f.p = o.optBool(f.key)
		}
	}
	for _, f := range ps.ints {
		if all || *f.p != nil {
			*f.p = o.optInt(f.key)
		}
	}
}
