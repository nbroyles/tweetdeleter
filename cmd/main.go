package main

import (
	"flag"
	"log"

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

	td, err := internal.NewTweetDeleter(internal.TweetDeleterOptions{
		Username:  *username,
		Password:  *password,
		StartDate: *startDate,
		EndDate:   *endDate,
		Logger:    logger,
	})
	if err != nil {
		logger.Fatal("could not create TweetDeleter", zap.Error(err))
	}
	if err = td.Run(); err != nil {
		logger.Error("error running TweetDeleter", zap.Error(err))
	}
}
