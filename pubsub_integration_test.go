// +build integration

package apn

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"

	"github.com/joonix/apn/proto"
)

var (
	gcpProject string
	gcpTopic   string
)

func init() {
	flag.StringVar(&gcpProject, "project", "", "Google Cloud Platform project")
	flag.StringVar(&gcpTopic, "topic", "", "Pub/Sub notification topic")
}

func TestPubsubNotificationIntegration(t *testing.T) {
	msg := proto.Notification{
		APS: proto.NotificationAPS{
			Sound: "default",
			Alert: proto.NotificationAlert{
				Title: "TestPubsubNotification",
				Body:  "this notification was sent through the Pub/Sub topic",
			},
		},
	}

	token, err := base64.StdEncoding.DecodeString(tokenFlag)
	if err != nil {
		t.Fatal(err)
	}

	if err := publish(t, &msg, token); err != nil {
		t.Error(err.Error())
	}
}

// publish the notification message with target device token specification.
// Creates the notification topic if it doesn't already exist.
func publish(t *testing.T, msg *proto.Notification, token []byte) error {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, gcpProject)
	if err != nil {
		return err
	}
	if _, err := client.CreateTopic(ctx, gcpTopic); err != nil {
		t.Logf("error creating topic %q: %s", gcpTopic, err.Error())
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	pubsubTopic := client.Topic(gcpTopic)
	_, err = pubsubTopic.Publish(ctx, &pubsub.Message{
		Data: b,
		Attributes: map[string]string{
			"identifier": fmt.Sprintf("%x%s", token, time.Now().Truncate(time.Minute).String()),
			"to":         fmt.Sprintf("%x", token),
		},
	})
	t.Logf("Pubsub.Publish: message: %s", string(b))
	return nil
}
