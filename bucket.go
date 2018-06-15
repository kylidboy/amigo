package amigo

import (
	"container/list"
	"math/rand"
	"time"
)

type Bucket struct {
	list *list.List

	cache idCache

	// upper limits of the list
	cap int

	// refreshing rate
	nextRefreshAt time.Time
}

type idCache = map[ID]struct{}

func NewBucket(cap int) *Bucket {
	return &Bucket{
		cap:           cap,
		list:          list.New(),
		nextRefreshAt: time.Now().Add(10 * time.Minute),
		cache:         make(idCache),
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
		if _, ok := b.cache[val.ID]; ok {
			delete(b.cache, val.ID)
		}
	}

	b.nextRefreshAt = time.Now().Add(10 * time.Minute)

	return true
}

func (b *Bucket) IsFull() bool {
	return b.list.Len() == b.cap
}

func (b *Bucket) Append(v NodeInfo, isAlive AliveChecker) bool {
	if !b.IsFull() {
		b.list.PushBack(v)
		b.cache[v.ID] = struct{}{}
		return true
	}

	frt := b.list.Front()
	ninfo := frt.Value.(NodeInfo)
	if alive, _ := isAlive(ninfo.Address); !alive {
		b.list.Remove(frt)
		if _, ok := b.cache[ninfo.ID]; ok {
			delete(b.cache, ninfo.ID)
		}
		b.list.PushBack(v)
		b.cache[v.ID] = struct{}{}
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
	b.cache = make(idCache)
}
