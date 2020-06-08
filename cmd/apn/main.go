package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
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
	server := flag.String("server", "api.development.push.apple.com", "which APN server to use")
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
	sub := client.Subscription(subscriberName)
	// Initialize health check endpoint, can be used for triggering a restart upon pubsub errors.
	http.Handle("/healthz", livenessProbe(sub))
	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	send, err := apn.NewNotificationProvider(*pub, *key, *bundleID, *server)
	if err != nil {
		log.Fatal(err)
	}
	it, err := sub.Pull(ctx)
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

// livenessProbe checks whether we can confirm that our subscription still exists.
func livenessProbe(sub *pubsub.Subscription) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		log.Debug("livenessProbe: checking that subscription exists")
		if ok, err := sub.Exists(ctx); !ok || err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			if err != nil {
				log.Error("livenessProbe: ", err)
			}
		}
	})
}
