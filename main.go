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
	"github.com/eclipse/paho.mqtt.golang"
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
func buildingSubscriptionJSON(client string, sub *gmqtt.Subscription) string{
	s_json := "{"
	s_json += string(34) + "topic" + string(34) + ":" + string(34)  + sub.TopicFilter + string(34)
	s_json += ","
	s_json += string(34) + "client_id" + string(34) + ":" + string(34) + client + string(34)
	s_json += "}"
	return s_json
}

func getMessage(client mqtt.Client, msg mqtt.Message){
	fmt.Printf("Message %s\n", msg.Payload())
}

//func buildClientJSON(client server.Client) string{
//	s_json := string(34) + "client_id" + string(34) + ":" + string(34) + client.ClientOptions().ClientID + string(34)
//	return s_json
//}


func main() {
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

				//fmt.Println("==================")
				//fmt.Println(buildingSubscriptionJSON(clientID , sub))
				//fmt.Println("==================")

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
