package main

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

const MaxBackoffMillisecond = 500

type Consumer interface {
	Fetch(context.Context) (int64, *time.Time, error)
}

type Producer interface {
	SendMessage(context.Context, int64, *time.Time) error
}

func Run(ctx context.Context, consumer Consumer, producer Producer) {
	for {
		storyID, createdAt, err := consumer.Fetch(ctx)
		if err != nil {
			slog.Error("Error fetching", "error", err)

			if errors.Is(err, ErrMessageExpired) {
				continue
			}

			panic(err)
		}

		err = producer.SendMessage(ctx, storyID, createdAt)
		if err != nil {
			slog.Error("Error sending message", "error", err)
			panic(err)
		}
	}
}
