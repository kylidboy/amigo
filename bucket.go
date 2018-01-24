package amigo

import (
	"container/list"
	"time"
)

const (
	BucketCap = 8
)

type bucket struct {
	list *list.List

	// current length of the list
	len int

	// upper limits of the list
	cap int

	// refreshing rate
	nextRefreshAt time.Time
}

func NewBucket(cap int) *bucket {
	return &bucket{
		cap:  cap,
		list: list.New(),
	}
}

func (b *bucket) NeedRefresh() bool {
	return time.Now().After(b.nextRefreshAt)
}

func (b *bucket) RefreshFront(r RefreshHandler) {
	if !b.NeedRefresh() {
		return
	}

	front := b.list.Front()
	val := front.Value.(*NodeInfo)

	if r(val) {
		b.list.MoveToBack(front)
	} else {
		b.list.Remove(front)
		b.len--
	}

	b.nextRefreshAt = time.Now().Add(10 * time.Minute)
}

func (b *bucket) IsFull() bool {
	return b.len == b.cap
}

func (b *bucket) Append(v interface{}) bool {
	if !b.IsFull() {
		b.len++
		b.list.PushBack(v)
		return true
	}
	return false
}

func (b *bucket) Destroy() {
	b.list = nil
	b.len = 0
}
