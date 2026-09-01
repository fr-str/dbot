package dbot

import (
	"testing"

	"dbot/pkg/store"

	"github.com/stretchr/testify/assert"
)

func TestMatchSound(t *testing.T) {
	sounds := []store.Sound{
		{Url: "oi.mp4", Aliases: []string{"oi"}},
		{Url: "meep.mp4", Aliases: []string{"meepvoiceinmyhead"}},
	}

	t.Run("matches short aliases exactly", func(t *testing.T) {
		sound, found := matchSound(sounds, "oi")

		assert.True(t, found)
		assert.Equal(t, "oi.mp4", sound.Url)
	})

	t.Run("matches an unambiguous alias prefix", func(t *testing.T) {
		sound, found := matchSound(sounds, "meepvoice")

		assert.True(t, found)
		assert.Equal(t, "meep.mp4", sound.Url)
	})

	t.Run("does not match a short alias inside another name", func(t *testing.T) {
		sound, found := matchSound([]store.Sound{
			{Url: "oi.mp4", Aliases: []string{"oi"}},
		}, "meepvoice")

		assert.False(t, found)
		assert.Empty(t, sound.Url)
	})

	t.Run("matches a typo in an alias prefix", func(t *testing.T) {
		sound, found := matchSound(sounds, "meepvoise")

		assert.True(t, found)
		assert.Equal(t, "meep.mp4", sound.Url)
	})

	t.Run("does not resolve an ambiguous fuzzy match", func(t *testing.T) {
		ambiguousSounds := []store.Sound{
			{Url: "one.mp4", Aliases: []string{"meepvoiceone"}},
			{Url: "two.mp4", Aliases: []string{"meepvoicetwo"}},
		}

		_, found := matchSound(ambiguousSounds, "meepvoic")

		assert.False(t, found)
	})
}
