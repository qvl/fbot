// Package fbot can be used to communicate with a Facebook Messenger bot.
// The supported API is limited to only features we use.
// Please open an issue for features you are missing.
package fbot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const defaultAPI = "https://graph.facebook.com/v2.10"

// Client can be used to communicate with a Messenger bot.
// Use New for initialization.
type Client struct {
	token       string
	secretProof string
	api         string
}

// Config is passed to New.
type Config struct {
	Token  string
	Secret string
	API    string // Optional. Overwrite the Facebook API URL.
}

// New rerturns a new client with credentials set up.
func New(c Config) Client {
	// Generate secret proof. See https://developers.facebook.com/docs/graph-api/securing-requests/#appsecret_proof
	mac := hmac.New(sha256.New, []byte(c.Secret))
	mac.Write([]byte(c.Token))

	api := strings.TrimSuffix(c.API, "/")
	if api == "" {
		api = defaultAPI
	}

	return Client{
		token:       c.Token,
		secretProof: hex.EncodeToString(mac.Sum(nil)),
		api:         api,
	}
}

// Button describes a button that can be send with a Button Template.
// Use URLButton or PayloadButton for initialization.
type Button interface{}

// Reply describes a text quick reply.
type Reply struct {
	// Text is the text on the button visible to the user
	Text string
	// Payload is a string to identify the quick reply event internally in your application.
	Payload string
}

// Helper to check for errors in reply
func checkError(r io.Reader) error {
	var qr struct {
		Error *struct {
			Message   string `json:"message"`
			Type      string `json:"type"`
			Code      int    `json:"code"`
			FBTraceID string `json:"fbtrace_id"`
		} `json:"error"`
		Result string `json:"result"`
	}

	err := json.NewDecoder(r).Decode(&qr)
	if qr.Error != nil {
		err = fmt.Errorf("Facebook error : %s", qr.Error.Message)
	}
	return err
}
