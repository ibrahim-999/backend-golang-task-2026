package messaging

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
)

type rabbitEnvelope struct {
	Name        string `json:"name"`
	AggregateID uint64 `json:"aggregate_id"`
}

type RabbitPublisher struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
	log      zerolog.Logger
}

func NewRabbitPublisher(url, exchange string, log zerolog.Logger) (*RabbitPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if err := channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, err
	}

	return &RabbitPublisher{
		conn:     conn,
		channel:  channel,
		exchange: exchange,
		log:      log,
	}, nil
}

func (p *RabbitPublisher) Publish(ctx context.Context, events ...shared.Event) error {
	for _, event := range events {
		body, err := json.Marshal(rabbitEnvelope{
			Name:        event.EventName(),
			AggregateID: event.AggregateID(),
		})
		if err != nil {
			return err
		}

		msg := amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		}

		if err := p.channel.PublishWithContext(ctx, p.exchange, event.EventName(), false, false, msg); err != nil {
			return err
		}

		p.log.Debug().
			Str("exchange", p.exchange).
			Str("routing_key", event.EventName()).
			Uint64("aggregate_id", event.AggregateID()).
			Msg("event published to rabbitmq")
	}
	return nil
}

func (p *RabbitPublisher) Close() error {
	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			p.log.Warn().Err(err).Msg("rabbitmq channel close failed")
		}
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

var _ ports.EventPublisher = (*RabbitPublisher)(nil)

type lifecycleBus interface {
	ports.EventBus
	Start()
	Stop()
}

type CompositeBus struct {
	primary     lifecycleBus
	alsoPublish ports.EventPublisher
	log         zerolog.Logger
}

func NewCompositeBus(primary ports.EventBus, alsoPublish ports.EventPublisher) *CompositeBus {
	c := &CompositeBus{alsoPublish: alsoPublish}
	if lb, ok := primary.(lifecycleBus); ok {
		c.primary = lb
	} else {
		c.primary = noopLifecycle{primary}
	}
	return c
}

func (c *CompositeBus) Subscribe(handler ports.EventHandler) {
	c.primary.Subscribe(handler)
}

func (c *CompositeBus) Publish(ctx context.Context, events ...shared.Event) error {
	err := c.primary.Publish(ctx, events...)
	if pubErr := c.alsoPublish.Publish(ctx, events...); pubErr != nil {
		c.log.Warn().Err(pubErr).Int("events", len(events)).Msg("rabbitmq publish failed; continuing")
	}
	return err
}

func (c *CompositeBus) Start() {
	c.primary.Start()
}

func (c *CompositeBus) Stop() {
	c.primary.Stop()
}

type noopLifecycle struct {
	ports.EventBus
}

func (noopLifecycle) Start() {}

func (noopLifecycle) Stop() {}

var _ ports.EventBus = (*CompositeBus)(nil)

type RabbitConsumer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
	queue    string
	log      zerolog.Logger
	done     chan struct{}
}

func NewRabbitConsumer(url, exchange string, log zerolog.Logger) (*RabbitConsumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if err := channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, err
	}

	queue, err := channel.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, err
	}

	if err := channel.QueueBind(queue.Name, "#", exchange, false, nil); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, err
	}

	return &RabbitConsumer{
		conn:     conn,
		channel:  channel,
		exchange: exchange,
		queue:    queue.Name,
		log:      log,
		done:     make(chan struct{}),
	}, nil
}

func (c *RabbitConsumer) Start(ctx context.Context) error {
	deliveries, err := c.channel.Consume(c.queue, "", true, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-c.done:
				return
			case d, ok := <-deliveries:
				if !ok {
					return
				}
				var env rabbitEnvelope
				if err := json.Unmarshal(d.Body, &env); err != nil {
					c.log.Warn().Err(err).Str("routing_key", d.RoutingKey).Msg("rabbitmq consumer failed to decode event")
					continue
				}
				c.log.Info().
					Str("exchange", c.exchange).
					Str("routing_key", d.RoutingKey).
					Str("name", env.Name).
					Uint64("aggregate_id", env.AggregateID).
					Msg("rabbitmq event received")
			}
		}
	}()

	return nil
}

func (c *RabbitConsumer) Stop() {
	close(c.done)
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			c.log.Warn().Err(err).Msg("rabbitmq consumer channel close failed")
		}
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			c.log.Warn().Err(err).Msg("rabbitmq consumer connection close failed")
		}
	}
}
