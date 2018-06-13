package amigo

import (
	"bytes"
	"math/big"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

const (
	K                 = 8
	Alpha             = 3
	ConnectionTimeout = 10
)

type (
	ID [20]byte

	// NodeInfo defines the data structure stored in Amigo
	NodeInfo struct {
		ID      ID
		Address net.Addr
		Data    interface{}
	}

	Amigo struct {
		mtx     sync.Mutex
		self    ID
		total   int
		buckets [160]*Bucket
	}
)

type AliveChecker func(addr net.Addr) (bool, error)

var (
	defaultAliveCheck AliveChecker = func(addr net.Addr) (bool, error) {
		conn, err := net.DialTimeout(addr.Network(), addr.String(), ConnectionTimeout*time.Second)
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
			if (0x01<<uint(j))&byteDist == 1 {
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
	closest := big.NewInt(0).SetBytes(bytes.Repeat([]byte{0xFF}, 20))
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
		self: id,
	}
}

func (a *Amigo) Print() {

}

func (a *Amigo) Add(val NodeInfo) (ok bool, err error) {
	if val.ID == a.self {
		return false, errors.New("ignore node itself")
	}

	idx := GetDistanceIndex(a.self, val.ID)
	if a.buckets[idx] == nil {
		a.buckets[idx] = NewBucket(K)
	}

	if added := a.buckets[idx].Append(val, defaultAliveCheck); added {
		a.total++
		return true, nil
	}

	return false, nil
}

func (a *Amigo) Lookup(nodeID ID) []NodeInfo {
	nodes := a.GetInitialCloseNodes(nodeID)
	a.lookup(nodeID, nodes)
}

func (a *Amigo) lookup(nodeID ID, querys []NodeInfo) {
	closestNode, closestDist := GetClosestTo(nodeID, querys)
	seen := make(map[ID]struct{})

BATCH:
	for j := 0; j < len(querys); j += Alpha {
		wg := sync.WaitGroup{}
		var resCHs []chan []NodeInfo
		if j+Alpha > len(querys) {
			wg.Add(len(querys) - j)
			resCHs = make([]chan []NodeInfo, len(querys)-j)
		} else {
			wg.Add(Alpha)
			resCHs = make([]chan []NodeInfo, Alpha)
		}

		mtx := sync.Mutex{}

		for i := 0; i < Alpha; i++ {
			if j+i > len(querys) {
				break
			}
			seen[querys[j+i].ID] = struct{}{}

			go func(ci int) {
				res, err := a.query(querys[j+ci], nodeID)
				mtx.Lock()
				if err != nil {
					resCHs[ci] <- nil
				} else {
					resCHs[ci] <- res
				}
				mtx.Unlock()
			}(i)
		}

		wg.Wait()

		aggrRes := make([]NodeInfo, 0)
		for _, c := range resCHs {
			if r := <-c; r != nil {
				aggrRes = append(aggrRes, r...)
			}
		}

		kClose := a.GetKClosest(aggrRes)
		closer := false
		for _, newNode := range kClose {
			if closestDist.Cmp(GetDistance(nodeID, newNode.ID)) > 0 {
				closer = true
				break
			}
		}

		if closer {
			a.lookup(nodeID, kClose)
		} else {
			break BATCH
		}
	}
}

func (a *Amigo) GetInitialCloseNodes(id ID) []NodeInfo {
	idx := GetDistanceIndex(a.self, id)
	ret := make([]NodeInfo, 0, Alpha)
	for idx >= 0 {
		ret = append(ret, a.buckets[idx].GetN(Alpha-len(ret))...)
		idx--
	}
	return ret
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

func (a *Amigo) query(queryNode NodeInfo, queryID ID) ([]NodeInfo, error) {
	return nil, nil
}

// TODO: use a better sort algorithm
func (a *Amigo) GetKClosest(nodes []NodeInfo) []NodeInfo {
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
