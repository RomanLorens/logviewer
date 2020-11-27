package auth

import (
	"net/http"
	"strings"
)

//UserAuth user auth details
type UserAuth struct {
	User  string   `json:"user"`
	Roles []string `json:"roles"`
}

//UserWithRoles user details
func UserWithRoles(r *http.Request) *UserAuth {
	user := UserFromRequest(r)
	roles := make([]string, 0)
	if user == "user" {
		roles = append(roles, "admin")
	} else {
		ra := r.RemoteAddr
		if strings.Contains(ra, "11.1.1.1") || strings.Contains(ra, "[::1]") {
			roles = append(roles, "admin")
		}
	}
	return &UserAuth{
		User:  user,
		Roles: roles,
	}
}

//UserFromRequest user
func UserFromRequest(r *http.Request) string {
	u := r.Header.Get("x-ssoid")
	if u == "" {
		u = r.Header.Get("x-LoginID")
	}
	if u == "" {
		u = "anonymous"
	}
	return strings.ToLower(u)
}
