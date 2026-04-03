package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
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

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var valBytes bytes.Buffer
	enc := gob.NewEncoder(&valBytes)
	err := enc.Encode(val)
	if err != nil {
		return fmt.Errorf("could not encode value: %w", err)
	}

	return ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "application/gob",
		Body:        valBytes.Bytes(),
	})
}

func PublishGameLog(publichCh *amqp.Channel, username, msg string) error {
	return PublishGob(
		publichCh,
		routing.ExchangePerilTopic,
		routing.GameLogSlug+"."+username,
		routing.GameLog{
			Username:    username,
			Message:     msg,
			CurrentTime: time.Now(),
		})
}
