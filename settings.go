package fbot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// URL to send settings to;
// is relative to the API URL.
const settingsURL = "%s/me/messenger_profile?access_token=%s&appsecret_proof=%s"

// Greeting is a localized message describing the bot.
type Greeting struct {
	// Locale should be 5 chars, like "en_US".
	// For supported locales see: https://developers.facebook.com/docs/messenger-platform/messenger-profile/supported-locales
	Locale string `json:"locale"`
	// Text can be any text, but it's restricted to 160 chars.
	// Supports a few template strings, see: https://developers.facebook.com/docs/messenger-platform/messenger-profile/greeting-text
	Text string `json:"text"`
}

// SetGreetings sets the text displayed in the bot description.
// Include "default" locale as fallback for missing locales.
func (c Client) SetGreetings(g []Greeting) error {
	return c.postSetting(struct {
		Greeting []Greeting `json:"greeting,omitempty"`
	}{Greeting: g})
}

// SetGetStartedPayload displays a "Get Started" button for new users.
// When a users pushes the button, a postback with the given payload is triggered.
func (c Client) SetGetStartedPayload(p string) error {
	return c.postSetting(struct {
		GetStarted getStartedPayload `json:"get_started,omitempty"`
	}{GetStarted: getStartedPayload{p}})
}

// Helper to send settings to the settings endpoint.
func (c Client) postSetting(data interface{}) error {
	encoded, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("json of %#v: %v", data, err)
	}

	url := fmt.Sprintf(settingsURL, c.api, c.token, c.secretProof)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(encoded))
	if err != nil {
		return fmt.Errorf("post \"%s\" to %#v: %v", encoded, url, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	return checkError(resp.Body)
}

type getStartedPayload struct {
	Payload string `json:"payload,omitempty"`
}
