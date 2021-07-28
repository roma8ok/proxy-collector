package main

import (
	"github.com/streadway/amqp"
)

type htmlPayload struct {
	HTML    string `json:"html"`
	FromURL string `json:"from_url"`
}

func initQueues(conn *amqp.Connection, queues []string) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	for _, q := range queues {
		if _, err := ch.QueueDeclare(
			q,
			false,
			false,
			false,
			false,
			nil,
		); err != nil {
			return err
		}
	}
	return nil
}

func publish(ch *amqp.Channel, queueName string, body []byte) error {
	if err := ch.Publish(
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
		}); err != nil {
		return err
	}

	return nil
}

func consume(ch *amqp.Channel, queueName string, handler func([]byte) error) error {
	msgs, err := ch.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for m := range msgs {
		if err := handler(m.Body); err != nil {
			if err := m.Nack(false, false); err != nil {
				return err
			}
		} else {
			if err := m.Ack(false); err != nil {
				return err
			}
		}
	}

	return nil
}
