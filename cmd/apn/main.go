package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/pubsub"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/joonix/apn"
	"github.com/joonix/apn/proto"
	logtools "github.com/joonix/log"
)

// ttl defines the threshold of what is considered a too old message for delivery.
const ttl = 10 * time.Minute

func main() {
	project := flag.String("project", "", "project that we belong to")
	bundleID := flag.String("bundleID", "", "bundle ID of the APN application")
	notificationTopic := flag.String("notification_topic", "notifications", "which topic to consume notifications from")
	pub := flag.String("cert", "/secrets/cert.pem", "public client certificate used for APN auth")
	key := flag.String("key", "/secrets/key.pem", "private client certificate used for APN auth")
	lvl := flag.String("level", log.DebugLevel.String(), "log level")
	flag.Parse()

	level, err := log.ParseLevel(*lvl)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	log.SetLevel(level)
	log.SetFormatter(&logtools.FluentdFormatter{})

	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, *project)
	if err != nil {
		log.Fatal(err)
	}

	topic, err := client.CreateTopic(ctx, *notificationTopic)
	if err != nil {
		log.Debug(err)
	}
	if topic == nil {
		topic = client.Topic(*notificationTopic)
	}

	subscriberName := *notificationTopic + "-apn"
	client.CreateSubscription(ctx, subscriberName, topic, time.Minute, nil)
	it, err := client.Subscription(subscriberName).Pull(ctx)
	if err != nil {
		log.Fatal(err)
	}

	send, err := apn.NewNotificationProvider(*pub, *key, *bundleID)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("listening for messages")
	for {
		msg, err := it.Next()
		if err != nil {
			log.Fatal(err)
		}

		var notification proto.Notification
		if err = json.Unmarshal(msg.Data, &notification); err != nil {
			log.Error(err)
			continue
		}

		// Filter messages that have become too old on the inbound queue.
		if time.Since(msg.PublishTime) > ttl {
			log.Debug("skipping message that was expired before processing")
			msg.Done(true)
			continue
		}
		// Send the notification with an expiry set for the outbound queue.
		if _, err = send(&notification, msg.Attributes["to"], msg.Attributes["identifier"], msg.PublishTime.Add(ttl)); err != nil {
			log.Error(err)
			continue
		}
		msg.Done(true)
		log.Debug("notification sent: ", notification, msg.Attributes)
	}
}
