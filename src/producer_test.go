package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockEnqueuer struct {
	mock.Mock
}

func (m *mockEnqueuer) Enqueue(ctx context.Context, msg Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *mockEnqueuer) ProcessAfter() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

func TestMessageProducerMakeMessage(t *testing.T) {
	config := QueueConfig{ProcessAfter: time.Hour}
	dst := &PriorityQueue{config: config}
	producer := NewMessageProducer(dst)

	storyID := int64(1)
	createdAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	expected := Message{
		StoryID:   storyID,
		CreatedAt: createdAt,
		ProcessAt: time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC),
	}

	actual := producer.MakeMessage(storyID, createdAt)

	assert.Equal(t, expected, actual)
}

func TestMessageProducerSendMessage(t *testing.T) {
	dst := new(mockEnqueuer)
	dst.On("Enqueue", mock.Anything, mock.Anything).Return(nil)
	dst.On("ProcessAfter").Return(time.Hour)

	producer := NewMessageProducer(dst)

	storyID := int64(1)
	createdAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	expectedMsg := Message{
		StoryID:   storyID,
		CreatedAt: createdAt,
		ProcessAt: time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC),
	}

	ctx := context.Background()
	err := producer.SendMessage(ctx, storyID, createdAt)

	assert.Nil(t, err)
	dst.AssertCalled(t, "Enqueue", ctx, expectedMsg)
}

func TestMessageProducerSendMessageWhenErrorEnqueuingReturnsError(t *testing.T) {
	dst := new(mockEnqueuer)
	dst.On("Enqueue", mock.Anything, mock.Anything).Return(fmt.Errorf("Error"))
	dst.On("ProcessAfter").Return(time.Hour)

	producer := NewMessageProducer(dst)

	storyID := int64(1)
	createdAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	expectedMsg := Message{
		StoryID:   storyID,
		CreatedAt: createdAt,
		ProcessAt: time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC),
	}

	ctx := context.Background()
	err := producer.SendMessage(ctx, storyID, createdAt)

	assert.NotNil(t, err)
	dst.AssertCalled(t, "Enqueue", ctx, expectedMsg)
}
