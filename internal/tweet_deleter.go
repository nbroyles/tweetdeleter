package internal

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"go.uber.org/zap"
)

// TweetDeleter deletes all tweets based on the parameters provided
type TweetDeleter struct {
	username  string
	password  string
	startDate string
	endDate   string
	logger    *zap.Logger
}

type TweetDeleterOptions struct {
	Username  string
	Password  string
	StartDate string
	EndDate   string
	Logger    *zap.Logger
}

// NewTweetDeleter creates a new TweetDeleter object
func NewTweetDeleter(opts TweetDeleterOptions) (*TweetDeleter, error) {
	return &TweetDeleter{
		username:  opts.Username,
		password:  opts.Password,
		startDate: opts.StartDate,
		endDate:   opts.EndDate,
		logger:    opts.Logger,
	}, nil
}

// Run starts the tweet deletion process. Run executes until
// all tweets are deleted or a fatal error occurs.
func (t *TweetDeleter) Run() error {
	// Create chrome instance
	ctx, cancel := chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", false),
			chromedp.Flag("auto-open-devtools-for-tabs", false))...,
	)
	ctx, cancel = chromedp.NewContext(ctx, chromedp.WithLogf(log.Printf))
	defer cancel()

	// Login to x.com
	if err := chromedp.Run(ctx, t.login()); err != nil {
		return fmt.Errorf("error while attempting to login: %w", err)
	}
	t.logger.Info("successfully logged in", zap.String("username", t.username))

	// Search provided date range
	if err := chromedp.Run(ctx, t.searchTweets()); err != nil {
		return fmt.Errorf("error while attempting to search for tweets: %w", err)
	}
	t.logger.Info("searched for latest tweets",
		zap.String("startDate", t.startDate), zap.String("endDate", t.endDate))

	// Loop through tweets and delete them
	t.logger.Info("commencing deleting tweets...")
	for i := 1; ; i++ {
		var tweets []*cdp.Node
		if err := chromedp.Run(ctx, chromedp.Nodes("article[data-testid=\"tweet\"]", &tweets)); err != nil {
			return fmt.Errorf("failed to retrieve tweets for deleting: %w", err)
		}

		if err := chromedp.Run(ctx, t.deleteTweet()); err != nil {
			return err
		}

		if i%10 == 0 {
			t.logger.Info(fmt.Sprintf("%d tweets deleted", i))
		}

		// We've deleted our last tweet so let's bail
		if len(tweets) == 1 {
			t.logger.Info("no more tweets to delete from provided time range", zap.Int("tweetsDeleted", i))
			break
		}
	}

	return nil
}

func (t *TweetDeleter) login() chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate("https://twitter.com/i/flow/login"),
		chromedp.WaitVisible("input[name=\"text\"]"),
		chromedp.Click("input[name=\"text\"]"),
		chromedp.SendKeys("input[name=\"text\"]", t.username),
		chromedp.Sleep(1 * time.Second), // NB: this may be unnecessary
		chromedp.Click("div[role=\"button\"]:nth-of-type(6)"),
		chromedp.WaitVisible("input[name=\"password\"]"),
		chromedp.Click("input[name=\"password\"]"),
		chromedp.SendKeys("input[name=\"password\"]", t.password),
		chromedp.Click("div[data-testid=\"LoginForm_Login_Button\"]"),
		chromedp.WaitVisible("a[href=\"/explore\"]"),
	}
}

func (t *TweetDeleter) searchTweets() chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate("https://twitter.com/explore"),
		chromedp.Click("input[data-testid=\"SearchBox_Search_Input\"]"),
		chromedp.SendKeys("input[data-testid=\"SearchBox_Search_Input\"]",
			fmt.Sprintf("from:%s since:%s until:%s"+kb.Enter, t.username, t.startDate, t.endDate)),
		// TODO: check that this search time range actually has tweets. Right now, will just hang if not.
		chromedp.WaitVisible("article[data-testid=\"tweet\"]"),
		chromedp.Click("a[href*=\"live\"][role=\"tab\"]"), // Click the "Latest" tab
		chromedp.WaitVisible("article[data-testid=\"tweet\"]"),
	}
}

func (t *TweetDeleter) deleteTweet() chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Click("article div[aria-label=\"More\"]"),
		chromedp.Sleep(1 * time.Second), // this worked -- for some reason the wait for visible condition below didn't quite work
		chromedp.WaitVisible("div[data-testid=\"Dropdown\"]"),
		chromedp.Click("div[role=\"menuitem\"]:first-child"),
		chromedp.WaitVisible("div[role=\"button\"][data-testid=\"confirmationSheetConfirm\"]"),
		chromedp.Click("div[role=\"button\"][data-testid=\"confirmationSheetConfirm\"]"),
	}
}
