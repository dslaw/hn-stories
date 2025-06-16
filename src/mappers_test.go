package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMakeStoryModel(t *testing.T) {
	comments := []HNComment{
		{
			By:     "author1",
			ID:     2921983,
			Kids:   []int64{2922097, 2922429},
			Parent: 2921506,
			Text:   "Aw shucks, guys",
			Time:   1314211127,
			Type:   "comment",
		},
	}
	story := HNStory{
		By:          "author2",
		Descendants: 71,
		ID:          8863,
		Kids:        []int64{8952, 9224, 8917},
		Score:       111,
		Time:        1175714200,
		Title:       "My YC app: Dropbox",
		Type:        "story",
		URL:         "http://www.getdropbox.com/u/2/screencast.html",
	}

	apiVersion := "v0"
	queueName := "pq"
	fetchedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	expected := StoryModel{
		StoryID:     8863,
		APIVersion:  apiVersion,
		QueueName:   queueName,
		FetchedAt:   fetchedAt,
		RawDocument: `{"by":"author2","descendants":71,"id":8863,"kids":[8952,9224,8917],"score":111,"time":1175714200,"title":"My YC app: Dropbox","type":"story","url":"http://www.getdropbox.com/u/2/screencast.html"}`,
		Comments: []CommentModel{
			{
				CommentID:   2921983,
				RawDocument: `{"by":"author1","id":2921983,"kids":[2922097,2922429],"parent":2921506,"text":"Aw shucks, guys","time":1314211127,"type":"comment"}`,
			},
		},
	}

	actual, err := MakeStoryModel(story, comments, apiVersion, queueName, fetchedAt)

	assert.Nil(t, err)
	assert.Equal(t, expected, actual)
}
