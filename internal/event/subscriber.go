package event

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"github.com/go-tangra/go-tangra-hr/internal/conf"

	appViewer "github.com/go-tangra/go-tangra-common/viewer"
)

// Subscriber handles Redis pub/sub event subscriptions for HR
type Subscriber struct {
	log     *log.Helper
	rdb     *redis.Client
	handler *Handler
	config  *conf.EventConfig
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
	mu      sync.Mutex
}

// NewSubscriber creates a new event subscriber
func NewSubscriber(ctx *bootstrap.Context, rdb *redis.Client, handler *Handler) *Subscriber {
	var eventCfg *conf.EventConfig
	if cfg, ok := ctx.GetCustomConfig("hr"); ok && cfg != nil {
		if hrCfg, ok := cfg.(*conf.HR); ok && hrCfg.Events != nil {
			eventCfg = hrCfg.Events
		}
	}

	// Default config if not set
	if eventCfg == nil {
		eventCfg = &conf.EventConfig{
			Enabled:     true,
			TopicPrefix: "paperless",
			SubscribeEvents: []string{
				"signing.request.completed",
			},
		}
	}

	return &Subscriber{
		log:     ctx.NewLoggerHelper("hr/event/subscriber"),
		rdb:     rdb,
		handler: handler,
		config:  eventCfg,
	}
}

// Start starts the event subscriber
func (s *Subscriber) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	if !s.config.Enabled {
		s.log.Info("Event subscriber is disabled")
		return nil
	}

	if s.rdb == nil {
		s.log.Warn("Redis client not available, event subscriber disabled")
		return nil
	}

	baseCtx := appViewer.NewSystemViewerContext(context.Background())
	s.ctx, s.cancel = context.WithCancel(baseCtx)
	s.running = true

	prefix := s.config.TopicPrefix
	if prefix == "" {
		prefix = "paperless"
	}

	channels := make([]string, len(s.config.SubscribeEvents))
	for i, event := range s.config.SubscribeEvents {
		channels[i] = fmt.Sprintf("%s.%s", prefix, event)
	}

	s.log.Infof("Starting event subscriber for channels: %v", channels)

	pubsub := s.rdb.PSubscribe(s.ctx, channels...)

	s.wg.Add(1)
	go s.listen(pubsub)

	return nil
}

// Stop stops the event subscriber
func (s *Subscriber) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.log.Info("Stopping event subscriber")
	s.cancel()
	s.wg.Wait()
	s.running = false

	return nil
}

// listen listens for events on the pub/sub channels
func (s *Subscriber) listen(pubsub *redis.PubSub) {
	defer s.wg.Done()
	defer pubsub.Close()

	ch := pubsub.Channel()

	for {
		select {
		case <-s.ctx.Done():
			s.log.Info("Event subscriber stopped")
			return
		case msg, ok := <-ch:
			if !ok {
				s.log.Warn("Pub/sub channel closed")
				return
			}
			s.handleMessage(msg)
		}
	}
}

// handleMessage processes a pub/sub message
func (s *Subscriber) handleMessage(msg *redis.Message) {
	s.log.Infof("Received event on channel %s", msg.Channel)

	var signingEvent SigningEvent
	if err := json.Unmarshal([]byte(msg.Payload), &signingEvent); err != nil {
		s.log.Errorf("Failed to unmarshal signing event: %v", err)
		return
	}

	// Extract event type from channel name
	prefix := s.config.TopicPrefix
	if prefix == "" {
		prefix = "paperless"
	}
	var eventType string
	if len(msg.Channel) > len(prefix)+1 {
		eventType = msg.Channel[len(prefix)+1:]
	}

	switch eventType {
	case "signing.request.completed":
		var data SigningRequestCompletedData
		if err := json.Unmarshal(signingEvent.Data, &data); err != nil {
			s.log.Errorf("Failed to parse signing.request.completed data: %v", err)
			return
		}
		if err := s.handler.HandleSigningCompleted(s.ctx, &data); err != nil {
			s.log.Errorf("Failed to handle signing completed event: %v", err)
		}
	default:
		s.log.Infof("Ignoring unknown event type: %s", eventType)
	}
}
