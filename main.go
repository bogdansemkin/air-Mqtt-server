package main

import (
	"context"
	"fmt"
	"net"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/DrmagicE/gmqtt"
	_ "github.com/DrmagicE/gmqtt/persistence"
	"github.com/DrmagicE/gmqtt/persistence/subscription"
	"github.com/DrmagicE/gmqtt/pkg/packets"
	"github.com/DrmagicE/gmqtt/server"
	_ "github.com/DrmagicE/gmqtt/topicalias/fifo"
)

type JsonStructure struct {
	TopicName string `json:"topic"`
	Qos string `json:"qos"`
	RetainHandling string `json:"retain_handling"`
	RetainAsPublished string `json:"retain_as_published"`
	NoLocal bool `json:"no_local"`
	Id int `json:"id"`
	ClientId string `json:"client_id"`
	RemoteAddr string `json:"remote_addr"`
}

//TODO create method for unparsing incoming json
func incomingDeviceJson() string{
	return ""
}

func main() {
	var testingLocalVariable string
	ln, err := net.Listen("tcp", ":7440")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	cfg := zap.NewDevelopmentConfig()
	cfg.Level.SetLevel(zap.InfoLevel)
	l, _ := cfg.Build()
	srv := server.New(
		server.WithTCPListener(ln),
		server.WithLogger(l),
	)

	var subService server.SubscriptionService
	err = srv.Init(server.WithHook(server.Hooks{
		OnConnected: func(ctx context.Context, client server.Client) {
			// add subscription for a client when it is connected

			var getString string
			getString = client.ClientOptions().ClientID

			fmt.Println("==================")
			fmt.Println("L is: " , getString)
			fmt.Println("==================")

			subService.Subscribe(client.ClientOptions().ClientID, &gmqtt.Subscription{
				TopicFilter: "topic",
				QoS:         packets.Qos0,

			})
		},
	}))
	subService = srv.SubscriptionService()

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	retainedService := srv.RetainedService()

	pub := srv.Publisher()

	retainedService.AddOrReplace(&gmqtt.Message{
		QoS:      packets.Qos1,
		Retained: true,
		Topic:    "a/b/c",
		Payload:  []byte("retained message"),
	})

	// publish service
	go func() {
		for {
			<-time.NewTimer(5 * time.Second).C
			// iterate all topics
			subService.Iterate(func(clientID string, sub *gmqtt.Subscription) bool {
				fmt.Printf("client id: %s, topic: %v \n", clientID, sub.TopicFilter)
				testingLocalVariable = clientID
				return true
			}, subscription.IterationOptions{
				Type: subscription.TypeAll,
			})
			// publish a message to the broker
			pub.Publish(&gmqtt.Message{
				Topic:   "topic",
				Payload: []byte("abc"),
				QoS:     packets.Qos1,
			})

		}
	}()

	go func() {
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
		<-signalCh
		srv.Stop(context.Background())
	}()
	err = srv.Run()
	if err != nil {
		panic(err)
	}
}
