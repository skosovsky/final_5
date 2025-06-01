package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type User struct {
	Name           string `json:"name"`
	FavoriteNumber int64  `json:"favorite_number"`
	FavoriteColor  string `json:"favorite_color"`
}

func main() {
	bootstrapServers := "kafka-0:9093"
	topic := "test"

	sslConfig := &kafka.ConfigMap{
		"bootstrap.servers":        bootstrapServers,
		"security.protocol":        "SSL",
		"ssl.ca.location":          "ca.crt",
		"ssl.certificate.location": "kafka-0-creds/kafka-0.crt",
		"ssl.key.location":         "kafka-0-creds/kafka-0.key",
		"ssl.key.password":         "kafka-password",
	}

	producer, err := kafka.NewProducer(sslConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close()

	fmt.Printf("Created Producer %v\n", producer)

	value := User{
		Name:           "John Doe",
		FavoriteNumber: 1,
		FavoriteColor:  "blue",
	}
	payload, err := json.Marshal(value)
	if err != nil {
		log.Fatal(err)
	}

	deliveryChan := make(chan kafka.Event)

	err = producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          payload,
		Headers:        []kafka.Header{{Key: "myTestHeader", Value: []byte("header values are binary")}},
	}, deliveryChan)
	if err != nil {
		log.Fatal(err)
	}

	event := <-deliveryChan
	message := event.(*kafka.Message)

	if message.TopicPartition.Error != nil {
		log.Printf("Delivery failed: %v\n", message.TopicPartition.Error)
	} else {
		log.Printf("Delivered message to topic %s [%d] at offset %v\n",
			*message.TopicPartition.Topic, message.TopicPartition.Partition, message.TopicPartition.Offset)
	}

	close(deliveryChan)

	runConsumer()
}

func runConsumer() {
	bootstrapServers := "kafka-0:9093"
	topic := "test"
	groupID := "my-consumer-group"

	sslConfig := &kafka.ConfigMap{
		"bootstrap.servers":        bootstrapServers,
		"security.protocol":        "SSL",
		"ssl.ca.location":          "ca.crt",
		"ssl.certificate.location": "kafka-0-creds/kafka-0.crt",
		"ssl.key.location":         "kafka-0-creds/kafka-0.key",
		"ssl.key.password":         "kafka-password",
		"group.id":                 groupID,
		"auto.offset.reset":        "earliest",
	}

	consumer, err := kafka.NewConsumer(sslConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer consumer.Close()

	err = consumer.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Subscribed to %v\n", topic)

	for {
		var msg *kafka.Message
		msg, err = consumer.ReadMessage(-1)
		if err != nil {
			var kafkaErr kafka.Error
			if errors.As(err, &kafkaErr) && kafkaErr.IsFatal() {
				log.Printf("Fatal error: %v\n", kafkaErr)
				break
			}

			continue
		}

		var user User
		err = json.Unmarshal(msg.Value, &user)
		if err != nil {
			log.Printf("Error unmarshalling message: %v\n", err)
		}

		fmt.Printf("Message on User: %v\n", user)
	}
}
