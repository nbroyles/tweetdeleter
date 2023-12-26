package main

import (
	"flag"
	"log"
	"time"

	"go.uber.org/zap"

	"tweetdeleter/internal"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Could not create zap logger")
	}

	username := flag.String("username", "", "x/twitter account to log into and delete tweets")
	password := flag.String("password", "", "password for provided account")
	startDate := flag.String("start-date", "", "start date of time range to delete tweets. must be formatted as YYYY-MM-DD")
	endDate := flag.String("end-date", "", "end date (inclusive) of time range to delete tweets. must be formatted as YYYY-MM-DD")

	flag.Parse()

	if *username == "" {
		logger.Fatal("username flag is required")
	}
	if *password == "" {
		logger.Fatal("password flag is required")
	}
	if *startDate == "" {
		logger.Fatal("start-date flag is required")
	}
	if *endDate == "" {
		logger.Fatal("end-date flag is required")
	}

	parsedStart, err := time.Parse(time.DateOnly, *startDate)
	if err != nil {
		logger.Fatal("could not parse start date", zap.Error(err))
	}
	parsedEnd, err := time.Parse(time.DateOnly, *endDate)
	if err != nil {
		logger.Fatal("could not parse end date", zap.Error(err))
	}

	if !parsedEnd.After(parsedStart) {
		logger.Fatal("invalid start and end time. start time must be before end time",
			zap.Time("startDate", parsedStart), zap.Time("endDate", parsedEnd))
	}

	td, err := internal.NewTweetDeleter(internal.TweetDeleterOptions{
		Username:  *username,
		Password:  *password,
		StartDate: parsedStart,
		EndDate:   parsedEnd,
		Logger:    logger,
	})
	if err != nil {
		logger.Fatal("could not create TweetDeleter", zap.Error(err))
	}
	if err = td.Run(); err != nil {
		logger.Error("error running TweetDeleter", zap.Error(err))
	}
}
