package dbot

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"strconv"
	"strings"
	"sync"

	"dbot/pkg/store"

	"github.com/fr-str/log"
	fuzzy "github.com/paul-mannino/go-fuzzywuzzy"
)

var ErrSoundNotFound = errors.New("sound not found")

type soundsRandQ struct {
	sync.Mutex
	s map[string][]store.Sound
}

func (srq *soundsRandQ) refresh(db *store.Queries, gid string) {
	srq.Lock()
	defer srq.Unlock()
	if len(srq.s[gid]) != 0 {
		return
	}
	sounds, err := db.SelectSounds(context.Background(), gid)
	if err != nil {
		log.Error("db select failed: %w", err)
	}
	srq.s[gid] = sounds
}

func (srq *soundsRandQ) dropAndLoad(db *store.Queries, gid string) {
	srq.Lock()
	srq.s[gid] = nil
	srq.Unlock()
	srq.refresh(db, gid)
}

func (srq *soundsRandQ) rand(db *store.Queries, gid string) store.Sound {
	srq.refresh(db, gid)
	srq.Lock()
	defer srq.Unlock()
	sounds := srq.s[gid]
	randInt, _ := rand.Int(rand.Reader, big.NewInt(int64(len(sounds))))

	log.Trace("soundsRandQ.rand", log.Any("srq.s[gid]", len(srq.s[gid])), log.Any("randInt", randInt))
	v := sounds[randInt.Int64()]

	sounds[randInt.Int64()] = sounds[len(sounds)-1]
	sounds = sounds[:len(sounds)-1]
	srq.s[gid] = sounds
	return v
}

var srq = soundsRandQ{
	s: map[string][]store.Sound{},
}

const (
	minimumFuzzySoundNameLength = 4
	minimumFuzzySoundScore      = 85
	minimumFuzzySoundScoreGap   = 8
)

func findSound(db *store.Queries, name string, gid string) ([]store.Sound, error) {
	srq.refresh(db, gid)
	var ss []store.Sound
	name = strings.ToLower(strings.ReplaceAll(name, " ", ""))
	if strings.HasPrefix(name, "sound") || strings.HasPrefix(name, "event") {
		if len(name) < 5 {
			return append(ss, srq.rand(db, gid)), nil
		}

		num := name[5:]
		numInt, err := strconv.Atoi(num)
		if err != nil {
			return append(ss, srq.rand(db, gid)), nil
		}

		for range min(numInt, 2137) {
			ss = append(ss, srq.rand(db, gid))
		}

		return ss, nil
	}

	sounds, err := db.SelectSounds(context.Background(), gid)
	if err != nil {
		log.Error("db select failed: %w", err)
	}

	sound, found := matchSound(sounds, name)
	if found {
		return append(ss, sound), nil
	}

	return ss, ErrSoundNotFound
}

func matchSound(sounds []store.Sound, name string) (store.Sound, bool) {
	for _, sound := range sounds {
		for _, alias := range sound.Aliases {
			if normalize(alias) == name {
				return sound, true
			}
		}
	}

	prefixSoundIndex := -1
	for soundIndex, sound := range sounds {
		for _, alias := range sound.Aliases {
			alias = normalize(alias)
			if len(name) >= 3 && strings.HasPrefix(alias, name) {
				if prefixSoundIndex != -1 {
					return fuzzyMatchSound(sounds, name)
				}
				prefixSoundIndex = soundIndex
				break
			}
		}
	}
	if prefixSoundIndex != -1 {
		return sounds[prefixSoundIndex], true
	}

	return fuzzyMatchSound(sounds, name)
}

func fuzzyMatchSound(sounds []store.Sound, name string) (store.Sound, bool) {
	if len(name) < minimumFuzzySoundNameLength {
		return store.Sound{}, false
	}

	bestSoundIndex := -1
	bestScore := 0
	secondBestScore := 0
	for soundIndex, sound := range sounds {
		soundScore := 0
		for _, alias := range sound.Aliases {
			alias = normalize(alias)
			if len(alias) < minimumFuzzySoundNameLength {
				continue
			}

			candidate := alias
			if len(name) < len(alias) {
				candidate = alias[:len(name)]
			}
			soundScore = max(soundScore, fuzzy.Ratio(candidate, name))
		}

		if soundScore > bestScore {
			secondBestScore = bestScore
			bestScore = soundScore
			bestSoundIndex = soundIndex
			continue
		}
		secondBestScore = max(secondBestScore, soundScore)
	}

	if bestSoundIndex == -1 || bestScore < minimumFuzzySoundScore || bestScore-secondBestScore < minimumFuzzySoundScoreGap {
		return store.Sound{}, false
	}

	log.Debug("fuzzy sound match", log.Int("score", bestScore), log.String("name", name))
	return sounds[bestSoundIndex], true
}
