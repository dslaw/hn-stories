package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHNClientGetWhenSuccessOnFirstRequest(t *testing.T) {
	httpClient := new(mockHTTPClient)
	httpClient.On("Get", mock.Anything).Return(
		makeMockResponse(http.StatusOK, "[10, 9, 8]"),
		nil,
	)

	client := NewHNClient(httpClient, "http://localhost", "v0", 0*time.Second, 1)

	actual, err := client.get("http://localhost/v0")

	assert.Nil(t, err)
	assert.Equal(t, []byte("[10, 9, 8]"), actual)
	httpClient.AssertCalled(t, "Get", "http://localhost/v0")
}

func TestHNClientGetWhenIssueRetries(t *testing.T) {
	for _, testCase := range []struct {
		statusCode int
		payload    string
		err        error
	}{
		// Retries after network error.
		{statusCode: http.StatusOK, err: fmt.Errorf("Network Error")},
		// Retries after null body with status 200.
		{statusCode: http.StatusOK, payload: "null", err: nil},
		// Retries after being throttled.
		{statusCode: http.StatusTooManyRequests, err: nil},
		// Retries after 5xx error.
		{statusCode: http.StatusInternalServerError, err: nil},
	} {
		httpClient := new(mockHTTPClient)
		mock.InOrder(
			httpClient.On("Get", mock.Anything).Return(
				makeMockResponse(testCase.statusCode, testCase.payload),
				testCase.err,
			).Once(),
			httpClient.On("Get", mock.Anything).Return(
				makeMockResponse(http.StatusOK, "[10, 9, 8]"),
				nil,
			).Once(),
		)

		client := NewHNClient(httpClient, "http://localhost", "v0", 0*time.Second, 2)

		actual, err := client.get("http://localhost/v0")

		assert.Nil(t, err)
		assert.Equal(t, []byte("[10, 9, 8]"), actual)
		httpClient.AssertCalled(t, "Get", "http://localhost/v0")
	}
}

func TestHNClientGetWhenMaxRetriesReachedReturnsError(t *testing.T) {
	httpClient := new(mockHTTPClient)
	httpClient.On("Get", mock.Anything).Return(
		makeMockResponse(http.StatusInternalServerError, ""),
		nil,
	).Times(2)

	client := NewHNClient(httpClient, "http://localhost", "v0", 0*time.Second, 2)

	_, err := client.get("http://localhost/v0")
	assert.ErrorIs(t, err, ErrMaxRetriesReached)
}

func TestHNClientGetWhenUnretryableErrorReturnsError(t *testing.T) {
	httpClient := new(mockHTTPClient)
	httpClient.On("Get", mock.Anything).Return(
		makeMockResponse(http.StatusNotFound, ""),
		nil,
	)

	client := NewHNClient(httpClient, "http://localhost", "v0", 0*time.Second, 1)

	_, err := client.get("http://localhost/v0")
	assert.NotNil(t, err)
}

func TestHNClientFetchNewStories(t *testing.T) {
	httpClient := new(mockHTTPClient)
	httpClient.On("Get", mock.Anything).Return(
		makeMockResponse(http.StatusOK, "[10, 9, 8]"),
		nil,
	)

	client := NewHNClient(httpClient, "http://localhost", "v0", 0*time.Second, 1)
	actual, err := client.FetchNewStories()

	assert.Nil(t, err)
	assert.Equal(t, []int64{10, 9, 8}, actual)
	httpClient.AssertCalled(t, "Get", "http://localhost/v0/newstories.json")
}

func TestHNClientFetchItem(t *testing.T) {
	type obj struct {
		ID   int64   `json:"id"`
		Kids []int64 `json:"kids"`
	}

	httpClient := new(mockHTTPClient)
	httpClient.On("Get", mock.Anything).Return(
		makeMockResponse(http.StatusOK, `{"id":1,"kids":[2,3]}`),
		nil,
	)

	client := NewHNClient(httpClient, "http://localhost", "v0", 0*time.Second, 1)

	o := obj{}
	err := client.FetchItem(int64(1), &o)

	assert.Nil(t, err)
	assert.Equal(t, int64(1), o.ID)
	assert.Equal(t, []int64{2, 3}, o.Kids)
	httpClient.AssertCalled(t, "Get", "http://localhost/v0/item/1.json")
}
