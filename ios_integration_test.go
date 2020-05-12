// +build integration

package apn

import (
	"encoding/base64"
	"flag"
	"fmt"
	"net/http/httputil"
	"testing"
	"time"

	"github.com/joonix/apn/proto"
)

var (
	tokenFlag string
	certPath  string
	keyPath   string
	bundleID  string
)

func init() {
	flag.StringVar(&bundleID, "bundle", "", "app bundle ID or APN topic")
	flag.StringVar(&tokenFlag, "token", "", "notification token of destination device")
	flag.StringVar(&certPath, "cert", "cert.pem", "path of public key used for APN authentication")
	flag.StringVar(&keyPath, "key", "key.pem", "path of private key used for APN authentication")
}

// https://developer.apple.com/library/prerelease/content/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/CreatingtheNotificationPayload.html#//apple_ref/doc/uid/TP40008194-CH10-SW1
func TestIOSNotificationIntegration(t *testing.T) {
	msg := proto.Notification{
		APS: proto.NotificationAPS{
			Alert: proto.NotificationAlert{
				Title: "This is a notification title",
				Body:  "This is the notification body",
			},
			Category: "FOOBAR_CATEGORY",
			Badge:    1,
			Sound:    "default",
		},
	}
	token, err := base64.StdEncoding.DecodeString(tokenFlag)
	if err != nil {
		t.Fatal(err)
	}

	send, err := NewNotificationProvider(certPath, keyPath, bundleID)
	if err != nil {
		t.Fatal(err)
	}
	identity := "randomuuid1" + time.Now().String()
	if _, err = send(&msg, fmt.Sprintf("%x", token), identity, time.Now().Add(time.Minute)); err != nil {
		switch err := err.(type) {
		case HTTPError:
			b, _ := httputil.DumpResponse(err.Response, true)
			t.Log(string(b))
		}
		t.Fatal(err)
	}
}

func TestIOSBackgroundUpdateNotificationIntegration(t *testing.T) {
	msg := proto.Notification{
		APS: proto.NotificationAPS{
			ContentAvailable: 1,
		},
	}
	token, err := base64.StdEncoding.DecodeString(tokenFlag)
	if err != nil {
		t.Fatal(err)
	}

	send, err := NewNotificationProvider(certPath, keyPath, bundleID)
	if err != nil {
		t.Fatal(err)
	}
	identity := "randomuuid1" + time.Now().String()
	if _, err = send(&msg, fmt.Sprintf("%x", token), identity, time.Now().Add(time.Minute)); err != nil {
		switch err := err.(type) {
		case HTTPError:
			b, _ := httputil.DumpResponse(err.Response, true)
			t.Log(string(b))
		}
		t.Fatal(err)
	}
}

type CustomNotification struct {
	proto.Notification
	FooField string `json:"foo_field"`
}

func (n CustomNotification) NotificationPayload() proto.Notification {
	return n.Notification
}

func TestIOSCustomNotificationIntegration(t *testing.T) {
	msg := CustomNotification{
		Notification: proto.Notification{
			APS: proto.NotificationAPS{
				Alert: proto.NotificationAlert{
					Title: "This is a notification title",
					Body:  "This is the notification body",
				},
				Category: "FOOBAR_CUSTOM",
				Badge:    1,
				Sound:    "default",
			},
		},
		FooField: "My custom foo data",
	}
	token, err := base64.StdEncoding.DecodeString(tokenFlag)
	if err != nil {
		t.Fatal(err)
	}

	send, err := NewNotificationProvider(certPath, keyPath, bundleID)
	if err != nil {
		t.Fatal(err)
	}
	identity := "randomuuid2" + time.Now().String()
	if _, err = send(&msg, fmt.Sprintf("%x", token), identity, time.Now().Add(time.Minute)); err != nil {
		switch err := err.(type) {
		case HTTPError:
			b, _ := httputil.DumpResponse(err.Response, true)
			t.Log(string(b))
		}
		t.Fatal(err)
	}
}
