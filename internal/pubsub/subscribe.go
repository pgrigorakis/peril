package pubsub

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T),
) error {
	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return fmt.Errorf("could not bind queue to exchange: %w", err)
	}

	deliveryCh, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("could not create channel: %w", err)
	}

	go func() {
		for msg := range deliveryCh {
			var dat T
			_ = json.Unmarshal(msg.Body, &dat)
			handler(dat)
			msg.Ack(false)
		}
	}()

	return nil
}
