package player

import (
	"sync"
)

type Audio struct {
	Link     string
	Filepath string
}

type list struct {
	l         sync.Mutex
	nextAudio chan *Audio
	list      []*Audio
	idx       int
}

func newList() list {
	return list{
		l:         sync.Mutex{},
		nextAudio: make(chan *Audio, 1),
		list:      make([]*Audio, 0, 10),
	}
}

func (l *list) add(link string) {
	l.l.Lock()
	l.list = append(l.list, &Audio{Link: link})
	l.l.Unlock()
}

func (l *list) next() {
	e := l.list[l.idx]
	l.idx++
	if l.idx >= len(l.list) {
		l.idx = 0
	}
	l.nextAudio <- e
}

func (l *list) peek() *Audio {
	if l.queueLen() == 0 {
		return l.list[0]
	}
	return l.list[l.idx+1]
}

func (l *list) more() bool {
	return !(l.idx+1 == len(l.list))
}

func (l *list) current() *Audio {
	return l.list[l.idx]
}

func (l *list) len() int {
	return len(l.list)
}

// returns number of Audio elements from [l.idx:]
func (l *list) queueLen() int {
	return len(l.list[l.idx+1:])
}
