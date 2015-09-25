// Package torrents provides the ability to search for torrents given a
// list of required items.
package torrents

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/haarts/getme/store"
)

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
	Search(*store.Show) ([]Torrent, error)
}

var searchEngines = map[string]SearchEngine{
	"kickass":      Kickass{},
	"torrentcd":    TorrentCD{},
	"extratorrent": ExtraTorrent{},
}

// TODO this is only a starting point for pull torrents for the same search
// engines. I need to come up with a way to pick the best on duplciates.
func Search(show *store.Show) ([]Torrent, error) {
	var torrents []Torrent
	var lastError error
	for _, searchEngine := range searchEngines {
		ts, err := searchEngine.Search(show)
		torrents = append(torrents, ts...)
		lastError = err
	}
	return torrents, lastError
}

var seasonQueryAlternatives = map[string]func(string, *store.Season) string{
	"%s season %d": func(title string, season *store.Season) string {
		return fmt.Sprintf("%s season %d", title, season.Season)
	},
}

var episodeQueryAlternatives = map[string]func(string, *store.Episode) string{
	"%s S%02dE%02d": func(title string, episode *store.Episode) string {
		return fmt.Sprintf("%s S%02dE%02d", title, episode.Season(), episode.Episode)
	},
	"%s %dx%d": func(title string, episode *store.Episode) string {
		return fmt.Sprintf("%s %dx%d", title, episode.Season(), episode.Episode)
	},
	// This is a bit of a gamble. I, now, no longer make the
	// distinction between a daily series and a regular one:
	"%s %d %02d %02d": func(title string, episode *store.Episode) string {
		y, m, d := episode.AirDate.Date()
		return fmt.Sprintf("%s %d %02d %02d", title, y, m, d)
	},
}

var titleMorphers = [...]func(string) string{
	func(title string) string { //noop
		return title
	},
	func(title string) string {
		re := regexp.MustCompile("[^ a-zA-Z0-9]")
		newTitle := string(re.ReplaceAll([]byte(title), []byte("")))
		return newTitle
	},
	func(title string) string {
		return truncateToNParts(title, 5)
	},
	func(title string) string {
		return truncateToNParts(title, 4)
	},
	func(title string) string {
		return truncateToNParts(title, 3)
	},
}

func truncateToNParts(title string, n int) string {
	parts := strings.Split(title, " ")
	if len(parts) < n {
		return title
	}
	return strings.Join(parts[:n], " ")
}
