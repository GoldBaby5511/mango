package dingding

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type Webhook struct {
	AccessToken string
	Secret      string
	EnableAt    bool
	AtAll       bool
}

// SendMessage Function to send message
//goland:noinspection GoUnhandledErrorResult
func (t *Webhook) SendTextMessage(s string, at ...string) error {
	msg := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": s,
		},
	}
	if t.EnableAt {
		if t.AtAll {
			if len(at) > 0 {
				return errors.New("the parameter \"AtAll\" is \"true\", but the \"at\" parameter of SendMessage is not empty")
			}
			msg["at"] = map[string]interface{}{
				"isAtAll": t.AtAll,
			}
		} else {
			msg["at"] = map[string]interface{}{
				"atMobiles": at,
				"isAtAll":   t.AtAll,
			}
		}
	} else {
		if len(at) > 0 {
			return errors.New("the parameter \"EnableAt\" is \"false\", but the \"at\" parameter of SendMessage is not empty")
		}
	}
	return t.sendMessage(msg)
}

func (t *Webhook) SendMarkdownMessage(s string, at ...string) error {
	msg := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": "告警消息",
			"text":  s,
		},
	}
	if t.EnableAt {
		if t.AtAll {
			if len(at) > 0 {
				return errors.New("the parameter \"AtAll\" is \"true\", but the \"at\" parameter of SendMessage is not empty")
			}
			msg["at"] = map[string]interface{}{
				"isAtAll": t.AtAll,
			}
		} else {
			msg["at"] = map[string]interface{}{
				"atMobiles": at,
				"isAtAll":   t.AtAll,
			}
		}
	} else {
		if len(at) > 0 {
			return errors.New("the parameter \"EnableAt\" is \"false\", but the \"at\" parameter of SendMessage is not empty")
		}
	}
	return t.sendMessage(msg)
}

func (t *Webhook) sendMessage(msg map[string]interface{}) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	resp, err := http.Post(t.getURL(), "application/json", bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func (t *Webhook) hmacSha256(stringToSign string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (t *Webhook) getURL() string {
	wh := "https://oapi.dingtalk.com/robot/send?access_token=" + t.AccessToken
	timestamp := time.Now().UnixNano() / 1e6
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, t.Secret)
	sign := t.hmacSha256(stringToSign, t.Secret)
	url := fmt.Sprintf("%s&timestamp=%d&sign=%s", wh, timestamp, sign)
	return url
}
