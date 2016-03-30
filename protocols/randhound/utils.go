// Utility functions used in the RandHound protocol.
package randhound

import (
	"errors"
	"fmt"

	"github.com/dedis/cothority/lib/network"
	"github.com/dedis/crypto/abstract"
	"github.com/dedis/crypto/random"
)

// CreateShards produces a pseudorandom sharding of the network entity list
// based on a seed and a number of requested shards.
func (rh *RandHound) CreateSharding(seed []byte, shards uint32) ([][]*network.Entity, error) {

	if rh.Group.N < shards {
		return nil, errors.New(fmt.Sprintf("Number of requested shards larger than available number of nodes"))
	}

	// Compute a permutation of [0,n-1]
	prng := rh.Node.Suite().Cipher(seed)
	m := make([]uint32, rh.Group.N)
	for i := range m {
		j := int(random.Uint64(prng) % uint64(i+1))
		m[i] = m[j]
		m[j] = uint32(i)
	}

	// Create sharding of the current EntityList according to the above permutation
	el := rh.Node.EntityList().List
	n := int(rh.Group.N / shards)
	sharding := [][]*network.Entity{}
	shard := []*network.Entity{}
	for i, j := range m {
		shard = append(shard, el[j])
		if (i%n == n-1) || (i == len(m)-1) {
			sharding = append(sharding, shard)
			shard = make([]*network.Entity, 0)
		}
	}
	return sharding, nil
}

func (rh *RandHound) chooseTrustees(Rc, Rs []byte) (map[uint32]uint32, []abstract.Point) {

	// Seed PRNG for selection of trustees
	var seed []byte
	seed = append(seed, Rc...)
	seed = append(seed, Rs...)
	prng := rh.Node.Suite().Cipher(seed)

	// Choose trustees uniquely
	shareIdx := make(map[uint32]uint32)
	trustees := make([]abstract.Point, rh.Group.K)
	tns := rh.Tree().ListNodes()
	j := uint32(0)
	for uint32(len(shareIdx)) < rh.Group.K {
		i := uint32(random.Uint64(prng) % uint64(len(tns)))
		// Add trustee only if not done so before; choosing yourself as an trustee is fine; ignore leader at index 0
		if _, ok := shareIdx[i]; !ok && !tns[i].IsRoot() {
			shareIdx[i] = j // j is the share index
			trustees[j] = tns[i].Entity.Public
			j += 1
		}
	}
	return shareIdx, trustees
}

func (rh *RandHound) hash(bytes ...[]byte) []byte {
	return abstract.Sum(rh.Node.Suite(), bytes...)
}

func (rh *RandHound) nodeIdx() uint32 {
	return uint32(rh.Node.TreeNode().EntityIdx)
}

func (rh *RandHound) sendToChildren(msg interface{}) error {
	for _, c := range rh.Children() {
		if err := rh.SendTo(c, msg); err != nil {
			return err
		}
	}
	return nil
}

func (rh *RandHound) generateTranscript() {} // TODO
func (rh *RandHound) verifyTranscript()   {} // TODO
