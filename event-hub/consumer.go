package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/IBM/sarama"
)

func consume() {

	brokerList := []string{"event hub:9093"}
	fmt.Println("Event Hubs broker", brokerList)
	consumerGroupID := "$Default"
	fmt.Println("Sarama client consumer group ID", consumerGroupID)

	config := getConfig()
	//config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Offsets.AutoCommit.Enable = false

	consumer, err := sarama.NewConsumerGroup(brokerList, consumerGroupID, config)

	if err != nil {
		fmt.Println("error creating new consumer group", err)
		os.Exit(1)
	}

	fmt.Println("new consumer group created")

	eventHubsTopic := "test-hub"
	fmt.Println("Event Hubs topic", eventHubsTopic)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			err = consumer.Consume(ctx, []string{"topico"}, messageHandler{})
			if err != nil {
				fmt.Println("error consuming from group", err)
				os.Exit(1)
			}

			if ctx.Err() != nil {
				//exit for loop
				return
			}
		}
	}()

	close := make(chan os.Signal)
	signal.Notify(close, syscall.SIGTERM, syscall.SIGINT)
	fmt.Println("Waiting for program to exit")
	<-close
	cancel()
	fmt.Println("closing consumer group....")

	if err := consumer.Close(); err != nil {
		fmt.Println("error trying to close consumer", err)
		os.Exit(1)
	}
	fmt.Println("consumer group closed")
}

type messageHandler struct{}

func (h messageHandler) Setup(s sarama.ConsumerGroupSession) error {
	fmt.Println("Partition allocation -", s.Claims())
	return nil
}

func (h messageHandler) Cleanup(s sarama.ConsumerGroupSession) error {
	fmt.Println("Consumer group clean up initiated")
	return nil
}
func (h messageHandler) ConsumeClaim(s sarama.ConsumerGroupSession, c sarama.ConsumerGroupClaim) error {
	for msg := range c.Messages() {
		fmt.Printf("Message topic:%q partition:%d offset:%d\n", msg.Topic, msg.Partition, msg.Offset)
		fmt.Println("Message key", string(msg.Key))
		for _, header := range msg.Headers {
			fmt.Println("Message header", string(header.Key), string(header.Value))
		}
		fmt.Println("Message content", string(msg.Value))
		s.MarkMessage(msg, "")
	}
	return nil
}
