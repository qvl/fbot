package fbot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// URL to fetch the profile from;
// is relative to the API URL.
const profileURL = "%s/%d?fields=first_name,locale,timezone&access_token=%s&appsecret_proof=%s"

// Profile has all public user information we need;
// needs to be in sync with the URL above.
type Profile struct {
	data struct {
		Name     string  `json:"first_name"`
		Locale   string  `json:"locale"`
		Timezone float64 `json:"timezone"`
	}
}

// Name returns the first name.
func (p Profile) Name() string {
	return p.data.Name
}

// Locale returns th locale in the form "en_GB".
func (p Profile) Locale() string {
	return p.data.Locale
}

// Timezone retunrs the timezone relative to UTC.
func (p Profile) Timezone() int {
	return p.data.Timezone
}

// GetProfile fetches a user profile for an ID.
func (c Client) GetProfile(id int64) (Profile, error) {
	var p Profile
	url := fmt.Sprintf(profileURL, c.api, id, c.token, c.secretProof)
	resp, err := http.Get(url)
	if err != nil {
		return p, fmt.Errorf("get %#v: %v", url, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return p, fmt.Errorf("failed to read body: %v", err)
	}

	if err = json.Unmarshal(content, &p.data); err != nil {
		return p, fmt.Errorf("failed to parse json from \"%s\": %v", content, err)
	}

	return p, checkError(bytes.NewReader(content))
}
