package sda

import (
	"errors"
	"github.com/dedis/cothority/lib/dbg"
	"github.com/dedis/cothority/lib/network"
	"github.com/dedis/crypto/abstract"
	"github.com/dedis/crypto/config"
	"github.com/satori/go.uuid"
	"strconv"
	"testing"
)

type LocalTest struct {
	Hosts       map[uuid.UUID]*Host
	Overlays    map[uuid.UUID]*Overlay
	EntityLists map[uuid.UUID]*EntityList
	Trees       map[uuid.UUID]*Tree
}

// NewLocalTest creates a new Local handler that can be used to test protocols
// locally
func NewLocalTest() *LocalTest {
	dbg.TestOutput(testing.Verbose(), 4)
	return &LocalTest{
		Hosts:       make(map[uuid.UUID]*Host),
		Overlays:    make(map[uuid.UUID]*Overlay),
		EntityLists: make(map[uuid.UUID]*EntityList),
		Trees:       make(map[uuid.UUID]*Tree),
	}
}

// StartNewNodeName takes a name and a tree and will create a
// new Node with the protocol 'name' running from the tree-root
func (l *LocalTest) StartNewNodeName(name string, t *Tree) (*Node, error) {
	rootEntityId := t.Root.Entity.Id
	for _, h := range l.Hosts {
		if uuid.Equal(h.Entity.Id, rootEntityId) {
			return l.Overlays[h.Entity.Id].StartNewNodeName(name, t)
		}
	}
	return nil, errors.New("Didn't find host for tree-root")
}

// GenTree will create a tree of n hosts. If connect is true, they will
// be connected to the root host. If register is true, the EntityList and Tree
// will be registered with the overlay.
func (l *LocalTest) GenTree(n int, connect bool, register bool) ([]*Host, *EntityList, *Tree) {
	hosts := GenLocalHosts(n, connect, true)
	for _, host := range hosts {
		l.Hosts[host.Entity.Id] = host
		l.Overlays[host.Entity.Id] = host.overlay
	}

	list := l.GenEntityListFromHost(hosts...)
	tree := list.GenerateBinaryTree()
	l.Trees[tree.Id] = tree
	if register {
		hosts[0].overlay.RegisterEntityList(list)
		hosts[0].overlay.RegisterTree(tree)
	}
	return hosts, list, tree
}

func (l *LocalTest) GenEntityListFromHost(hosts ...*Host) *EntityList {
	var entities []*network.Entity
	for i := range hosts {
		entities = append(entities, hosts[i].Entity)
	}
	list := NewEntityList(entities)
	l.EntityLists[list.Id] = list
	return list
}

// CloseAll takes a list of hosts that will be closed
func (l *LocalTest) CloseAll() {
	for _, host := range l.Hosts {
		err := host.Close()
		if err != nil {
			dbg.Error("Closing host", host, "gives error", err)
		}
	}
}

func (l *LocalTest) AddPendingTreeMarshal(h *Host, tm *TreeMarshal) {
	h.addPendingTreeMarshal(tm)
}

func (l *LocalTest) CheckPendingTreeMarshal(h *Host, el *EntityList) {
	h.checkPendingTreeMarshal(el)
}

// NewLocalHost creates a new host with the given address and registers it
func NewLocalHost(port int) *Host {
	address := "localhost:" + strconv.Itoa(port)
	priv, pub := PrivPub()
	id := network.NewEntity(pub, address)
	return NewHost(id, priv)
}

// GenLocalHosts will create n hosts with the first one being connected to each of
// the other nodes if connect is true
func GenLocalHosts(n int, connect bool, processMessages bool) []*Host {
	var hosts []*Host
	for i := 0; i < n; i++ {
		host := NewLocalHost(2000 + i*10)
		hosts = append(hosts, host)
	}
	root := hosts[0]
	for _, host := range hosts {
		host.Listen()
		if processMessages {
			go host.ProcessMessages()
		}
		if connect {
			if _, err := host.Connect(root.Entity); err != nil {
				dbg.Fatal("Could not connect hosts")
			}
		}
	}
	return hosts
}

// PrivPub creates a private/public key pair
func PrivPub() (abstract.Secret, abstract.Point) {
	keypair := config.NewKeyPair(network.Suite)
	return keypair.Secret, keypair.Public
}
