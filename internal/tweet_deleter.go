package internal

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"go.uber.org/zap"
)

// TweetDeleter deletes all tweets based on the parameters provided
type TweetDeleter struct {
	username  string
	password  string
	startDate time.Time
	endDate   time.Time
	logger    *zap.Logger
}

type TweetDeleterOptions struct {
	Username  string
	Password  string
	StartDate time.Time
	EndDate   time.Time
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

	// Search and delete tweets in 7 day chunks. Larger chunks, like a year, tend to not return
	// all available tweets
	for since, until := t.startDate, t.startDate; until.Before(t.endDate); since = until {
		until = since.Add(7 * 86400 * time.Second) // 7 days
		if until.After(t.endDate) {
			until = t.endDate
		}

		// Search provided date range
		if err := chromedp.Run(ctx, t.searchTweets(since, until)); err != nil {
			return fmt.Errorf("error while attempting to search for tweets: %w", err)
		}
		t.logger.Info("searched for latest tweets",
			zap.Time("startDate", since), zap.Time("endDate", until))

		// Check to see if search yielding any results
		var emptyState bool
		err := chromedp.Run(ctx,
			// Give the search results some time to load.
			// TODO: Have a more reliable check here that's not just waiting
			chromedp.Sleep(3*time.Second),
			// This is very hacky. There's not a clean way to just check if an element exists so
			// we query for a selector, indicate that we're ok with 0 elements being returned so we don't wait
			// indefinitely, and then we add a custom wait QueryOption to see how many elements were found.
			// Either way, we return a non-nil slice of nodes so that the check doesn't repeat indefinitely.
			chromedp.Query(
				"div[data-testid=\"emptyState\"]",
				chromedp.AtLeast(0),
				chromedp.WaitFunc(func(ctx context.Context, frame *cdp.Frame, id runtime.ExecutionContextID, nid ...cdp.NodeID) ([]*cdp.Node, error) {
					if len(nid) > 0 {
						emptyState = true
					}
					return []*cdp.Node{}, nil
				}),
			))
		if err != nil {
			return fmt.Errorf("error checking if search returned tweets: %w", err)
		}

		if emptyState {
			t.logger.Info("no tweets found for time range. skipping to next",
				zap.Time("startDate", since), zap.Time("endDate", until))
			continue
		}

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
	}

	return nil
}

func (t *TweetDeleter) login() chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate("https://twitter.com/i/flow/login"),
		chromedp.Click("input[name=\"text\"]", chromedp.NodeVisible),
		chromedp.SendKeys("input[name=\"text\"]", t.username),
		chromedp.Sleep(1 * time.Second), // NB: this may be unnecessary
		chromedp.Click("div[role=\"button\"]:nth-of-type(6)"),
		chromedp.Click("input[name=\"password\"]", chromedp.NodeVisible),
		chromedp.SendKeys("input[name=\"password\"]", t.password),
		chromedp.Click("div[data-testid=\"LoginForm_Login_Button\"]"),
		chromedp.WaitVisible("a[href=\"/explore\"]"),
	}
}

func (t *TweetDeleter) searchTweets(since, until time.Time) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate("https://twitter.com/explore"),
		chromedp.Click("input[data-testid=\"SearchBox_Search_Input\"]"),
		chromedp.SendKeys("input[data-testid=\"SearchBox_Search_Input\"]",
			fmt.Sprintf(
				"from:%s since:%s until:%s"+kb.Enter, t.username, since.Format(time.DateOnly), until.Format(time.DateOnly),
			)),
		chromedp.Click("a[href*=\"live\"][role=\"tab\"]", chromedp.NodeVisible), // Click the "Latest" tab
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
