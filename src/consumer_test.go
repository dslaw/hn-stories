package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) WriteStory(ctx context.Context, model StoryModel) error {
	args := m.Called(ctx, model)
	return args.Error(0)
}

func TestWaitUntil(t *testing.T) {
	now := time.Date(2020, 1, 1, 13, 0, 0, 0, time.UTC)

	type TestCase struct {
		WindowStart time.Time
		WindowEnd   time.Time
		Expected    bool
	}
	for _, testCase := range []TestCase{
		// Before window.
		{WindowStart: time.Date(2020, 1, 1, 14, 0, 0, 0, time.UTC), WindowEnd: time.Date(2020, 1, 1, 15, 0, 0, 0, time.UTC), Expected: false},
		// Exactly beginning of window.
		{WindowStart: time.Date(2020, 1, 1, 13, 0, 0, 0, time.UTC), WindowEnd: time.Date(2020, 1, 1, 15, 0, 0, 0, time.UTC), Expected: false},
		// Within window.
		{WindowStart: time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC), WindowEnd: time.Date(2020, 1, 1, 15, 0, 0, 0, time.UTC), Expected: false},
		// Exactly end of window.
		{WindowStart: time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC), WindowEnd: time.Date(2020, 1, 1, 13, 0, 0, 0, time.UTC), Expected: false},
		// After window.
		{WindowStart: time.Date(2020, 1, 1, 11, 0, 0, 0, time.UTC), WindowEnd: time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC), Expected: true},
	} {
		actual := WaitUntil(now, testCase.WindowStart, testCase.WindowEnd)
		assert.Equal(t, testCase.Expected, actual)
	}
}

func TestMessageConsumerFetch(t *testing.T) {
	payload := `{
        "by" : "user",
        "descendants" : 71,
        "id" : 1,
        "kids" : [],
        "score" : 111,
        "time" : 1175714200,
        "title" : "My YC app: Dropbox - Throw away your USB drive",
        "type" : "story",
        "url" : "http://www.getdropbox.com/u/2/screencast.html"
}`
	httpClient := new(mockHTTPClient)
	httpClient.On("Get", mock.Anything).Return(
		makeMockResponse(http.StatusOK, payload),
		nil,
	)

	expectedStoryID := int64(1)
	expectedCreatedAt := time.Date(2007, 4, 4, 19, 16, 40, 0, time.UTC)

	client := NewHNClient(httpClient, "http://localhost", "v0", 0*time.Second, 1)

	item := redis.ZWithKey{
		Z: redis.Z{
			Member: `{"story_id":1,"created_at":"2020-01-01T00:00:00Z"}`,
			Score:  float64(time.Now().UTC().Unix()), // Process immediately.
		},
	}

	broker := new(mockBroker)
	broker.On("BZPopMin", mock.Anything, mock.Anything, mock.Anything).Return(
		redis.NewZWithKeyCmdResult(&item, nil),
	)

	repo := new(mockRepo)
	repo.On("WriteStory", mock.Anything, mock.Anything).Return(nil)

	queueConfig := QueueConfig{Name: "pq", GracePeriod: time.Hour}
	src := NewPriorityQueue(broker, queueConfig, time.Nanosecond)

	consumer := NewMessageConsumer(client, src, repo)
	actualStoryID, actualCreatedAt, err := consumer.Fetch(context.Background())

	assert.Nil(t, err)
	assert.Equal(t, expectedStoryID, actualStoryID)
	assert.Equal(t, expectedCreatedAt, actualCreatedAt)

	httpClient.AssertCalled(t, "Get", "http://localhost/v0/item/1.json")
	repo.AssertNumberOfCalls(t, "WriteStory", 1)
}

