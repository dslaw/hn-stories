package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	ResourceNameNewStories       = "newstories"
	ResourceNameItem             = "item"
	MaxBackoffJitterMilliseconds = 250
)

var ErrMaxRetriesReached = errors.New("Maximum retries reached")

// HNComment represents a marshalled story from the Hacker News API.
type HNComment struct {
	By     string  `json:"by"`
	ID     int64   `json:"id"`
	Kids   []int64 `json:"kids"`
	Parent int64   `json:"parent"`
	Text   string  `json:"text"`
	Time   int64   `json:"time"`
	Type   string  `json:"type"`
}

// HNStory represents a marshalled story from the Hacker News API.
type HNStory struct {
	By          string  `json:"by"`
	Descendants int32   `json:"descendants"`
	ID          int64   `json:"id"`
	Kids        []int64 `json:"kids"`
	Score       int32   `json:"score"`
	Time        int64   `json:"time"`
	Title       string  `json:"title"`
	Type        string  `json:"type"`
	URL         string  `json:"url"`
}

type HTTPGetter interface {
	Get(string) (*http.Response, error)
}

// HNClient is an HTTP client for the Hacker News API.
type HNClient struct {
	client      HTTPGetter
	BaseURL     string
	APIVersion  string
	Backoff     time.Duration
	MaxAttempts int
}

func NewHNClient(client HTTPGetter, baseURL, apiVersion string, backoff time.Duration, maxAttempts int) *HNClient {
	if maxAttempts <= 0 {
		panic("Max attempts must be positive")
	}

	return &HNClient{
		client:      client,
		BaseURL:     baseURL,
		APIVersion:  apiVersion,
		Backoff:     backoff,
		MaxAttempts: maxAttempts,
	}
}

func (c *HNClient) get(url string) ([]byte, error) {
	var (
		rsp     *http.Response
		err     error
		payload []byte
	)

	for attempt := 0; attempt < c.MaxAttempts; attempt++ {
		if attempt > 0 {
			jitter := time.Duration(rand.Int63n(MaxBackoffJitterMilliseconds)) * time.Millisecond
			backoff := time.Duration(attempt)*c.Backoff + jitter
			time.Sleep(backoff)
		}

		rsp, err = c.client.Get(url)
		if err != nil {
			continue
		}
		defer rsp.Body.Close()

		payload, err = io.ReadAll(rsp.Body)
		if err != nil {
			continue
		}

		switch {
		case rsp.StatusCode == http.StatusOK && string(payload) == "null":
			// XXX: The HN API will return 200 with a body of `null` for
			//      non-existent resources. This also occurs when the resource
			//      exists, but wasn't able to be retrieved for whatever reason.
			//      The latter case should be ephemeral, and can be resolved by
			//      retrying.
			continue
		case rsp.StatusCode == http.StatusOK:
			return payload, nil
		case rsp.StatusCode == http.StatusTooManyRequests:
			continue
		case rsp.StatusCode >= http.StatusInternalServerError:
			continue
		default:
			return payload, fmt.Errorf("HTTP Error: %d", rsp.StatusCode)
		}

	}

	if err != nil {
		err = fmt.Errorf("%w: %s", ErrMaxRetriesReached, err.Error())
	} else {
		err = ErrMaxRetriesReached
	}

	return payload, err
}

func (c *HNClient) FetchNewStories() ([]int64, error) {
	var newStoryIDs []int64

	url := strings.Join([]string{c.BaseURL, c.APIVersion, ResourceNameNewStories}, "/") + ".json"

	payload, err := c.get(url)
	if err != nil {
		return newStoryIDs, err
	}

	err = json.Unmarshal(payload, &newStoryIDs)
	return newStoryIDs, err
}

func (c *HNClient) FetchItem(id int64, o interface{}) error {
	idString := strconv.Itoa(int(id))
	url := strings.Join([]string{c.BaseURL, c.APIVersion, ResourceNameItem, idString}, "/") + ".json"

	payload, err := c.get(url)
	if err != nil {
		return err
	}

	return json.Unmarshal(payload, &o)
}
