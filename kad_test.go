package amigo

import (
	"crypto/rand"
	"fmt"
	"testing"
)

func genIDRand() (id ID) {
	if n, err := rand.Read(id[:]); n != 20 || err != nil {
		panic("genIDRand")
	}

	return
}

func TestXORDistance(t *testing.T) {
	id1 := genIDRand()
	id2 := genIDRand()

	fmt.Printf("ID1: %08b\nID2: %08b\n", id1, id2)

	dis := XORDistance(id1, id2)
	fmt.Printf("Distance: %08b\n", dis)
}

func refresh(info *NodeInfo) bool {
	return true
}

func TestKadTopInsert5(t *testing.T) {
	ids := make([]ID, 0)

	for i := 0; i < 5; i++ {
		ids = append(ids, genIDRand())
	}

	kad := NewKadTop(refresh)

	for _, n := range ids[1:] {
		info := NodeInfo{
			ID:       n,
			Distance: XORDistance(ids[0], n),
			Address:  nil,
		}
		kad.Insert(info)
	}

	kad.Print()
}

func TestKadTopInsert100(t *testing.T) {
	ids := make([]ID, 0)

	for i := 0; i < 100; i++ {
		ids = append(ids, genIDRand())
	}

	kad := NewKadTop(refresh)

	for _, n := range ids[1:] {
		info := NodeInfo{
			ID:       n,
			Distance: XORDistance(ids[0], n),
			Address:  nil,
		}
		kad.Insert(info)
	}

	kad.Print()
}
