package amigo

import (
	"fmt"

	"github.com/pkg/errors"
)

type kadPrefix struct {
	lastByteLen uint
	bytes       []byte
}

type kadNode struct {
	left   *kadNode
	right  *kadNode
	prefix kadPrefix
	bucket *bucket
}

func NewKadPrefix() kadPrefix {
	return kadPrefix{
		bytes: make([]byte, 0),
	}
}

func NewKadNode(prefix kadPrefix) *kadNode {
	return &kadNode{
		prefix: prefix,
	}
}

func (n *kadNode) Print() {
	tabs := len(n.prefix.bytes)*8 - 8 + int(n.prefix.lastByteLen)
	indent := ""
	for i := 0; i < tabs; i++ {
		indent += "\t"
	}

	if !n.IsLeaf() {
		fmt.Println(indent, "/\\")
		n.left.Print()
		n.right.Print()
	} else {
		fmt.Println(indent, "len: ", n.bucket.len)
		fmt.Println(indent, "prefix: ", n.prefix.String())

		e := n.bucket.list.Front()
		for {
			val := e.Value.(*NodeInfo)
			fmt.Printf("%s%08b: %X\n", indent, val.Distance, val.ID)
			if e = e.Next(); e == nil {
				break
			}
		}
	}
}

func (n *kadNode) IsLeaf() bool {
	return n.left == nil && n.right == nil
}

func (n *kadNode) split() error {
	if n.left != nil || n.right != nil {
		return errors.New("node is not leaf")
	}

	leftBucket := NewBucket(BucketCap)
	rightBucket := NewBucket(BucketCap)
	leftBucket.nextRefreshAt = n.bucket.nextRefreshAt
	rightBucket.nextRefreshAt = n.bucket.nextRefreshAt

	{
		c := n.bucket.list.Front()
		for c != nil {
			var bkt *bucket
			dis := c.Value.(*NodeInfo).Distance
			if n.prefix.IsNil() {
				if dis[0]&(0x01<<7) > 0x00 {
					bkt = leftBucket
				} else {
					bkt = rightBucket
				}
			} else {
				if dis[len(n.prefix.bytes)-1]&(0x01<<(7-n.prefix.lastByteLen)) > 0x00 {
					bkt = leftBucket
				} else {
					bkt = rightBucket
				}
			}
			bkt.list.PushBack(c.Value)
			bkt.len++

			c = c.Next()
		}
	}

	n.bucket.Destroy()
	n.bucket = nil

	n.left = NewKadNode(n.prefix.grow(1))
	n.right = NewKadNode(n.prefix.grow(0))
	n.left.bucket = leftBucket
	n.right.bucket = rightBucket

	return nil
}

func (n *kadNode) insert(offset uint, v *NodeInfo) (ok bool, err error) {
	if n.IsLeaf() {
		if n.bucket == nil {
			n.bucket = NewBucket(BucketCap)
		}

		if !n.bucket.IsFull() {
			n.bucket.Append(v)
			return true, nil
		}

		n.split()

		return n.insert(offset, v)
	}

	bytesID := offset / 8
	bitOffset := 7 - offset%8

	if v.Distance[bytesID]&(0x01<<bitOffset) == 0 {
		// go right
		return n.right.insert(offset+1, v)
	} else {
		// go left
		return n.left.insert(offset+1, v)
	}

}

func (p kadPrefix) grow(b byte) kadPrefix {
	cp := p.Copy()

	if cp.lastByteLen == 0 {
		cp.bytes = append(cp.bytes, 0x00)
	}

	if b == 1 {
		cp.bytes[len(cp.bytes)-1] |= (0x01 << (7 - cp.lastByteLen))
	}

	cp.lastByteLen++
	cp.lastByteLen %= 8

	return cp
}

func (p kadPrefix) Copy() kadPrefix {
	cp := kadPrefix{
		bytes:       make([]byte, len(p.bytes)),
		lastByteLen: p.lastByteLen,
	}
	copy(cp.bytes, p.bytes)
	return cp
}

func (p kadPrefix) IsNil() bool {
	return len(p.bytes) == 0 && p.lastByteLen == 0
}

func (p kadPrefix) String() string {
	prefix := ""
	for bi := 0; bi < len(p.bytes)-1; bi++ {
		for i := 7; i >= 0; i-- {
			if p.bytes[bi]&(0x01<<uint(i)) > 0x00 {
				prefix += "1"
			} else {
				prefix += "0"
			}
		}
	}
	for i := 0; uint(i) < p.lastByteLen; i++ {
		if p.bytes[len(p.bytes)-1]&(0x01<<(7-uint(i))) > 0x00 {
			prefix += "1"
		} else {
			prefix += "0"
		}
	}

	return prefix
}
