package dbot

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"dbot/pkg/store"

	"github.com/fr-str/log"
	fuzzy "github.com/paul-mannino/go-fuzzywuzzy"
)

var ErrSoundNotFound = errors.New("sound not found")

func findSound(db *store.Queries, name string, gid string) (store.Sound, error) {
	sounds, err := db.SelectSounds(context.Background(), gid)
	if err != nil {
		return store.Sound{}, fmt.Errorf("db select failed: %w", err)
	}
	if len(sounds) == 0 {
		return store.Sound{}, fmt.Errorf("no sounds in soundboard")
	}

	name = strings.ToLower(strings.ReplaceAll(name, " ", ""))
	if name == "sound" || name == "event" {
		return randSound(sounds), nil
	}

	var fullSound store.Sound
	var fullRatio int
	var partialSound store.Sound
	var partialRatio int
	for _, sound := range sounds {
		for _, alias := range sound.Aliases {
			ratio := fuzzy.Ratio(alias, name)
			if ratio > 80 && ratio > fullRatio {
				log.Debug("ratio fuzzy match", log.Int("ratio", ratio), log.String("alias", alias))
				fullRatio = ratio
				fullSound = sound
			}
			ratio = fuzzy.PartialRatio(alias, name)
			if ratio > 80 && ratio > partialRatio {
				log.Debug("partial ratio fuzzy match", log.Int("ratio", ratio), log.String("alias", alias))
				partialRatio = ratio
				partialSound = sound
			}
		}
	}

	// if sound found return it
	if fullRatio >= 80 {
		return fullSound, nil
	}

	// if sound not found return partial match
	if partialRatio >= 80 {
		return partialSound, nil
	}

	return store.Sound{}, ErrSoundNotFound
}

func randSound(sounds []store.Sound) store.Sound {
	randInt, _ := rand.Int(rand.Reader, big.NewInt(int64(len(sounds))))
	log.Trace("randSound", log.Any("randInt", randInt))
	return sounds[randInt.Int64()]
}
