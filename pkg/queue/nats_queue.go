package queue

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/adiyakaihsan/go-logger/pkg/types"
	"github.com/nats-io/nats.go"
)

type NatsQueue struct {
	subject   string
	conn      *nats.Conn
	queueName string
	msgChan   chan *nats.Msg
	sub       *nats.Subscription
	js        nats.JetStreamContext
}

func NewNatsQueue(url, subject, queueName string, jsEnabled bool) (*NatsQueue, error) {
	opts := []nats.Option{
		nats.Timeout(5 * time.Second),   // Connection timeout
		nats.ReconnectWait(time.Second), // Wait 1 second before reconnect
		nats.MaxReconnects(5),           // Maximum reconnection attempts
		// Add error handlers
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			log.Printf("NATS error: %v", err)
		}),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			log.Printf("NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			log.Printf("NATS reconnected")
		}),
	}

	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, err
	}
	log.Printf("Connected to NATS")
	var js nats.JetStreamContext
	var sub *nats.Subscription
	if jsEnabled {
		js, err = nc.JetStream()
		if err != nil {
			return nil, err
		}
		log.Printf("Initialized Jetstream")

		_, err = js.AddStream(&nats.StreamConfig{
			Name:     queueName,
			Subjects: []string{subject},
		})
		if err != nil && err != nats.ErrStreamNameAlreadyInUse {
			log.Fatal("Error adding stream:", err)
		}
		log.Printf("Created Stream")

	}

	nq := &NatsQueue{
		conn:      nc,
		subject:   subject,
		queueName: queueName,
		msgChan:   make(chan *nats.Msg),
		// stream:    stream,
		// consumer:  consumer,
		js: js,
		// sub: sub,
	}

	if jsEnabled {
		sub, err = nq.js.Subscribe(nq.subject, func(msg *nats.Msg) {
			nq.msgChan <- msg
			msg.Ack() // Manually acknowledge the message
		}, nats.Durable("my-durable-consumer"), nats.ManualAck(), nats.AckWait(30*time.Second))

		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Subscribed to subject")
	} else {
		sub, err = nq.conn.QueueSubscribe(nq.subject, nq.queueName, func(msg *nats.Msg) {
			nq.msgChan <- msg
		})
		if err != nil {
			nq.conn.Close()
			return nil, err
		}
	}
	nq.sub = sub

	return nq, err
}

func (nq *NatsQueue) Enqueue(lg types.LogFormat) error {
	jsonLog, err := json.Marshal(lg)
	if err != nil {
		log.Printf("Cannot marshal log. Error: %v", err)
		return err
	}
	// nq.conn.Publish(nq.subject, jsonLog)
	ack, err := nq.js.Publish(nq.subject, []byte(jsonLog))
	if err != nil {
		return err
	}
	log.Printf("Published msg with sequence number %d on stream %q", ack.Sequence, ack.Stream)
	return nil
}

func (nq *NatsQueue) Dequeue() (types.LogFormat, error) {
	msg, ok := <-nq.msgChan
	if !ok {
		log.Println("Channel is closed")
		return types.LogFormat{}, errors.New("channel is closed")
	}

	var logFormat types.LogFormat
	if err := json.Unmarshal(msg.Data, &logFormat); err != nil {
		return types.LogFormat{}, err
	}
	return logFormat, nil
}

func (nq *NatsQueue) Close() {
	if nq.sub != nil {
		nq.sub.Unsubscribe()
	}
	close(nq.msgChan)
	nq.conn.Close()
}
