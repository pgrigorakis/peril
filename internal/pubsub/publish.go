package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {

	valBytes, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("could not marshal value: %w", err)
	}

	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        valBytes,
	})
	if err != nil {
		return fmt.Errorf("could not publish message: %w\n", err)
	}

	return nil
}
