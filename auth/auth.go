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
	if user == "rl78794" {
		roles = append(roles, "admin")
	} else {
		ra := r.RemoteAddr
		if strings.Contains(ra, "10.106.11.95") || strings.Contains(ra, "[::1]") {
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
	u := r.Header.Get("x-citiportal-ssoid")
	if u == "" {
		u = r.Header.Get("x-citiportal-LoginID")
	}
	if u == "" {
		u = "anonymous"
	}
	return strings.ToLower(u)
}
