package amqp

import (
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

type Publisher struct {
	pub message.Publisher
	sub message.Subscriber
}

func NewPublisherSubscriber(pub message.Publisher, sub message.Subscriber) Publisher {
	return Publisher{pub: pub, sub: sub}
}

func (p Publisher) Publish(topic string, payload interface{}) error {
	marshaledPayload, _ := json.Marshal(payload)
	msg := message.NewMessage(watermill.NewUUID(), marshaledPayload)
	middleware.SetCorrelationID(watermill.NewUUID(), msg)
	if err := p.pub.Publish(topic, msg); err != nil {
		return err
	}
	return nil
}
