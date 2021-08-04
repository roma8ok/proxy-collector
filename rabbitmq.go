package main

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

type RabbitMQSession struct {
	name            string
	logger          Logger
	connection      *amqp.Connection
	channel         *amqp.Channel
	done            chan bool
	notifyConnClose chan *amqp.Error
	notifyChanClose chan *amqp.Error
	notifyConfirm   chan amqp.Confirmation
	isReady         bool
}

type RabbitMQStreamHandler func(in []byte) (ack bool)

type htmlPayload struct {
	HTML    string `json:"html"`
	FromURL string `json:"from_url"`
}

var (
	errNotConnected  = errors.New("not connected to a server")
	errAlreadyClosed = errors.New("already closed: not connected to the server")
	errShutdown      = errors.New("session is shutting down")
)

// NewRabbitMQSession creates a new rabbitMQSession state instance, and automatically attempts to connect to the server
func NewRabbitMQSession(loki Logger, name string, addr string) *RabbitMQSession {
	session := RabbitMQSession{
		logger: loki,
		name:   name,
		done:   make(chan bool),
	}
	go session.handleReconnect(addr)
	return &session
}

// handleReconnect will wait for a connection error on notifyConnClose, and then continuously attempt to reconnect
func (session *RabbitMQSession) handleReconnect(addr string) {
	for {
		session.isReady = false
		session.logger.debug(fmt.Sprintf("Attempting to connect to rabbitMQ to %s", session.name))

		conn, err := session.connect(addr)

		if err != nil {
			session.logger.debug(fmt.Sprintf("Failed to connect to rabbitMQ to %s. Retrying...", session.name))

			select {
			case <-session.done:
				return
			case <-time.After(rabbitMQReconnectDelay):
			}
			continue
		}

		if done := session.handleReInit(conn); done {
			break
		}
	}
}

// connect will create a new AMQP connection
func (session *RabbitMQSession) connect(addr string) (*amqp.Connection, error) {
	conn, err := amqp.Dial(addr)

	if err != nil {
		return nil, err
	}

	session.changeConnection(conn)
	session.logger.debug(fmt.Sprintf("Connected to rabbitMQ to %s successfully", session.name))
	return conn, nil
}

// handleReInit will wait for a channel error and then continuously attempt to re-initialize both channels
func (session *RabbitMQSession) handleReInit(conn *amqp.Connection) bool {
	for {
		session.isReady = false

		err := session.init(conn)

		if err != nil {
			session.logger.debug(fmt.Sprintf("Failed to initialize channel in rabbitMQ to %s. Retrying...", session.name))

			select {
			case <-session.done:
				return true
			case <-time.After(rabbitMQReInitDelay):
			}
			continue
		}

		select {
		case <-session.done:
			return true
		case <-session.notifyConnClose:
			session.logger.debug(fmt.Sprintf("Connection to rabbitMQ to %s closed. Reconnecting...", session.name))

			return false
		case <-session.notifyChanClose:
			session.logger.debug(fmt.Sprintf("Channel to rabbitMQ to %s closed. Re-running init...", session.name))
		}
	}
}

// init will initialize channel & declare queue
func (session *RabbitMQSession) init(conn *amqp.Connection) error {
	ch, err := conn.Channel()

	if err != nil {
		return err
	}

	err = ch.Confirm(false)
	if err != nil {
		return err
	}

	if err := ch.Qos(3, 0, false); err != nil {
		return err
	}

	_, err = ch.QueueDeclare(
		session.name,
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return err
	}

	session.changeChannel(ch)
	session.isReady = true
	session.logger.debug(fmt.Sprintf("Setup rabbitMQ session to %s", session.name))

	return nil
}

// changeConnection takes a new connection to the queue,
// and updates the close listener to reflect this.
func (session *RabbitMQSession) changeConnection(connection *amqp.Connection) {
	session.connection = connection
	session.notifyConnClose = make(chan *amqp.Error)
	session.connection.NotifyClose(session.notifyConnClose)
}

// changeChannel takes a new channel to the queue,
// and updates the channel listeners to reflect this.
func (session *RabbitMQSession) changeChannel(channel *amqp.Channel) {
	session.channel = channel
	session.notifyChanClose = make(chan *amqp.Error)
	session.notifyConfirm = make(chan amqp.Confirmation, 1)
	session.channel.NotifyClose(session.notifyChanClose)
	session.channel.NotifyPublish(session.notifyConfirm)
}

// Push will push data onto the queue, and wait for a confirm.
// If no confirms are received until within the resendTimeout,
// it continuously re-sends messages until a confirm is received.
// This will block until the server sends a confirm. Errors are
// only returned if the push action itself fails, see unsafePush.
func (session *RabbitMQSession) Push(data []byte) error {
	if !session.isReady {
		return errors.New("Failed RabbitMQSession.push: not connected")
	}
	for {
		err := session.unsafePush(data)
		if err != nil {
			session.logger.debug(fmt.Sprintf("Push to %s failed. Retrying...", session.name))
			select {
			case <-session.done:
				return errShutdown
			case <-time.After(rabbitMQResendDelay):
			}
			continue
		}
		select {
		case confirm := <-session.notifyConfirm:
			if confirm.Ack {
				return nil
			}
		case <-time.After(rabbitMQResendDelay):
		}
		session.logger.debug(fmt.Sprintf("Push to %s didn't confirm. Retrying...", session.name))
	}
}

// unsafePush will push to the queue without checking for
// confirmation. It returns an error if it fails to connect.
// No guarantees are provided for whether the server will
// receive the message.
func (session *RabbitMQSession) unsafePush(data []byte) error {
	if !session.isReady {
		return errNotConnected
	}
	return session.channel.Publish(
		"",
		session.name,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        data,
		},
	)
}

// stream will continuously put queue items on the channel.
// It is required to call delivery.Ack when it has been
// successfully processed, or delivery.Nack when it fails.
// Ignoring this will cause data to build up on the server.
func (session *RabbitMQSession) stream() (<-chan amqp.Delivery, error) {
	if !session.isReady {
		return nil, errNotConnected
	}

	return session.channel.Consume(
		session.name,
		"", // Consumer
		false,
		false,
		false,
		false,
		nil,
	)
}

func (session *RabbitMQSession) handleStream(handler RabbitMQStreamHandler) {
LOOP:
	if !session.isReady {
		time.Sleep(rabbitMQRecheckStreamDelay)
		goto LOOP
	}

	for {
		msgs, err := session.stream()
		if err != nil {
			session.logger.debug(fmt.Sprintf("Stream to %s failed. Retrying...", session.name))
			select {
			case <-session.done:
				return
			case <-time.After(rabbitMQRecheckStreamDelay):
			}
			continue
		}

		for {
			if !session.isReady {
				goto LOOP
			}
			select {
			case msg := <-msgs:
				if ack := handler(msg.Body); ack {
					if err := msg.Ack(false); err != nil {
					}
				} else {
					if err := msg.Nack(false, false); err != nil {
					}
				}
				continue
			case <-time.After(rabbitMQRecheckStreamDelay):
			}
		}
	}
}

// close will cleanly shutdown the channel and connection
func (session *RabbitMQSession) close() error {
	if !session.isReady {
		return errAlreadyClosed
	}
	err := session.channel.Close()
	if err != nil {
		return err
	}
	err = session.connection.Close()
	if err != nil {
		return err
	}
	close(session.done)
	session.isReady = false
	return nil
}
