package amigo

import (
	"math/big"
	"math/rand"
	"net"
	"net/rpc"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	// K is just the k-bucket factor
	K = 20

	// Alpha is a system-wide currency factor
	Alpha = 3

	PingTimeout = 10 * time.Second
)

type (
	// ID defines the the type of ID, fixed to [20]byte
	ID [20]byte

	// Amigo
	Amigo struct {
		mtx             sync.Mutex
		self            ID
		buckets         [160]*Bucket
		closestNonEmpty int
	}
)

type AliveChecker func(addr net.Addr) (bool, error)

var (
	defaultAliveCheck AliveChecker = func(addr net.Addr) (bool, error) {
		conn, err := net.DialTimeout(addr.Network(), addr.String(), PingTimeout)
		defer conn.Close()
		if err != nil {
			return false, err
		}
		return true, nil
	}
)

// GetDistanceIndex calculates the distance between two IDs, return the index of the k-bucket
// index is of course between 0 and 160
// [0] is the most remote group
func GetDistanceIndex(self, other ID) int {
	idx := 0
LOOP:
	for i := range self {
		byteDist := self[i] ^ other[i]
		for j := 7; j >= 0; j-- {
			if (0x01<<uint(j))&byteDist != 0x00 {
				idx += 7 - j
				break LOOP
			}
		}
		idx += 8
	}
	return idx
}

func GetDistance(idA, idB ID) *big.Int {
	dist := [20]byte{}
	for i := range idA {
		dist[i] = idA[i] ^ idB[i]
	}
	return big.NewInt(0).SetBytes(dist[:])
}

func GetClosestTo(id ID, nodes []NodeInfo) (NodeInfo, *big.Int) {
	closest := GetDistance(id, nodes[0].ID)
	idx := 0
	for i, n := range nodes {
		dist := GetDistance(id, n.ID)
		if closest.Cmp(dist) == 1 {
			closest = dist
			idx = i
		}
	}
	return nodes[idx], closest
}

func NewAmigo(id ID) *Amigo {
	return &Amigo{
		self:            id,
		closestNonEmpty: -1,
	}
}

// GetKClosest get my own K closest nodes from local buckets
func (a *Amigo) GetKClosest() []NodeInfo {
	ret := make([]NodeInfo, 0, K)
	idx := 0
	if a.closestNonEmpty == -1 {
		idx = len(a.buckets) - 1
	} else {
		idx = a.closestNonEmpty
	}
	for ; idx >= 0 && len(ret) < K; idx-- {
		if a.buckets[idx] != nil && a.buckets[idx].Size() > 0 {
			ret = append(ret, a.buckets[idx].GetN(Alpha-len(ret))...)
		}
	}
	return ret
}

// GetKClosestTo get K closest nodes to the target id, and return a slice with K items
// TODO: use a better sort algorithm
func (a *Amigo) GetKClosestTo(target ID, nodes []NodeInfo) []NodeInfo {
	if len(nodes) <= K {
		return nodes
	}

	for j := 0; j < K; j++ {
		closest := GetDistance(a.self, nodes[j].ID)
		idx := j
		for i := j + 1; i < len(nodes); i++ {
			dist := GetDistance(a.self, nodes[i].ID)
			if closest.Cmp(dist) == 1 {
				closest = dist
				idx = i
			}
		}
		nodes[j], nodes[idx] = nodes[idx], nodes[j]
	}

	return nodes[:K]
}

func (a *Amigo) Add(val NodeInfo) (ok bool, err error) {
	if val.ID == a.self {
		return false, errors.New("ignore node itself")
	}

	idx := GetDistanceIndex(a.self, val.ID)
	if a.buckets[idx] == nil {
		a.buckets[idx] = NewBucket(K)
		if a.closestNonEmpty > idx {
			a.closestNonEmpty = idx
		}
	}

	if added := a.buckets[idx].Append(val, defaultAliveCheck); added {
		return true, nil
	}

	return false, nil
}

func (a *Amigo) Lookup(target ID) []NodeInfo {
	nodes := a.GetKClosest() // my K closest friends
	a.lookup(target, nodes)
}

func (a *Amigo) getAlphaNodesByRTT(nodes []NodeInfo) []NodeInfo {
	if len(nodes) <= Alpha {
		return nodes
	}
	sort.Sort(byRTT(nodes))
	return nodes[:Alpha]
}

func (a *Amigo) getAlphaNodesByRand(nodes []NodeInfo) []NodeInfo {
	if len(nodes) <= Alpha {
		return nodes
	}

	perm := rand.Perm(len(nodes))
	ret := make([]NodeInfo, 0, Alpha)
	for i := range perm {
		ret = append(ret, nodes[perm[i]])
	}
	return ret
}

func (a *Amigo) lookup(target ID, nodes []NodeInfo) {
	closestNode, closestDist := GetClosestTo(target, nodes) // closest info in this round
	seen := make(map[ID]struct{})

	if len(nodes) > K {
		nodes = a.GetKClosestTo(target, nodes)
	}

	for len(seen) < K && len(nodes) > 0 {
		alphas := a.getAlphaNodesByRand(nodes)
		nodes = removeFromSlice(alphas, nodes)
		newNodes, seen := a.query(target, alphas, seen)

		_, dist := GetClosestTo(target, newNodes)
		if closestDist.Cmp(dist) == -1 {
			newNodes, seen = a.query(target, nodes, seen)
		} else {
			// getting closer
			a.lookup(target, newNodes)
		}
	}

}

func (a *Amigo) Start() {

}

func (a *Amigo) Ping(val NodeInfo) bool {
	return false
}

func (a *Amigo) removeSeen(seen map[ID]struct{}, cand []NodeInfo) []NodeInfo {
	toBeDel := make([]int, 0)
	for i, n := range cand {
		if _, ok := seen[n.ID]; ok {
			toBeDel = append(toBeDel, i)
		}
	}
	ret := make([]NodeInfo, len(cand)-len(toBeDel))
	dst, src := 0, 0
	for i := range toBeDel {
		copy(ret[dst:dst+toBeDel[i]-src], cand[src:toBeDel[i]])
		dst += toBeDel[i] - src
		src = toBeDel[i] + 1
	}
	return ret
}

func (a *Amigo) query(queryID ID, queryNodes []NodeInfo, seen map[ID]struct{}) ([]NodeInfo, map[ID]struct{}) {
	for i := range queryNodes {
		seen[queryNodes[i].ID] = struct{}{}
	}

	newNodes := fromRPC()

	for _, n := range newNodes {
		a.Add(n)
	}
	return newNodes, seen
}

func removeFromSlice(toRemove []NodeInfo, slice []NodeInfo) []NodeInfo {
	for r := range toRemove {
		for s := range slice {
			if toRemove[r].ID == slice[s].ID {
				if s == 0 {
					slice = slice[1:]
				} else if s == len(slice)-1 {
					slice = slice[:s]
				} else {
					slice = append(slice[0:s], slice[s+1:]...)
				}
				break
			}
		}
	}
	return slice
}

func (a *Amigo) Print() {

}
