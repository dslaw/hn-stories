package main

import (
	"context"
	"time"
)

type Enqueuer interface {
	Enqueue(context.Context, Message) error
	ProcessAfter() time.Duration
}

// MessageProducer produces messages for later delayed processing.
type MessageProducer struct {
	dst Enqueuer
}

func NewMessageProducer(dst Enqueuer) *MessageProducer {
	return &MessageProducer{dst: dst}
}

func (p *MessageProducer) MakeMessage(storyID int64, createdAt time.Time) Message {
	processAt := createdAt.Add(p.dst.ProcessAfter())
	return Message{StoryID: storyID, CreatedAt: createdAt, ProcessAt: processAt}
}

func (p *MessageProducer) SendMessage(ctx context.Context, storyID int64, createdAt time.Time) error {
	msg := p.MakeMessage(storyID, createdAt)
	return p.dst.Enqueue(ctx, msg)
}

// NopProducer does not produce messages.
type NopProducer struct{}

func (p *NopProducer) SendMessage(_ context.Context, _ int64, _ time.Time) error {
	return nil
}
