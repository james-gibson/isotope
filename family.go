package isotope

import "regexp"

// familyTagRe matches a family-type declaration in a spec's header comment:
//
//	# family: deployment
//
// It is a comment, so the canon drops it — the tag marks a spec as family-typed
// without entering the isotope. The family VALUE is not read from the spec: the
// caller assigns it at canonicalization time (the family argument to
// CanonicalizeGherkinToIsotope). The tag declares the TYPE; the deployment
// assigns the value. That is the point of the split — a spec is written once,
// generically, and the same bytes are signed into different families by whoever
// deploys them.
var familyTagRe = regexp.MustCompile(`(?m)^[ \t]*#[ \t]*family[ \t]*:[ \t]*(\S+)[ \t]*$`)

// FamilyType returns the family type a spec declares in its header, or "" when
// the spec declares none.
func FamilyType(gherkin string) string {
	m := familyTagRe.FindStringSubmatch(gherkin)
	if m == nil {
		return ""
	}
	return m[1]
}
