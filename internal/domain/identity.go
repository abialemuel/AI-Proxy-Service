package domain

// Principal represents the authenticated caller.
// Either an end-user (via SSO) or a backend service (via shared secret).
type Principal struct {
	Kind     PrincipalKind
	Subject  string // email for users, service-name for services
	Tribe    string
	Username string // for service principals
}

type PrincipalKind string

const (
	PrincipalUser    PrincipalKind = "user"
	PrincipalService PrincipalKind = "service"
)
