package user

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/RomanLorens/logviewer/config"
	l "github.com/RomanLorens/logviewer/logger"
)

var (
	client = &http.Client{}
	cache  = make(map[string]*User)
	logger = l.L
)

//User user
type User struct {
	FirstName  string `json:"firstName"`
	LirstName  string `json:"lastName"`
	MiddleName string `json:"middleName"`
}

//Details get user details
func Details(ctx context.Context, ssoid string) (*User, error) {
	u, ok := cache[strings.ToLower(ssoid)]
	if ok {
		return u, nil
	}
	logger.Info(ctx, "user %v not exists in  cache yet", ssoid)
	url := fmt.Sprintf(config.Config.UserByLoginIDURL, ssoid)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error when creating req %v, %v", url, err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error on %v, %v", url, err)
	}
	defer resp.Body.Close()
	var out User
	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("could not serialize %v, %v", resp, err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Could not get user from %v", url)
	}
	cache[strings.ToLower(ssoid)] = &out
	return &out, nil
}
