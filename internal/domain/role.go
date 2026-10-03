package domain

type Role string

const (
	RoleClient Role = "client"
	RoleAdmin  Role = "admin"
)

func (r Role) IsValid() bool {
	return r == RoleClient || r == RoleAdmin
}
