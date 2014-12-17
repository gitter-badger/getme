package sources

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
)

type ratings struct {
	Votes int `json:"votes"`
}

type traktMatch struct {
	FoundTitle string  `json:"title"`
	Ratings    ratings `json:"ratings"`
}

func (t traktMatch) Title() string {
	return t.FoundTitle
}

type traktMatches []traktMatch

func (tm traktMatches) BestMatch() Match {
	return tm[0]
}

type byRating []traktMatch

func (a byRating) Len() int           { return len(a) }
func (a byRating) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byRating) Less(i, j int) bool { return a[i].Ratings.Votes > a[j].Ratings.Votes }

var traktURL = "http://api.trakt.tv/search/shows.json/5bc6254d3bbde304a49557cf2845d921"

func constructUrl(query string) string {
	escapedQuery := url.Values{}
	escapedQuery.Add("query", query)
	return traktURL + "?query=" + escapedQuery.Encode()
}

func Search(query string) (Matches, error) {
	resp, err := http.Get(constructUrl(query))
	if err != nil {
		return nil, err //TODO retry a couple of times when it's a timeout.
	}
	if resp.StatusCode != 200 {
		return nil, errors.New("Search return non 200 status code")
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var ms []traktMatch
	err = json.Unmarshal(body, &ms)
	if err != nil {
		return nil, err
	}

	sort.Sort(byRating(ms))

	return traktMatches(ms), nil
}