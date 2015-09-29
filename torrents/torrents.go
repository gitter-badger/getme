// Package torrents provides the ability to search for torrents given a
// list of required items.
package torrents

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/haarts/getme/store"
)

// batchSize controls how many torrent will be fetch in 1 go. This is
// important when downloading very long running series.
const batchSize = 50

// Mark a piece of media as done. Currently only Show.
type Doner interface {
	Done()
}

type Torrent struct {
	URL             string
	OriginalName    string
	seeds           int
	AssociatedMedia Doner
}

type SearchEngine interface {
	Search(string) ([]Torrent, error)
	Name() string
}

var searchEngines = map[string]SearchEngine{
	"kickass":      NewKickass(),
	"torrentcd":    NewTorrentCD(),
	"extratorrent": ExtraTorrent{},
}

type queryJob struct {
	media   Doner
	snippet store.Snippet
	query   string
}

func Search(show *store.Show) ([]Torrent, error) {
	// torrents holds the torrents to complete a serie
	var torrents []Torrent
	queryJobs := createQueryJobs(show)
	for _, queryJob := range queryJobs {
		torrent, err := executeJob(queryJob.query)
		if err != nil {
			continue
		}

		torrent.AssociatedMedia = queryJob.media
		queryJob.snippet.Score = torrent.seeds
		// *ouch* this type switch is ugly
		switch queryJob.media.(type) {
		case *store.Season:
			show.StoreSeasonSnippet(queryJob.snippet)
		case *store.Episode:
			show.StoreEpisodeSnippet(queryJob.snippet)
		default:
			panic("unknown media type")
		}
		torrents = append(torrents, *torrent)
	}

	return torrents, nil
}

func executeJob(query string) (*Torrent, error) {
	// c emits the torrents found for one search request on one search engine
	c := make(chan []Torrent)
	for _, searchEngine := range searchEngines {
		go func(s SearchEngine) {
			torrents, err := s.Search(query)
			if err != nil {
				log.WithFields(log.Fields{
					"err":           err,
					"search_engine": s.Name(),
				}).Error("Search engine returned error")
			}
			// TODO filter isSeason
			torrents = applyFilter(torrents)
			c <- torrents
		}(searchEngine)
	}

	var torrentsPerQuery []Torrent
	timeout := time.After(5 * time.Second)
	for i := 0; i < len(searchEngines); i++ {
		select {
		case result := <-c:
			torrentsPerQuery = append(torrentsPerQuery, result...)
		case <-timeout:
			log.Error("Search timed out")
		}
	}

	if len(torrentsPerQuery) == 0 {
		return nil, fmt.Errorf("No torrents found for %s", query)
	}

	sort.Sort(bySeeds(torrentsPerQuery))
	bestTorrent := torrentsPerQuery[0]

	return &bestTorrent, nil
}

func createQueryJobs(show *store.Show) []queryJob {
	seasonQueries := queriesForSeasons(show)
	episodeQueries := queriesForEpisodes(show)
	return append(seasonQueries, episodeQueries...)
}

func queriesForEpisodes(show *store.Show) []queryJob {
	episodes := show.PendingEpisodes()
	sort.Sort(store.ByAirDate(episodes))
	min := math.Min(float64(len(episodes)), float64(batchSize))

	queries := []queryJob{}
	for _, episode := range episodes[0:int(min)] {
		snippet := selectEpisodeSnippet(show)

		query := episodeQueryAlternatives[snippet.FormatSnippet](snippet.TitleSnippet, episode)
		queries = append(queries, queryJob{snippet: snippet, query: query, media: episode})
	}
	return queries
}

func queriesForSeasons(show *store.Show) []queryJob {
	queries := []queryJob{}
	for _, season := range show.PendingSeasons() {
		// ignore Season 0, which are specials and are rarely found and/or
		// interesting.
		if season.Season == 0 {
			continue
		}

		snippet := selectSeasonSnippet(show)

		query := seasonQueryAlternatives[snippet.FormatSnippet](snippet.TitleSnippet, season)
		queries = append(queries, queryJob{snippet: snippet, query: query, media: season})
	}
	return queries
}

func applyFilter(torrents []Torrent) []Torrent {
	ok := []Torrent{}
	for _, torrent := range torrents {
		if isEnglish(torrent.OriginalName) {
			ok = append(ok, torrent)
		}
	}
	return ok
}

func isEnglish(fileName string) bool {
	lowerCaseFileName := strings.ToLower(fileName)
	// Too weak a check but it is the easiest. I hope there aren't any series
	// with 'french' in the title.
	if strings.Contains(lowerCaseFileName, "french") {
		return false
	}

	if strings.Contains(lowerCaseFileName, "spanish") {
		return false
	}

	if strings.Contains(lowerCaseFileName, "español") {
		return false
	}

	// Ignore Version Originale Sous-Titrée en FRançais. Hard coded, French subtitles.
	if strings.Contains(lowerCaseFileName, "vostfr") {
		return false
	}

	// Ignore Italian (ITA) dubs.
	regex := regexp.MustCompile(`\bITA\b`)
	if regex.MatchString(fileName) {
		return false
	}

	// Ignore hard coded (HC) subtitles.
	regex = regexp.MustCompile(`\bHC\b`)
	if regex.MatchString(fileName) {
		return false
	}

	return true
}

func selectBest(torrents []Torrent) *Torrent {
	return &(torrents[0]) //most peers
}

type bySeeds []Torrent

func (a bySeeds) Len() int           { return len(a) }
func (a bySeeds) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a bySeeds) Less(i, j int) bool { return a[i].seeds > a[j].seeds }
