package main

import (
	"encoding/json"
	"time"
)

func MakeStoryModel(
	story HNStory,
	comments []HNComment,
	apiVersion string,
	queueName string,
	fetchedAt time.Time,
) (StoryModel, error) {
	model := StoryModel{
		StoryID:    story.ID,
		APIVersion: apiVersion,
		QueueName:  queueName,
		FetchedAt:  fetchedAt,
	}

	raw, err := json.Marshal(story)
	if err != nil {
		return model, err
	}

	model.RawDocument = string(raw)

	for _, comment := range comments {
		raw, err := json.Marshal(comment)
		if err != nil {
			return model, err
		}

		model.Comments = append(model.Comments, CommentModel{
			CommentID:   comment.ID,
			RawDocument: string(raw),
		})
	}

	return model, nil
}
