package fbot

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// EventType helps to distinguish the different type of events.
type EventType int

const (
	// EventError is triggered when the webhook is called with invalid JSON content.
	EventError EventType = 1 + iota
	// EventMessage is triggered when a user sends Text, stickers or other content.
	// Only text is available at the moment.
	EventMessage
	// EventPayload is triggered when a quickReply or postback Payload is sent.
	EventPayload
	// EventRead is triggered when a user read a message.
	EventRead
	// EventAttachment is triggered when attachemnts are send.
	EventAttachment
	// EventReferral is triggered when referring through a link or other source.
	EventReferral
)

// Event contains information about a user action.
type Event struct {
	// Type helps to decide how to react to an event.
	Type EventType
	// ChatID identifies the user. It's a Facebook user ID.
	ChatID int64
	// Time describes when the event occured.
	Time time.Time
	// Text is a message a user send for EventMessage and error description for EventError.
	Text string
	// Payload is a predefined payload for a quick reply or postback sent with EventPayload.
	Payload string
	// MessageID is a unique ID for each message.
	MessageID string
	// Attachments are multiple attachment types.
	Attachments []Attachment
	// Ref contains the ref data from the URL for EventReferral.
	// Ref is also set for EventPayload if the Event was triggered through the Get Started button
	// and the user used a refferal link to get there.
	Ref string
}

// Attachment describes an attachment.
// Type is one of "image", "video", audio, "location", "file" or "feedback".
// Currently only the URL field is loaded because we only use "file".
// If a sticker is sent the type is "image" and Sticker != 0.
// For more see: https://developers.facebook.com/docs/messenger-platform/webhook-reference/message
type Attachment struct {
	Type    string
	URL     string
	Sticker int64
}

// Webhook returns a handler for HTTP requests that can be registered with Facebook.
// The passed event handler will be called with all received events.
func (c Client) Webhook(handler func(Event), secret, verifyToken string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if r.FormValue("hub.verify_token") == verifyToken {
				fmt.Fprintln(w, r.FormValue("hub.challenge"))
				return
			}
			fmt.Fprintln(w, "Incorrect verify token.")
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			handler(Event{Type: EventError, Text: fmt.Sprintf("method not allowed: %s", r.Method)})
			return
		}

		// Read body
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "unable to read body", http.StatusInternalServerError)
			handler(Event{Type: EventError, Text: fmt.Sprintf("unable to read body: %v", err)})
			return
		}

		// Authenticate using header
		signature := r.Header.Get("X-Hub-Signature")
		if !validSignature(signature, secret, data) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			handler(Event{Type: EventError, Text: fmt.Sprintf("invalid signature header: %#v", signature)})
			return
		}

		// Parse JSON
		var rec receive
		if err := json.Unmarshal(data, &rec); err != nil {
			http.Error(w, "JSON invalid", http.StatusBadRequest)
			handler(Event{Type: EventError, Text: fmt.Sprintf("invalid JSON \"%s\": %v", data, err)})
			return
		}
		_ = r.Body.Close()

		// Return response as soon as possible.
		// Facebook doesn't care about the event handling.
		// Responses are sent separatly.
		fmt.Fprintln(w, `{"status":"ok"}`)

		for _, e := range rec.Entry {
			for _, m := range e.Messaging {
				if event := getEvent(m); event.Type != 0 {
					handler(event)
				}
			}
		}
	})
}

// Expects a signature of the form "sha1=xxx".
// Generates the sha1 sum for the given secret and data.
// Checks equality with constant timing to prevent timing attacks.
func validSignature(signature, secret string, data []byte) bool {
	// Remove " sha1=" from header, compute sha1 of secret+body, compare them
	if len(signature) <= 5 {
		return false
	}
	sum, err := hex.DecodeString(signature[5:])
	if err != nil {
		return false
	}
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(data)
	return hmac.Equal(sum, mac.Sum(nil))
}

func getEvent(m messageInfo) Event {
	if m.Postback != nil {
		ref := ""
		if m.Postback.Referral != nil {
			ref = m.Postback.Referral.Ref
		}
		return Event{
			Type:    EventPayload,
			ChatID:  m.Sender.ID,
			Time:    msToTime(m.Timestamp),
			Payload: m.Postback.Payload,
			Ref:     ref,
		}
	}
	if m.Read != nil {
		return Event{
			Type:   EventRead,
			ChatID: m.Sender.ID,
			Time:   msToTime(m.Read.Watermark),
		}
	}
	if m.Referral != nil {
		return Event{
			Type:   EventReferral,
			ChatID: m.Sender.ID,
			Time:   msToTime(m.Timestamp),
			Ref:    m.Referral.Ref,
		}
	}
	if m.Message != nil {
		if m.Message.IsEcho {
			return Event{}
		}
		if m.Message.QuickReply != nil {
			return Event{
				Type:    EventPayload,
				ChatID:  m.Sender.ID,
				Time:    msToTime(m.Timestamp),
				Payload: m.Message.QuickReply.Payload,
			}
		}
		if m.Message.Attachments != nil {
			var as []Attachment
			for _, a := range m.Message.Attachments {
				if a.Type == "fallback" {
					as = append(as, Attachment{
						Type:    a.Type,
						URL:     a.URL,
						Sticker: a.Payload.Sticker,
					})
				} else {
					as = append(as, Attachment{
						Type:    a.Type,
						URL:     a.Payload.URL,
						Sticker: a.Payload.Sticker,
					})
				}
			}
			return Event{
				Type:        EventAttachment,
				ChatID:      m.Sender.ID,
				Time:        msToTime(m.Timestamp),
				MessageID:   m.Message.MID,
				Attachments: as,
			}
		}
		return Event{
			Type:      EventMessage,
			ChatID:    m.Sender.ID,
			Time:      msToTime(m.Timestamp),
			Text:      m.Message.Text,
			MessageID: m.Message.MID,
		}
	}
	return Event{}
}

func msToTime(ms int64) time.Time {
	return time.Unix(ms/int64(time.Microsecond), 0)
}

type receive struct {
	Entry []struct {
		Messaging []messageInfo `json:"messaging"`
	} `json:"entry"`
}

type messageInfo struct {
	Sender struct {
		ID int64 `json:"id,string"`
	} `json:"sender"`
	Timestamp int64 `json:"timestamp"`
	Message   *struct {
		IsEcho      bool        `json:"is_echo,omitempty"`
		Text        string      `json:"text"`
		QuickReply  *quickReply `json:"quick_reply,omitempty"`
		MID         string      `json:"mid,omitempty"`
		Attachments []struct {
			Type    string `json:"type,omitempty"`
			Payload struct {
				Sticker int64  `json:"sticker_id,omitempty"`
				URL     string `json:"url,omitempty"`
			} `json:"payload,omitempty"`
			// used by fallback
			URL string `json:"url,omitempty"`
		} `json:"attachments,omitempty"`
	} `json:"message"`
	Postback *struct {
		Payload  string    `json:"payload"`
		Referral *referral `json:"referral"`
	} `json:"postback"`
	Read *struct {
		Watermark int64 `json:"watermark"`
	} `json:"read"`
	Referral *referral `json:"referral"`
}

// For now Source and Type are ignored.
// The can be used later to handle ADS and other sources.
// https://developers.facebook.com/docs/messenger-platform/referral-params
type referral struct {
	Ref    string `json:"ref"`
	Source string `json:"source"`
	Type   string `json:"type"`
}
