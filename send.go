package fbot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// URL to send messages to;
// is relative to the API URL.
const sendMessageURL = "%s/me/messages?access_token=%s&appsecret_proof=%s"

// Send a text message with a set of quick reply buttons to a user.
func (c Client) Send(id int64, message string, replies []Reply) error {
	return c.send(id, struct {
		Text         string       `json:"text"`
		QuickReplies []quickReply `json:"quick_replies,omitempty"`
	}{
		Text:         message,
		QuickReplies: parseReplies(replies),
	})
}

// SendWithButtons sends a message with a set of buttons and quick replies to a user.
func (c Client) SendWithButtons(id int64, message string, replies []Reply, buttons []Button) error {
	return c.send(id, struct {
		Attachment   buttonAttachment `json:"attachment"`
		QuickReplies []quickReply     `json:"quick_replies,omitempty"`
	}{
		Attachment: buttonAttachment{
			Type: "template",
			Payload: buttonPayload{
				Type:    "button",
				Text:    message,
				Buttons: buttons,
			},
		},
		QuickReplies: parseReplies(replies),
	})
}

func (c Client) send(id int64, message interface{}) error {
	m := struct {
		Recipient recipient   `json:"recipient"`
		Message   interface{} `json:"message"`
	}{
		Recipient: recipient{ID: id},
		Message:   message,
	}
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("json of %#v: %v", m, err)
	}

	url := fmt.Sprintf(sendMessageURL, c.api, c.token, c.secretProof)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("post \"%s\" to %#v: %v", m, url, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode == 200 {
		return nil
	}
	return checkError(resp.Body)
}

func parseReplies(replies []Reply) []quickReply {
	var qs []quickReply
	for _, r := range replies {
		qs = append(qs, quickReply{
			ContentType: "text",
			Title:       r.Text,
			Payload:     r.Payload,
		})
	}
	return qs
}

// PayloadButton returns a button that posts a payload back to the bot.
func PayloadButton(text, payload string) Button {
	return button{
		Type:    "postback",
		Title:   text,
		Payload: payload,
	}
}

// URLButton returns a button that opens a URL in a full-screen webview.
func URLButton(text, url string) Button {
	return button{
		Type:        "web_url",
		Title:       text,
		URL:         url,
		ShareButton: "hide",
		Extensions:  true,
	}
}

// LinkButton returns a button that opens a URL in the browser.
func LinkButton(text, url string) Button {
	return button{
		Type:        "web_url",
		Title:       text,
		URL:         url,
		ShareButton: "hide",
		Extensions:  false,
	}
}

type buttonAttachment struct {
	Type    string        `json:"type"`
	Payload buttonPayload `json:"payload"`
}

type buttonPayload struct {
	Type    string   `json:"template_type"`
	Text    string   `json:"text"`
	Buttons []Button `json:"buttons"`
}

type button struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Payload     string `json:"payload,omitempty"`
	URL         string `json:"url,omitempty"`
	ShareButton string `json:"webview_share_button,omitempty"`
	Extensions  bool   `json:"messenger_extensions,omitempty"`
}

type recipient struct {
	ID int64 `json:"id,string"`
}

type quickReply struct {
	ContentType string `json:"content_type,omitempty"`
	Title       string `json:"title,omitempty"`
	Payload     string `json:"payload"`
}
