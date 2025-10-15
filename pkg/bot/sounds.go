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

func (srq *soundsRandQ) iter(gid string) <-chan store.Sound {
	c := make(chan store.Sound)
	go func() {
		defer close(c)

		srq.Lock()
		defer srq.Unlock()

		sounds, ok := srq.s[gid]
		if !ok {
			return
		}

		for _, s := range sounds {
			c <- s
		}
	}()
	return c
}

var srq = soundsRandQ{
	s: map[string][]store.Sound{},
}

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

	var fullSound store.Sound
	var fullRatio int
	var partialSound store.Sound
	var partialRatio int
	for sound := range srq.iter(gid) {
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