func TestMessageConsumerFetchWhenDequeueErrorReturnsError(t *testing.T) {
	httpClient := new(mockHTTPClient)
	httpClient.On("Get", mock.Anything).Return(nil, nil) // Shouldn't be called.

	client := NewHNClient(httpClient, "http://localhost", "v0", 0*time.Second, 1)

	broker := new(mockBroker)
	broker.On("BZPopMin", mock.Anything, mock.Anything, mock.Anything).Return(
		redis.NewZWithKeyCmdResult(&redis.ZWithKey{}, fmt.Errorf("Error")),
	)

	repo := new(mockRepo)
	repo.On("Save", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	queueConfig := QueueConfig{Name: "pq", GracePeriod: time.Hour}
	src := NewPriorityQueue(broker, queueConfig, time.Nanosecond)

	consumer := NewMessageConsumer(client, src, repo)
	_, _, err := consumer.Fetch(context.Background())

	assert.NotNil(t, err)

	httpClient.AssertNotCalled(t, "Get")
	repo.AssertNotCalled(t, "Save")
}

func TestMessageConsumerFetchWhenProcessingWindowPassedEarlyReturn(t *testing.T) {
	httpClient := new(mockHTTPClient)
	httpClient.On("Get", mock.Anything).Return(nil, nil) // Shouldn't be called.

	client := NewHNClient(httpClient, "http://localhost", "v0", 0*time.Second, 1)

	expectedStoryID := int64(1)

	item := redis.ZWithKey{
		Z: redis.Z{
			Member: `{"story_id":1,"created_at":"2020-01-01T00:00:00Z"}`,
			Score:  float64(time.Now().UTC().Add(-12 * time.Hour).Unix()),
		},
	}

	broker := new(mockBroker)
	broker.On("BZPopMin", mock.Anything, mock.Anything, mock.Anything).Return(
		redis.NewZWithKeyCmdResult(&item, nil),
	)

	repo := new(mockRepo)
	repo.On("Save", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	queueConfig := QueueConfig{Name: "pq", GracePeriod: time.Hour}
	src := NewPriorityQueue(broker, queueConfig, 0*time.Second)

	consumer := NewMessageConsumer(client, src, repo)
	actualStoryID, _, err := consumer.Fetch(context.Background())

	assert.ErrorIs(t, err, ErrMessageExpired)
	assert.Equal(t, expectedStoryID, actualStoryID)

	httpClient.AssertNotCalled(t, "Get")
	repo.AssertNotCalled(t, "Save")
}

func TestHasDeadline(t *testing.T) {
	for _, testCase := range []struct {
		timeout  time.Duration
		expected bool
	}{
		{timeout: 0, expected: false},
		{timeout: time.Second, expected: true},
	} {
		actual := HasDeadline(testCase.timeout)
		assert.Equal(t, testCase.expected, actual)
	}
}

func TestFilterNewStories(t *testing.T) {
	for _, testCase := range []struct {
		newStoryIDs    []int64
		maxSeenStoryID int64
		expected       []int64
	}{
		{newStoryIDs: []int64{10, 9, 8}, maxSeenStoryID: 6, expected: []int64{10, 9, 8}},
		{newStoryIDs: []int64{10, 9, 8}, maxSeenStoryID: 7, expected: []int64{10, 9, 8}},
		{newStoryIDs: []int64{10, 9, 8}, maxSeenStoryID: 8, expected: []int64{10, 9}},
		{newStoryIDs: []int64{10, 9, 8}, maxSeenStoryID: 9, expected: []int64{10}},
		{newStoryIDs: []int64{10, 9, 8}, maxSeenStoryID: 11, expected: []int64{}},
	} {
		actual := FilterNewStories(testCase.newStoryIDs, testCase.maxSeenStoryID)
		assert.Equal(t, testCase.expected, actual)
	}
}

func TestLatestStoryConsumerPollForNewStories(t *testing.T) {
	payload := "[10, 9, 8]"

	for _, testCase := range []struct {
		buffer   []int64
		expected []int64
	}{
		// First call post-initialization.
		{buffer: []int64{}, expected: []int64{10, 9, 8}},
		// Using the last remaining buffered story id as a filter to avoid
		// duplicates.
		{buffer: []int64{9}, expected: []int64{10}},
	} {
		httpClient := new(mockHTTPClient)
		httpClient.On("Get", mock.Anything).Return(
			makeMockResponse(http.StatusOK, payload),
			nil,
		)
		client := NewHNClient(httpClient, "http://localhost", "v0", 0*time.Second, 1)
		consumer := NewLatestStoryConsumer(client, time.Second, time.Minute)
		consumer.buffer = testCase.buffer

		actual, err := consumer.PollForNewStories()

		assert.Nil(t, err)
		assert.Equal(t, testCase.expected, actual)
	}
}

func TestLatestStoryConsumerPollForNewStoriesWhenNoNewStoriesRepolls(t *testing.T) {
	httpClient := new(mockHTTPClient)
	mock.InOrder(
		httpClient.On("Get", mock.Anything).Return(
			makeMockResponse(http.StatusOK, "[9, 8, 7]"),
			nil,
		).Once(),
		httpClient.On("Get", mock.Anything).Return(
			makeMockResponse(http.StatusOK, "[10, 9, 8]"),
			nil,
		).Once(),
	)

	client := NewHNClient(httpClient, "http://localhost", "v0", 0*time.Second, 1)
	consumer := NewLatestStoryConsumer(client, 0*time.Second, 2*time.Second)
	consumer.buffer = []int64{9}

	actual, err := consumer.PollForNewStories()

	assert.Nil(t, err)
	assert.Equal(t, []int64{10}, actual)
}

func TestLatestStoryConsumerPollForNewStoriesWhenNoNewStoriesPollUntilTimeout(t *testing.T) {
	httpClient := new(mockHTTPClient)
	httpClient.On("Get", mock.Anything).Return(
		makeMockResponse(http.StatusOK, "[]"),
		nil,
	)

	client := NewHNClient(httpClient, "http://localhost", "v0", 0*time.Second, 1)
	consumer := NewLatestStoryConsumer(client, 0*time.Second, time.Nanosecond)

	_, err := consumer.PollForNewStories()

	assert.ErrorIs(t, err, ErrTimeoutExceeded)
	httpClient.AssertNumberOfCalls(t, "Get", 1)
}

func TestLatestStoryConsumerPollForNewStoriesWhenErrorFetching(t *testing.T) {
	httpClient := new(mockHTTPClient)
	httpClient.On("Get", mock.Anything).Return(&http.Response{}, fmt.Errorf("500 Internal Server Error"))

	client := NewHNClient(httpClient, "http://localhost", "v0", 0*time.Second, 1)
	consumer := NewLatestStoryConsumer(client, 0*time.Second, time.Nanosecond)

	_, err := consumer.PollForNewStories()

	// Client error is propagated.
	assert.ErrorIs(t, err, ErrMaxRetriesReached)
	httpClient.AssertNumberOfCalls(t, "Get", 1)
}

func TestLatestStoryConsumerFetch(t *testing.T) {
	payload := "[10, 9, 8]"

	for _, testCase := range []struct {
		buffer           []int64
		expectedBuffer   []int64
		expectedStoryID  int64
		expectedGetCalls int
	}{
		// First call post-initialization.
		{buffer: []int64{}, expectedBuffer: []int64{10, 9}, expectedStoryID: int64(8), expectedGetCalls: 1},
		// Using the last remaining buffered story id as a filter to avoid
		// duplicates.
		{buffer: []int64{9}, expectedBuffer: []int64{10}, expectedStoryID: int64(9), expectedGetCalls: 1},
		// Use buffered story ids without fetching.
		{buffer: []int64{10, 9, 8}, expectedBuffer: []int64{10, 9}, expectedStoryID: int64(8), expectedGetCalls: 0},
	} {
		httpClient := new(mockHTTPClient)
		httpClient.On("Get", mock.Anything).Return(
			makeMockResponse(http.StatusOK, payload),
			nil,
		)

		client := NewHNClient(httpClient, "http://localhost", "v0", 0*time.Second, 1)
		consumer := NewLatestStoryConsumer(client, 0*time.Second, time.Nanosecond)
		consumer.buffer = testCase.buffer

		actualStoryID, _, err := consumer.Fetch(context.Background())

		assert.Nil(t, err)
		// Story with smallest id is returned.
		assert.Equal(t, testCase.expectedStoryID, actualStoryID)
		// Buffer is updated. The returned story is not buffered, and the buffer
		// is updated with fetched new stories, if new stories were fetched.
		assert.Equal(t, testCase.expectedBuffer, consumer.buffer)
		// New stories are only fetched if there aren't enough in the buffer.
		httpClient.AssertNumberOfCalls(t, "Get", testCase.expectedGetCalls)
	}
}
