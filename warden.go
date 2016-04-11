package ladon

import (
	"regexp"

	"github.com/go-errors/errors"
	"github.com/ory-am/common/compiler"
)

type Request struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Subject  string `json:"subject"`
	Context  *Context
}

// Warden is responsible for deciding if subject s can perform action a on resource r with context c.
type Warden interface {
	// IsAllowed returns nil if subject s can perform action a on resource r with context c or an error otherwise.
	//  if err := guard.IsAllowed(&Request{Resource: "article/1234", Action: "update", Subject: "peter"}); err != nil {
	//    return errors.New("Not allowed")
	//  }
	IsAllowed(r *Request) error
}

// Ladon is an implementation of Warden.
type Ladon struct{
	Manager Manager
}

// IsGranted returns nil if subject s has permission p on resource r with context c or an error otherwise.
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
		// Does the resource match with one of the policies?
		if rm, err := Match(p, p.GetResources(), r.Resource); err != nil {
			return errors.New(err)
		} else if !rm {
			// no, continue to next policy
			continue
		}

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

		// Are the policies conditions met?
		if !g.passesConditions(p, r) {
			// no, continue to next policy
			continue
		}

		// Is the policies effect deny? If yes, this overrides all allow policies -> access denied.
		if !p.HasAccess() {
			return errors.New(ErrForbidden)
		}
		allowed = true
	}

	if !allowed {
		return  errors.New(ErrForbidden)
	}

	return nil
}

func Match(p Policy, patterns []string, match string) (bool, error) {
	var reg *regexp.Regexp
	var err error
	var matches bool
	for _, h := range patterns {
		reg, err = compiler.CompileRegex(h, p.GetStartDelimiter(), p.GetEndDelimiter())
		if err != nil {
			return false, err
		}

		matches = reg.MatchString(match)
		if matches {
			return true, nil
		}
	}
	return false, nil
}

func (g *Ladon) passesConditions(p Policy, r *Request) (bool) {
	for _, condition := range p.GetConditions() {
		if pass := condition.FullFills(r); !pass {
			return false
		}
	}
	return true
}
