package amigo

import (
	"container/list"
	"math/rand"
	"net"
	"time"
)

type Bucket struct {
	list *list.List

	// upper limits of the list
	cap int

	// refreshing rate
	nextRefreshAt time.Time
}

func NewBucket(cap int) *Bucket {
	return &Bucket{
		cap:           cap,
		list:          list.New(),
		nextRefreshAt: time.Now().Add(10 * time.Minute),
	}
}

func (b *Bucket) needRefresh() bool {
	return time.Now().After(b.nextRefreshAt)
}

func (b *Bucket) Size() int {
	return b.list.Len()
}

func (b *Bucket) Refresh(isAlive AliveChecker) bool {
	if !b.needRefresh() {
		return false
	}

	front := b.list.Front()
	val := front.Value.(*NodeInfo)

	if alive, _ := isAlive(val.Address); alive {
		b.list.MoveToBack(front)
	} else {
		b.list.Remove(front)
	}

	b.nextRefreshAt = time.Now().Add(10 * time.Minute)

	return true
}

func (b *Bucket) IsFull() bool {
	return b.list.Len() == b.cap
}

func (b *Bucket) Append(v interface{}, isAlive AliveChecker) bool {
	if !b.IsFull() {
		b.list.PushBack(v)
		return true
	}

	frt := b.list.Front()
	ninfo := frt.Value.(NodeInfo)
	if alive, _ := isAlive(ninfo.Address); !alive {
		b.list.Remove(frt)
		b.list.PushBack(v)
		return true
	}

	b.list.MoveToBack(frt)

	return false
}

// GetAll returns all NodeInfos in this bucket
func (b *Bucket) GetAll() []NodeInfo {
	ret := make([]NodeInfo, 0, b.cap)
	if b.list.Len() == 0 {
		return nil
	}
	for cur := b.list.Front(); cur != nil; cur = cur.Next() {
		ret = append(ret, cur.Value.(NodeInfo))
	}
	return ret
}

// GetN returns n out of all NodeInfo's randomly
func (b *Bucket) GetN(n int) []NodeInfo {
	all := b.GetAll()
	if len(all) <= n {
		return all
	}
	p := rand.Perm(len(all))
	ret := make([]NodeInfo, n)
	for i := range ret {
		ret[i] = all[p[i]]
	}
	return ret
}

// Destroy emptys the bucket list
func (b *Bucket) Destroy() {
	for i := b.list.Len(); i > 0; i-- {
		b.list.Remove(b.list.Front())
	}
}
