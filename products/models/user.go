package models

type Role int

const (
	Admin Role = iota + 1
	Operator
	Guest
)

func (r Role) String() string {
	return []string{"admin", "operator", "guest"}[r-1]
}

type User struct {
	Email string `bson:"email" json:"email"`
	Role  Role   `bson:"role" json:"role"`
}
