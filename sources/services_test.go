package sources

import "testing"

func TestPopularShowAtIndex(t *testing.T) {
	oldShows := SHOWS
	SHOWS = [100]string{
		"one",
		"two",
		"three",
	}

	matches := []Show{
		Show{Title: "not one"},
		Show{Title: "two"},
	}

	if popularShowAtIndex(matches) != 1 {
		t.Error("Expected to find the popular show at index 1, got:", popularShowAtIndex(matches))
	}
	SHOWS = oldShows // reset
}
