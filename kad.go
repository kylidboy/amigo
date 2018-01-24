package amigo

import (
	"net"
)

type (
	RefreshHandler func(info *NodeInfo) (aliveOrDead bool)

	ID       [20]byte
	Distance [20]byte

	NodeInfo struct {
		ID       ID
		Distance Distance
		Address  net.Addr
	}

	kad struct {
		root *kadNode

		refreshHandler RefreshHandler
	}
)

func XORDistance(id1, id2 ID) (distance Distance) {
	for i := range id1 {
		distance[i] = id1[i] ^ id2[i]
	}
	return
}

func NewKadTop(r RefreshHandler) *kad {
	return &kad{
		root:           NewKadNode(NewKadPrefix()),
		refreshHandler: r,
	}
}

func (top *kad) Print() {
	top.root.Print()
}

func (top *kad) Insert(val NodeInfo) (ok bool, err error) {
	return top.root.insert(0, &val)
}
