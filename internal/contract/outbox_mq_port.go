package contract

import "context"

// OutboxMQMessage is the transport view of an outbox event. It deliberately
// contains no queue or outbox implementation types.
type OutboxMQMessage struct {
	ID          uint64
	EventType   string
	Payload     string
	Traceparent string
	Tracestate  string
}

// OutboxMQHandler consumes one message from the outbox bridge.
type OutboxMQHandler func(context.Context, OutboxMQMessage) error

// OutboxMQPublisher publishes outbox messages to the selected queue driver.
type OutboxMQPublisher interface {
	Publish(context.Context, ...OutboxMQMessage) error
}

// OutboxMQFactory lazily creates the queue client and its worker component.
// Start must be called only when outbox.publisher is mq.
type OutboxMQFactory interface {
	StartOutbox(context.Context, []string, OutboxMQHandler) (OutboxMQPublisher, error)
}

const OutboxMQPortName = "outbox_mq"
