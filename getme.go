package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/haarts/getme/search_engines"
	"github.com/haarts/getme/sources"
	"github.com/haarts/getme/store"
	"github.com/haarts/getme/ui"
)

func handleShow(show *sources.Match) error {
	store := store.Open()
	defer store.Close()

	store.CreateShow(*show)

	// Fetch the seasons associated with the found show.
	seasons, err := ui.LookupSeasons(*show)
	if err != nil {
		fmt.Println("We've encountered a problem looking up seasons for the show. The error:")
		fmt.Println(" ", err)
		return err
	}
	episodes := sources.CreateEpisodes(seasons)

	// We have two entry points. One on the first run and one when running as daemon.
	// So we create episodes based on seasons always. Then look at the disk/store and figure out
	// what is missing.
	// Then we take that list of episodes and create search queries.
	// The types defined in sources pkg are wrong.
	// * Match -> Show (ORLY?)
	// * intro Movie
	// * intro Episode
	// How can I make main work with both Show and Movie? An interface? Then I need to intro
	// getters/setters...
	//

	torrents, err := search_engines.Search(episodes)
	if err != nil {
		fmt.Println("Something went wrong looking for your episodes.", err)
		return err
	}
	for _, torrent := range torrents {
		download(string(torrent))
	}

	return nil
}

func main() {
	matches, err := ui.Search(ui.GetQuery())
	if err != nil {
		fmt.Println("We've encountered a problem searching. The error:")
		fmt.Println(" ", err)
	}
	if len(matches) == 0 {
		fmt.Println("We haven't found what you were looking for.")
		return
	}

	// Determine which show/movie ppl want.
	match := ui.DisplayBestMatchConfirmation(matches)
	if match == nil {
		match = ui.DisplayAlternatives(matches)
	}

	if match == nil {
		fmt.Println("We're sorry we couldn't find it for you.")
		return
	}

	// TODO Handle 'Movie' case.

	// Handle the 'Show' case.
	err = handleShow(match)
	if err != nil {
		return
	}
}

// This is an odd function here. Perhaps I'll group it with the 'getBody' function.
func download(url string) {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]
	fmt.Println("Downloading", url, "to", fileName)

	// TODO: check file existence first with io.IsExist
	output, err := os.Create("/tmp/getme/" + fileName)
	if err != nil {
		fmt.Println("Error while creating", fileName, "-", err)
		return
	}
	defer output.Close()

	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return
	}

	fmt.Println(n, "bytes downloaded.")
}
