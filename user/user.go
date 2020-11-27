package user

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	e "github.com/RomanLorens/logviewer/error"
	log "github.com/RomanLorens/logviewer/logger"
)

var (
	client = &http.Client{}
	cache  = make(map[string]*User)
)

//User user
type User struct {
	FirstName  string `json:"firstName"`
	LirstName  string `json:"lastName"`
	MiddleName string `json:"middleName"`
}

//Details get user details
func Details(ctx context.Context, ssoid string) (*User, *e.Error) {
	u, ok := cache[strings.ToLower(ssoid)]
	if ok {
		return u, nil
	}
	log.Info(ctx, "user %v not exists in  cache yet", ssoid)
	//todo url from config or use webservice...
	url := fmt.Sprintf("https://qa.citivelocity.com/portal-services/user/getUserByLoginId/%v", ssoid)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, e.Errorf(500, "error when creating req %v, %v", url, err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, e.Errorf(500, "error on %v, %v", url, err)
	}
	defer resp.Body.Close()
	var out User
	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, e.Errorf(400, "could not serialize %v, %v", resp, err)
	}
	if resp.StatusCode != 200 {
		return nil, e.Errorf(resp.StatusCode, "Could not get user from %v", url)
	}
	cache[strings.ToLower(ssoid)] = &out
	return &out, nil
}
