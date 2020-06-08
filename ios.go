package apn

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/joonix/apn/proto"
	"golang.org/x/net/http2"
)

// NotificationProvider defines the necessary parameters for sending a notification.
type NotificationProvider func(msg proto.NotificationPayload, token, identifier string, expiration time.Time) (*http.Response, error)

// NewNotificationProvider prepares an Apple APN provider for sending push notifications.
// Read more about it in the Apple documentation https://goo.gl/ywkRfD
func NewNotificationProvider(pub, key, topic, server string) (NotificationProvider, error) {
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

	// https://developer.apple.com/documentation/usernotifications/setting_up_a_remote_notification_server/sending_notification_requests_to_apns#2947607
	return func(msg proto.NotificationPayload, token, identifier string, expiration time.Time) (*http.Response, error) {
		var body bytes.Buffer
		if err := json.NewEncoder(&body).Encode(msg); err != nil {
			return nil, err
		}
		url := fmt.Sprintf("https://%s/3/device/%s", server, token)
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

		if msg.NotificationPayload().APS.ContentAvailable == 1 {
			req.Header.Set("apns-push-type", "background")
			req.Header.Set("apns-priority", "5")
		} else {
			req.Header.Set("apns-push-type", "alert")
			req.Header.Set("apns-priority", "10")
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

// HTTPError will be returned when the HTTP code suggest an error occured when sending notifications.
type HTTPError struct {
	Response *http.Response
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("invalid HTTP response (%d)", e.Response.StatusCode)
}
