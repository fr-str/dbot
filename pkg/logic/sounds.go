package logic

import (
	"context"
	"fmt"
	"strings"

	"dbot/pkg/store"

	"github.com/fr-str/log"
	fuzzy "github.com/paul-mannino/go-fuzzywuzzy"
)

func FindSound(db *store.Queries, name string, gid string) (store.Sound, error) {
	sounds, err := db.SelectSounds(context.Background(), gid)
	if err != nil {
		return store.Sound{}, err
	}

	name = strings.ToLower(name)
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

	return store.Sound{}, fmt.Errorf("sound '%s' not found", name)
}
