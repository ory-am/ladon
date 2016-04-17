package ladon

type Policies []Policy

// Policy represent a policy model.
type Policy interface {
	// GetID returns the policies id.
	GetID() string

	// GetDescription returns the policies description.
	GetDescription() string

	// GetSubjects returns the policies subjects.
	GetSubjects() []string

	// HasAccess returns true if the policy effect is allow, otherwise false.
	AllowAccess() bool

	// GetEffect returns the policies effect which might be 'allow' or 'deny'.
	GetEffect() string

	// GetResources returns the policies resources.
	GetResources() []string

	// GetActions returns the policies actions.
	GetActions() []string

	// GetConditions returns the policies conditions.
	GetConditions() Conditions

	// GetStartDelimiter returns the delimiter which identifies the beginning of a regular expression
	GetStartDelimiter() byte

	// GetEndDelimiter returns the delimiter which identifies the end of a regular expression
	GetEndDelimiter() byte
}

type DefaultPolicy struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	Subjects    []string   `json:"subjects"`
	Effect      string     `json:"effect"`
	Resources   []string   `json:"resources"`
	Actions     []string   `json:"actions"`
	Conditions  Conditions `json:"conditions"`
}

func (p *DefaultPolicy) GetID() string {
	return p.ID
}

func (p *DefaultPolicy) GetDescription() string {
	return p.Description
}

func (p *DefaultPolicy) GetSubjects() []string {
	return p.Subjects
}

func (p *DefaultPolicy) AllowAccess() bool {
	return p.Effect == AllowAccess
}

func (p *DefaultPolicy) GetEffect() string {
	return p.Effect
}

func (p *DefaultPolicy) GetResources() []string {
	return p.Resources
}

func (p *DefaultPolicy) GetActions() []string {
	return p.Actions
}

func (p *DefaultPolicy) GetConditions() Conditions {
	return p.Conditions
}

func (p *DefaultPolicy) GetEndDelimiter() byte {
	return '>'
}

func (p *DefaultPolicy) GetStartDelimiter() byte {
	return '<'
}
