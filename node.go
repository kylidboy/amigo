package amigo

import (
	"net"
)

// NodeInfo defines the data structure stored in Amigo
type NodeInfo struct {
	ID      ID
	Address net.Addr
	Data    interface{}

	// round trip time
	rtt int
}

type byRTT []NodeInfo

func (n byRTT) Len() int {
	return len(n)
}

func (n byRTT) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

func (n byRTT) Less(i, j int) bool {
	return n[i].rtt < n[j].rtt
}
