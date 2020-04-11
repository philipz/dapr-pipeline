package main

import (
	"errors"
	"strings"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

const (
	maxTweets = 100
)

// Query represents Twitter search query in specific user context
type Query struct {

	// Text is the full text of the Twitter search query including operators
	// e.g. 'dapr AND microsoft'
	Text string `json:"text"`

	// Lang is the ISO 639-1 code which will be used to filter tweets
	Lang string `json:"lang"`

	// Count is the number of tweets to return (no paging for now)
	Count int `json:"count"`

	// SinceID is the id of the tweet to start search from
	// Set to the last tweet returned by this query in handler
	SinceID int64 `json:"-"`

	// Username is the Twitter username who's Token/Secrets are assciated with
	Username string `json:"user"`

	// Token is the Twitter AccessTokenKey
	Token string `json:"token"`

	// Secret is the Twitter AccessTokenSecrets
	Secret string `json:"secret"`
}

func (q *Query) validate() error {

	if q.Text == "" {
		return errors.New("empty search query text")
	}

	if q.Username == "" {
		return errors.New("empty search query user")
	}

	if q.Token == "" {
		return errors.New("empty search query token")
	}

	if q.Secret == "" {
		return errors.New("empty search query secret")
	}

	if q.Count == 0 {
		q.Count = maxTweets
	}

	if q.Count > 100 {
		logger.Printf("invalid query.count (want: 0-%d, got: %d), re-setting to max: %d",
			maxTweets, q.Count, maxTweets)
		q.Count = maxTweets
	}

	if q.Lang == "" {
		q.Lang = "en"
	}

	return nil

}

// SimpleTweet represents the Twiter query result item
type SimpleTweet struct {
	// ID is the string representation of the tweet ID
	ID int64 `json:"id"`
	// Query is the text of the original query
	Query string `json:"query"`
	// Author is the name of the tweet user
	Author string `json:"author"`
	// Content is the full text body of the tweet
	Content string `json:"content"`
	// Published is the parsed tweet create timestamp
	Published time.Time `json:"published"`
}

// SearchResult is the metadata from executed search
type SearchResult struct {
	// LastID is the last tweet ID
	SinceID int64 `json:"since_id"`
	// MaxID is the last tweet ID
	MaxID int64 `json:"max_id"`
	// Query is the text of the search query
	Query string `json:"query"`
	// Found is the number of items returned by search
	Found int `json:"items_found"`
	// Published is the number of items published
	Published int `json:"items_published"`
	// Duration is the number of items published
	Duration float64 `json:"search_duration"`
}

func search(q *Query) (r *SearchResult, err error) {

	if q == nil {
		return nil, errors.New("nil search query")
	}

	if err := q.validate(); err != nil {
		return nil, err
	}

	config := oauth1.NewConfig(consumerKey, consumerSecret)
	token := oauth1.NewToken(q.Token, q.Secret)

	httpClient := config.Client(oauth1.NoContext, token)
	tc := twitter.NewClient(httpClient)

	logger.Printf("searching for '%s' since id: %d", q.Text, q.SinceID)
	search, resp, err := tc.Search.Tweets(&twitter.SearchTweetParams{
		Query:           q.Text,
		Count:           maxTweets,
		Lang:            q.Lang,
		SinceID:         q.SinceID,
		IncludeEntities: twitter.Bool(true),
		TweetMode:       "extended",
		ResultType:      "popular",
	})

	if err != nil {
		logger.Printf("error on search: %v - %v", resp, err)
		return nil, err
	}

	r = &SearchResult{
		Query:    search.Metadata.Query,
		Found:    len(search.Statuses),
		SinceID:  q.SinceID,
		MaxID:    q.SinceID, // start with the previous max in case there is no more results
		Duration: search.Metadata.CompletedIn,
	}

	for _, t := range search.Statuses {
		t := &SimpleTweet{
			ID:        t.ID,
			Query:     q.Text,
			Author:    strings.TrimSpace(strings.ToLower(t.User.ScreenName)),
			Content:   t.FullText,
			Published: convertTwitterTime(t.CreatedAt),
		}
		if err = publishQueryResult(t); err != nil {
			return nil, err
		}
		r.Published++
		r.MaxID = t.ID
	}

	return r, nil

}

func convertTwitterTime(v string) time.Time {
	t, err := time.Parse(time.RubyDate, v)
	if err != nil {
		t = time.Now()
	}
	return t.UTC()
}