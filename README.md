# TweetDeleter

Repository for a program that allows you to purge old Tweets from your Twitter/X account based on a provided time range.

This program manipulates a non-headless Chrome to log into the provided account and delete matching tweets.

## Usage

Easiest thing to do is clone the repository and build the binary. From there it can be used how it's described below.

```
$ ./tweetdeleter -h
Usage of ./tweetdeleter:
  -end-date string
    	end date (inclusive) of time range to delete tweets. must be formatted as YYYY-MM-DD
  -password string
    	password for provided account
  -start-date string
    	start date of time range to delete tweets. must be formatted as YYYY-MM-DD
  -username string
    	x/twitter account to log into and delete tweets
```
