package player

import (
	"errors"
	"testing"

	"github.com/matryer/is"
)

func TestPlayerErr(t *testing.T) {
	is := is.New(t)
	err := playerErr("dupa", errors.New("to dupa or not to dupa"))
	is.Equal(err.Error(), "player: dupa: to dupa or not to dupa")

	err = playerErr("dupa is %s", "xD", errors.New("to dupa or not to dupa"))
	is.Equal(err.Error(), "player: dupa is xD: to dupa or not to dupa")
}
