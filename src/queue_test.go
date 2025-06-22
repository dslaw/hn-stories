package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockBroker struct {
	mock.Mock
}

func (m *mockBroker) BZPopMin(ctx context.Context, timeout time.Duration, keys ...string) *redis.ZWithKeyCmd {
	args := m.Called(ctx, timeout, keys)
	return args.Get(0).(*redis.ZWithKeyCmd)
}

func (m *mockBroker) ZAddNX(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	args := m.Called(ctx, key, members)
	return args.Get(0).(*redis.IntCmd)
}

func TestPriorityQueueQueueName(t *testing.T) {
	config := QueueConfig{Name: "pq"}
	pq := PriorityQueue{config: config}
	assert.Equal(t, "pq", pq.QueueName())
}

func TestPriorityQueueProcessAfter(t *testing.T) {
	config := QueueConfig{Name: "pq", ProcessAfter: time.Second}
	pq := PriorityQueue{config: config}
	assert.Equal(t, time.Second, pq.ProcessAfter())
}

func TestPriorityQueueGracePeriod(t *testing.T) {
	config := QueueConfig{Name: "pq", GracePeriod: time.Second}
	pq := PriorityQueue{config: config}
	assert.Equal(t, time.Second, pq.GracePeriod())
}

func TestPriorityQueueEnqueue(t *testing.T) {
	broker := new(mockBroker)
	broker.On("ZAddNX", mock.Anything, mock.Anything, mock.Anything).Return(
		redis.NewIntResult(0, nil),
	)

	config := QueueConfig{Name: "pq"}
	pq := NewPriorityQueue(broker, config, 0*time.Second)

	ctx := context.Background()
	createdAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	msg := Message{
		StoryID:   1,
		CreatedAt: &createdAt,
		ProcessAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	err := pq.Enqueue(ctx, msg)

	expectedItem := redis.Z{
		Member: `{"story_id":1,"created_at":"2020-01-01T00:00:00Z"}`,
		Score:  float64(1577836800),
	}

	assert.Nil(t, err)
	broker.AssertCalled(t, "ZAddNX", ctx, "ingestion-queue:pq", []redis.Z{expectedItem})
}

func TestPriorityQueueEnqueueWhenErrorReturnsError(t *testing.T) {
	broker := new(mockBroker)
	broker.On("ZAddNX", mock.Anything, mock.Anything, mock.Anything).Return(
		redis.NewIntResult(0, fmt.Errorf("Error")),
	)

	config := QueueConfig{Name: "pq"}
	pq := NewPriorityQueue(broker, config, 0*time.Second)

	ctx := context.Background()
	createdAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	msg := Message{
		StoryID:   1,
		CreatedAt: &createdAt,
		ProcessAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	err := pq.Enqueue(ctx, msg)

	expectedItem := redis.Z{
		Member: `{"story_id":1,"created_at":"2020-01-01T00:00:00Z"}`,
		Score:  float64(1577836800),
	}

	assert.NotNil(t, err)
	broker.AssertCalled(t, "ZAddNX", ctx, "ingestion-queue:pq", []redis.Z{expectedItem})
}

func TestPriorityQueueDequeue(t *testing.T) {
	item := redis.ZWithKey{
		Z: redis.Z{
			Member: `{"story_id":1,"created_at":"2020-01-01T00:00:00Z"}`,
			Score:  float64(1577836800),
		},
	}

	createdAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	expected := Message{
		StoryID:   1,
		CreatedAt: &createdAt,
		ProcessAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	broker := new(mockBroker)
	broker.On("BZPopMin", mock.Anything, mock.Anything, mock.Anything).Return(
		redis.NewZWithKeyCmdResult(&item, nil),
	)

	config := QueueConfig{Name: "pq"}
	timeout := 0 * time.Second
	pq := NewPriorityQueue(broker, config, timeout)

	ctx := context.Background()
	actual, err := pq.Dequeue(ctx)

	assert.Nil(t, err)
	assert.Equal(t, expected, actual)
	broker.AssertCalled(t, "BZPopMin", ctx, timeout, []string{"ingestion-queue:pq"})
}

func TestPriorityQueueDequeueWhenErrorReturnsError(t *testing.T) {
	broker := new(mockBroker)
	broker.On("BZPopMin", mock.Anything, mock.Anything, mock.Anything).Return(
		redis.NewZWithKeyCmdResult(nil, fmt.Errorf("Error")),
	)

	config := QueueConfig{Name: "pq"}
	timeout := 0 * time.Second
	pq := NewPriorityQueue(broker, config, timeout)

	ctx := context.Background()
	_, err := pq.Dequeue(ctx)

	assert.NotNil(t, err)
	broker.AssertCalled(t, "BZPopMin", ctx, timeout, []string{"ingestion-queue:pq"})
}

func TestPriorityQueueDequeueWhenTimeoutReturnsErrtimeout(t *testing.T) {
	broker := new(mockBroker)
	broker.On("BZPopMin", mock.Anything, mock.Anything, mock.Anything).Return(
		redis.NewZWithKeyCmdResult(nil, nil),
	)

	config := QueueConfig{Name: "pq"}
	timeout := 0 * time.Second
	pq := NewPriorityQueue(broker, config, timeout)

	ctx := context.Background()
	_, err := pq.Dequeue(ctx)

	assert.ErrorIs(t, ErrTimeout, err)
	broker.AssertCalled(t, "BZPopMin", ctx, timeout, []string{"ingestion-queue:pq"})
}
