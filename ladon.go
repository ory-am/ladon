package ladon

import (
	"regexp"

	"github.com/go-errors/errors"
	"github.com/ory-am/common/compiler"
)

// Ladon is an implementation of Warden.
type Ladon struct {
	Manager Manager
}

// IsAllowed returns nil if subject s has permission p on resource r with context c or an error otherwise.
func (g *Ladon) IsAllowed(r *Request) (err error) {
	policies, err := g.Manager.FindPoliciesForSubject(r.Subject)
	if err != nil {
		return errors.New(err)
	}

	return g.doPoliciesAllow(r, policies)
}

func (g *Ladon) doPoliciesAllow(r *Request, policies []Policy) (err error) {
	var allowed = false

	// Iterate through all policies
	for _, p := range policies {
		// Does the action match with one of the policies?
		if pm, err := Match(p, p.GetActions(), r.Action); err != nil {
			return errors.New(err)
		} else if !pm {
			// no, continue to next policy
			continue
		}

		// Does the subject match with one of the policies?
		if sm, err := Match(p, p.GetSubjects(), r.Subject); err != nil {
			return err
		} else if !sm && len(p.GetSubjects()) > 0 {
			// no, continue to next policy
			continue
		}

		// If no resource is given, the policy should not be scoped to resources as well.
		if r.Resource == "" && len(p.GetResources()) == 0 {
			// Does the resource match with one of the policies?
		} else if rm, err := Match(p, p.GetResources(), r.Resource); err != nil {
			return errors.New(err)
		} else if !rm {
			// no, continue to next policy
			continue
		}

		// Are the policies conditions met?
		if !g.passesConditions(p, r) {
			// no, continue to next policy
			continue
		}

		// Is the policies effect deny? If yes, this overrides all allow policies -> access denied.
		if !p.AllowAccess() {
			return errors.New(ErrForbidden)
		}
		allowed = true
	}

	if !allowed {
		return errors.New(ErrForbidden)
	}

	return nil
}

// Match matches a needle with an array of regular expressions and returns true if a match was found.
func Match(p Policy, haystack []string, needle string) (bool, error) {
	var reg *regexp.Regexp
	var err error
	var matches bool
	for _, h := range haystack {
		reg, err = compiler.CompileRegex(h, p.GetStartDelimiter(), p.GetEndDelimiter())
		if err != nil {
			return false, err
		}

		matches = reg.MatchString(needle)
		if matches {
			return true, nil
		}
	}
	return false, nil
}

func (g *Ladon) passesConditions(p Policy, r *Request) bool {
	for _, condition := range p.GetConditions() {
		if pass := condition.Fulfills(r); !pass {
			return false
		}
	}
	return true
}
