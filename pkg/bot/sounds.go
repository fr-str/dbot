package dbot

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"dbot/pkg/store"

	"github.com/fr-str/log"
	fuzzy "github.com/paul-mannino/go-fuzzywuzzy"
)

var ErrSoundNotFound = errors.New("sound not found")

func findSound(db *store.Queries, name string, gid string) ([]store.Sound, error) {
	var ss []store.Sound
	sounds, err := db.SelectSounds(context.Background(), gid)
	if err != nil {
		return ss, fmt.Errorf("db select failed: %w", err)
	}
	if len(sounds) == 0 {
		return ss, fmt.Errorf("no sounds in soundboard")
	}

	name = strings.ToLower(strings.ReplaceAll(name, " ", ""))
	if strings.HasPrefix(name, "sound") || strings.HasPrefix(name, "event") {
		if len(name) < 5 {
			return append(ss, randSound(sounds)), nil
		}

		num := name[5:]
		numInt, err := strconv.Atoi(num)
		if err != nil {
			return append(ss, randSound(sounds)), nil
		}

		for range min(numInt, 2137) {
			ss = append(ss, randSound(sounds))
		}
		return ss, nil

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
		return append(ss, fullSound), nil
	}

	// if sound not found return partial match
	if partialRatio >= 80 {
		return append(ss, partialSound), nil
	}

	return ss, ErrSoundNotFound
}

func randSound(sounds []store.Sound) store.Sound {
	randInt, _ := rand.Int(rand.Reader, big.NewInt(int64(len(sounds))))
	log.Trace("randSound", log.Any("randInt", randInt))
	return sounds[randInt.Int64()]
}
