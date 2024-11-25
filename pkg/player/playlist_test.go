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
	is.Equal(1, l.queueLen())

	l.add("2")
	is.Equal(2, l.len())
	is.Equal(2, l.queueLen())
	l.add("3")
	l.add("4")

	// getting audio from list
	e := l.next()
	is.Equal(e.Link, "1")
	is.Equal(4, l.len())
	is.Equal(3, l.queueLen())

	e = l.next()
	is.Equal(e.Link, "2")
	is.Equal(4, l.len())
	is.Equal(2, l.queueLen())

	// test wrap around
	l.next()
	l.next()
	is.Equal(4, l.len())
	is.Equal(4, l.queueLen())
}
