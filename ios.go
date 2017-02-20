package apn

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/net/http2"
)

type HTTPError struct {
	Response *http.Response
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("invalid HTTP response (%d)", e.Response.StatusCode)
}

// https://developer.apple.com/library/prerelease/content/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/CommunicatingwithAPNs.html#//apple_ref/doc/uid/TP40008194-CH11-SW1
type NotificationProvider func(msg interface{}, token, identifier string, expiration time.Time) (*http.Response, error)

// NewNotificationProvider prepares an Apple APN provider for sending push notifications.
func NewNotificationProvider(pub, key, topic string) (NotificationProvider, error) {
	cert, err := tls.LoadX509KeyPair(pub, key)
	if err != nil {
		return nil, err
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	config.BuildNameToCertificate()

	tr := http.Transport{
		TLSClientConfig: config,
	}
	if err = http2.ConfigureTransport(&tr); err != nil {
		return nil, err
	}

	client := &http.Client{
		Transport: &tr,
	}

	return func(msg interface{}, token, identifier string, expiration time.Time) (*http.Response, error) {
		var body bytes.Buffer
		if err := json.NewEncoder(&body).Encode(msg); err != nil {
			return nil, err
		}
		url := fmt.Sprintf("https://api.development.push.apple.com/3/device/%s", token)
		req, err := http.NewRequest("POST", url, &body)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		var exp int64
		if !expiration.IsZero() {
			exp = expiration.Unix()
		}
		// collapse id max length is 64 bytes.
		if len(identifier) > 64 {
			identifier = fmt.Sprintf("%x", sha256.Sum256([]byte(identifier)))
		}
		req.Header.Set("apns-expiration", fmt.Sprintf("%d", exp))
		req.Header.Set("apns-collapse-id", identifier)
		req.Header.Set("apns-topic", topic)

		res, err := client.Do(req)
		if err != nil {
			return res, err
		}
		if code := res.StatusCode; code != http.StatusOK {
			err = HTTPError{res}
		}
		return res, err
	}, nil
}
