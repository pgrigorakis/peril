package pubsub

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	return subscribe(
		conn,
		exchange,
		queueName,
		key,
		queueType,
		handler,
		jsonUnmarshaller[T])
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	return subscribe(
		conn,
		exchange,
		queueName,
		key,
		queueType,
		handler,
		gobUnmarshaller[T])
}

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
) error {
	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return fmt.Errorf("could not bind queue to exchange: %w", err)
	}

	deliveryCh, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("could not create channel: %w", err)
	}

	go func() {
		for msg := range deliveryCh {
			decodedMsg, err := unmarshaller(msg.Body)
			if err != nil {
				log.Printf("could not decode message: %v", err)
			}
			ackType := handler(decodedMsg)
			switch ackType {
			case Ack:
				msg.Ack(false)
			case NackRequeue:
				msg.Nack(false, true)
			case NackDiscard:
				msg.Nack(false, false)
			}
		}
	}()
	return nil
}

func jsonUnmarshaller[T any](msg []byte) (T, error) {
	var dat T
	err := json.Unmarshal(msg, &dat)
	if err != nil {
		return dat, fmt.Errorf("error: %v", err)
	}
	return dat, err
}

func gobUnmarshaller[T any](msg []byte) (T, error) {
	dat := bytes.NewBuffer(msg)
	dec := gob.NewDecoder(dat)

	var decodedMessage T

	err := dec.Decode(&decodedMessage)
	if err != nil {
		return decodedMessage, fmt.Errorf("could not decode value: %v", err)
	}
	return decodedMessage, err
}
