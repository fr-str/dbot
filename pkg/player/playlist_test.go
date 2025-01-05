package player

import (
	"testing"

	"github.com/matryer/is"
)

func TestPlaylist(t *testing.T) {
	is := is.New(t)
	l := list{}

	// add elements
	l.add("1")
	is.Equal(1, l.len())

	l.add("2")
	is.Equal(2, l.len())
	l.add("3")
	l.add("4")

	// test wrap around
	l.next()
	is.Equal(false, l.more())
	l.next()
	is.Equal(4, l.len())
	is.Equal(true, l.more())
}
