package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrTimeoutExceeded = errors.New("Timeout exceeded")
	ErrMessageExpired  = errors.New("Message expired")
	ErrFetching        = errors.New("Unable to fetch")
)

// Repoer provides a method to store Hacker News stories and comments.
type Repoer interface {
	WriteStory(context.Context, StoryModel) error
}

// WindowWaiter provides a method to wait until a processing window has begun.
type WindowWaiter interface {
	WaitUntil(time.Time, time.Time) bool
}

// WaitUntil waits until the given window has begun, or returns immediately
// if the window's start has already passed. If the window has elapsed at the
// time of calling, true is returned.
func WaitUntil(now time.Time, windowStart time.Time, windowEnd time.Time) bool {
	if now.After(windowEnd) {
		return true
	} else if now.Before(windowStart) {
		time.Sleep(time.Until(windowStart))
	}
	return false
}

type MessageConsumer struct {
	client *HNClient
	src    *PriorityQueue
	repo   Repoer
}

func NewMessageConsumer(client *HNClient, src *PriorityQueue, repo Repoer) *MessageConsumer {
	return &MessageConsumer{client: client, src: src, repo: repo}
}

func (c *MessageConsumer) Fetch(ctx context.Context) (storyID int64, createdAt *time.Time, err error) {
	msg, err := c.src.Dequeue(ctx)
	if err != nil {
		return
	}

	storyID = msg.StoryID

	processingWindowStart := msg.ProcessAt
	processingWindowEnd := msg.ProcessAt.Add(c.src.GracePeriod())

	processingWindowPassed := WaitUntil(time.Now().UTC(), processingWindowStart, processingWindowEnd)
	if processingWindowPassed {
		err = fmt.Errorf("%w: expired at %s", ErrMessageExpired, processingWindowEnd)
		return
	}

	story := HNStory{}
	err = c.client.FetchItem(msg.StoryID, &story)
	if err != nil {
		err = fmt.Errorf("%w story: %w", ErrFetching, err)
		return
	}

	storyCreatedAt := time.Unix(story.Time, 0).UTC()
	createdAt = &storyCreatedAt

	comments := make([]HNComment, len(story.Kids))
	for _, commentID := range story.Kids {
		comment := HNComment{}
		err = c.client.FetchItem(commentID, &comment)
		if err != nil {
			err = fmt.Errorf("%w comment: %w", ErrFetching, err)
			return
		}

		comments = append(comments, comment)
	}

	model, err := MakeStoryModel(
		story,
		comments,
		c.client.APIVersion,
		c.src.QueueName(),
		time.Now().UTC(),
	)
	if err != nil {
		return
	}

	err = c.repo.WriteStory(ctx, model)
	return
}

func HasDeadline(timeout time.Duration) bool {
	return timeout > 0
}

func FilterNewStories(newStoryIDs []int64, maxSeenStoryID int64) []int64 {
	for idx, id := range newStoryIDs {
		if id <= maxSeenStoryID {
			return newStoryIDs[:idx]
		}
	}

	return newStoryIDs
}

// LatestStoryConsumer consumes new story ids from the Hacker News API and
// provides a method to retrieve them, in order.
//
// A Timeout value of 0 indicates that the consumer should not timeout.
type LatestStoryConsumer struct {
	client       *HNClient
	buffer       []int64
	PollInterval time.Duration
	Timeout      time.Duration
}

func NewLatestStoryConsumer(client *HNClient, pollInterval, timeout time.Duration) *LatestStoryConsumer {
	return &LatestStoryConsumer{client: client, PollInterval: pollInterval, Timeout: timeout}
}

// PollForNewStories fetches new story ids from the Hacker News API, polling
// until new stories are found or the configured timeout is reached, as
// necessary.
func (c *LatestStoryConsumer) PollForNewStories() (ids []int64, err error) {
	deadline := time.Now().UTC().Add(c.Timeout)
	hasDeadline := HasDeadline(c.Timeout)

	for {
		ids, err = c.client.FetchNewStories()
		if err != nil {
			break
		}

		// Filter out new stories that have already been enqueued (ostensibly).
		//
		// New story ids are expected to be returned in sorted, descending
		// order. That is, the newest story is the first element, then the next
		// newest, etc. `c.buffer` follows this ordering, as well.
		if len(c.buffer) > 0 {
			ids = FilterNewStories(ids, c.buffer[0])
		}

		if len(ids) > 0 {
			break
		}

		time.Sleep(c.PollInterval)

		if hasDeadline && time.Now().UTC().After(deadline) {
			err = ErrTimeoutExceeded
			break
		}
	}

	return
}

// Fetch returns the id of the next new story, fetching new story ids from the
// Hacker News API as necessary. If no new story ids are available, it will
// block until new story ids become available or the configured deadline is
// reached.
func (c *LatestStoryConsumer) Fetch(_ context.Context) (storyID int64, _ *time.Time, err error) {
	// Fill up the buffer of new story ids, using the last remaining buffered
	// story id to filter out the API's returned new stories, if available.
	if len(c.buffer) <= 1 {
		var ids []int64
		ids, err = c.PollForNewStories()
		if err != nil {
			return
		}

		c.buffer = append(ids, c.buffer...)
	}

	n := len(c.buffer)
	c.buffer, storyID = c.buffer[:n-1], c.buffer[n-1]
	return
}
