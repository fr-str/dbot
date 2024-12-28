package player

import (
	"fmt"
	"sync"
)

type Audio struct {
	Link     string
	Filepath string
}

type list struct {
	l         sync.Mutex
	nextAudio chan *Audio
	list      []Audio
	idx       int
}

func newList() list {
	return list{
		l:         sync.Mutex{},
		nextAudio: make(chan *Audio, 1),
		list:      make([]Audio, 0, 10),
	}
}

func (l *list) add(link string) *Audio {
	l.l.Lock()
	l.list = append(l.list, Audio{Link: link})
	ret := &l.list[len(l.list)-1]
	l.l.Unlock()
	return ret
}

func (l *list) next() int {
	e := l.list[l.idx]
	fmt.Println("[dupa] l.idx: ", l.idx)
	fmt.Println("[dupa] len(l.list): ", len(l.list))
	if l.idx >= len(l.list) {
		l.idx = 0
	}
	l.idx++

	l.nextAudio <- &e
	return l.idx
}

func (l *list) peek() *Audio {
	if !l.more() {
		return &l.list[0]
	}
	return &l.list[l.idx+1]
}

func (l *list) more() bool {
	return !(l.idx+1 >= len(l.list))
}

func (l *list) current() *Audio {
	return &l.list[l.idx]
}

func (l *list) len() int {
	return len(l.list)
}
