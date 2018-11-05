package dht

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/spacemeshos/go-spacemesh/p2p/config"
	"github.com/spacemeshos/go-spacemesh/p2p/node"
	"github.com/spacemeshos/go-spacemesh/p2p/service"
	"github.com/spacemeshos/go-spacemesh/p2p/simulator"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	ln, _ := node.GenerateTestNode(t)

	cfg := config.DefaultConfig()
	sim := simulator.New()

	n1 := sim.NewNodeFrom(ln.Node)

	d := New(ln, cfg.SwarmConfig, n1)
	assert.NotNil(t, d, "D is not nil")
}

func TestKadDHT_EveryNodeIsInRoutingTable(t *testing.T) {

	numPeers, connections := 100, 5

	bncfg := config.DefaultConfig()
	sim := simulator.New()

	bn, _ := node.GenerateTestNode(t)
	b1 := sim.NewNodeFrom(bn.Node)
	bdht := New(bn, bncfg.SwarmConfig, b1)
	b1.AttachDHT(bdht)

	cfg := config.DefaultConfig().SwarmConfig
	cfg.Gossip = false
	cfg.Bootstrap = true
	cfg.RandomConnections = connections
	cfg.BootstrapNodes = append(cfg.BootstrapNodes, node.StringFromNode(bn.Node))

	viper.Unmarshal(cfg)

	type all struct {
		err  error
		dht  *KadDHT
		ln   *node.LocalNode
		node service.Service
	}

	booted := make(chan all, numPeers)

	bootWaitRetrnAll := func(t *testing.T, ln *node.LocalNode, serv *simulator.Node, dht *KadDHT, c chan all) {
		err := dht.Bootstrap()
		c <- all{
			err,
			dht,
			ln,
			serv,
		}
	}

	idstoFind := make([]string, 0, numPeers)
	dhtsToLook := make([]*KadDHT, 0, numPeers)
	//selectedIds := make(map[string][]node.Node, numPeers)

	for n := 0; n < numPeers; n++ {
		ln, _ := node.GenerateTestNode(t)
		n := sim.NewNodeFrom(ln.Node)
		dht := New(ln, cfg, n)
		n.AttachDHT(dht)
		go bootWaitRetrnAll(t, ln, n, dht, booted)
	}

	i := 0
	for {
		if i == numPeers {
			break
		}
		b := <-booted
		if !assert.NoError(t, b.err) {
			t.FailNow()
		}
		idstoFind = append(idstoFind, b.ln.String())
		dhtsToLook = append(dhtsToLook, b.dht)

		fmt.Printf("Node %v finished bootstrap with rt of %v \r\n", b.ln.String(), b.dht.Size())
		i++
	}

	assert.Equal(t, len(idstoFind), numPeers)
	assert.Equal(t, len(dhtsToLook), numPeers)
	//assert.Equal(t, len(selectedIds), numPeers)

	var passed = make([]string, 0, numPeers)
NL:
	for n := range idstoFind { // iterate nodes
		id := idstoFind[n]
		for j := range dhtsToLook { // iterate all selected set
			if n == j {
				continue
			}

			cb := make(PeerOpChannel)
			dhtsToLook[j].rt.NearestPeer(PeerByIDRequest{node.NewDhtIDFromBase58(id), cb})
			c := <-cb

			if c.Peer != node.EmptyNode && c.Peer.String() == id {
				passed = append(passed, id)
				continue NL
			}

		}
	}

	assert.Equal(t, len(passed), numPeers)
	spew.Dump(passed)
}

func TestSwarm_EveryNodeIsInSelected(t *testing.T) {

	for i := 0; i < 1; i++ {
		t.Run(fmt.Sprintf("t%v", i), func(t *testing.T) {
			numPeers, connections := 500, 8

			bncfg := config.DefaultConfig()
			sim := simulator.New()

			bn, _ := node.GenerateTestNode(t)
			b1 := sim.NewNodeFrom(bn.Node)
			bdht := New(bn, bncfg.SwarmConfig, b1)
			b1.AttachDHT(bdht)

			cfg := config.DefaultConfig().SwarmConfig
			cfg.Gossip = false
			cfg.Bootstrap = true
			cfg.RandomConnections = connections
			cfg.BootstrapNodes = append(cfg.BootstrapNodes, node.StringFromNode(bn.Node))

			type all struct {
				err  error
				dht  DHT
				ln   *node.LocalNode
				node service.Service
			}

			booted := make(chan all, numPeers)

			idstoFind := make([]string, 0, numPeers)
			selectedIds := make(map[string][]node.Node, numPeers)

			for n := 0; n < numPeers; n++ {
				go func() {
					ln, _ := node.GenerateTestNode(t)
					n := sim.NewNodeFrom(ln.Node)
					dht := New(ln, cfg, n)
					n.AttachDHT(dht)
					err := dht.Bootstrap()

					booted <- all{
						err,
						dht,
						ln,
						n,
					}
				}()
			}

			i := 0
			for {
				if i == numPeers {
					break
				}
				b := <-booted
				if !assert.NoError(t, b.err) {
					t.FailNow()
				}
				idstoFind = append(idstoFind, b.ln.String())
				selectedIds[b.ln.String()] = b.dht.SelectPeers(connections)

				fmt.Printf("Node %v finished bootstrap with rt of %v \r\n", b.ln.String(), b.dht.Size())
				i++
			}

			assert.Equal(t, len(idstoFind), numPeers)
			assert.Equal(t, len(selectedIds), numPeers)

			// check everyone is selected
			var passed = make([]string, 0, numPeers)
		NL:
			for n := range idstoFind { // iterate nodes
				id := idstoFind[n]
				for j := range selectedIds { // iterate all selected set
					if id == j {
						continue
					}

					for s := range selectedIds[j] {
						if selectedIds[j][s].String() == id {
							passed = append(passed, id)
							continue NL
						}
					}
				}
			}

			// we got enough selections and we found everyone.
			assert.Equal(t, len(passed), numPeers)
		})
	}

}

func TestDHT_Update(t *testing.T) {
	ln, _ := node.GenerateTestNode(t)

	cfg := config.DefaultConfig()
	sim := simulator.New()

	n1 := sim.NewNodeFrom(ln.Node)

	dht := New(ln, cfg.SwarmConfig, n1)

	randnode := node.GenerateRandomNodeData()
	dht.Update(randnode)

	req := make(chan int)
	dht.rt.Size(req)
	size := <-req

	assert.Equal(t, 1, size, "Routing table filled")

	morenodes := node.GenerateRandomNodesData(config.DefaultConfig().SwarmConfig.RoutingTableBucketSize - 2) // more than bucketsize might result is some nodes not getting in

	for i := range morenodes {
		dht.Update(morenodes[i])
	}

	dht.rt.Size(req)
	size = <-req

	assert.Equal(t, config.DefaultConfig().SwarmConfig.RoutingTableBucketSize-1, size)

	evenmorenodes := node.GenerateRandomNodesData(30) // more than bucketsize might result is some nodes not getting in

	for i := range evenmorenodes {
		dht.Update(evenmorenodes[i])
	}

	dht.rt.Size(req)
	size = <-req

	assert.True(t, size > config.DefaultConfig().SwarmConfig.RoutingTableBucketSize, "Routing table should be at least as big as bucket size")

	lastnode := evenmorenodes[0]

	looked, err := dht.Lookup(lastnode.PublicKey().String())

	assert.NoError(t, err, "error finding existing node ")

	assert.Equal(t, looked.String(), lastnode.String(), "didnt find the same node")
	assert.Equal(t, looked.Address(), lastnode.Address(), "didnt find the same node")

}

func TestDHT_Lookup(t *testing.T) {
	ln, _ := node.GenerateTestNode(t)

	cfg := config.DefaultConfig()
	sim := simulator.New()

	n1 := sim.NewNodeFrom(ln.Node)

	dht := New(ln, cfg.SwarmConfig, n1)

	randnode := node.GenerateRandomNodeData()

	dht.Update(randnode)

	node, err := dht.Lookup(randnode.PublicKey().String())

	assert.NoError(t, err, "Should not return an error")

	assert.True(t, node.String() == randnode.String(), "should return the same node")
}

func TestDHT_Lookup2(t *testing.T) {
	ln, _ := node.GenerateTestNode(t)

	cfg := config.DefaultConfig()
	sim := simulator.New()

	n1 := sim.NewNodeFrom(ln.Node)

	dht := New(ln, cfg.SwarmConfig, n1)

	randnode := node.GenerateRandomNodeData()

	dht.Update(randnode)

	ln2, _ := node.GenerateTestNode(t)

	n2 := sim.NewNodeFrom(ln2.Node)

	dht2 := New(ln2, cfg.SwarmConfig, n2)

	dht2.Update(dht.local.Node)

	node, err := dht2.Lookup(randnode.PublicKey().String())
	assert.NoError(t, err, "error finding node ", err)

	assert.Equal(t, node.String(), randnode.String(), "not the same node")

}

func simNodeWithDHT(t *testing.T, sc config.SwarmConfig, sim *simulator.Simulator) (*simulator.Node, DHT) {
	ln, _ := node.GenerateTestNode(t)
	n := sim.NewNodeFrom(ln.Node)
	dht := New(ln, sc, n)
	n.AttachDHT(dht)

	return n, dht
}

func bootAndWait(t *testing.T, dht DHT, errchan chan error) {
	err := dht.Bootstrap()
	errchan <- err
}

func TestDHT_Bootstrap(t *testing.T) {
	// Create a bootstrap node
	sim := simulator.New()
	bn, _ := simNodeWithDHT(t, config.DefaultConfig().SwarmConfig, sim)

	// config for other nodes
	cfg2 := config.DefaultConfig()
	cfg2.SwarmConfig.RandomConnections = 2 // min numbers of peers to succeed in bootstrap
	cfg2.SwarmConfig.BootstrapNodes = []string{node.StringFromNode(bn.Node)}

	booted := make(chan error)

	// boot 3 more dhts

	_, dht2 := simNodeWithDHT(t, cfg2.SwarmConfig, sim)
	_, dht3 := simNodeWithDHT(t, cfg2.SwarmConfig, sim)
	_, dht4 := simNodeWithDHT(t, cfg2.SwarmConfig, sim)

	go bootAndWait(t, dht2, booted)
	go bootAndWait(t, dht3, booted)
	go bootAndWait(t, dht4, booted)

	// Collect errors
	err := <-booted
	assert.NoError(t, err, "should be able to bootstrap a node")
	err = <-booted
	assert.NoError(t, err, "should be able to bootstrap another node")
	err = <-booted
	assert.NoError(t, err, "should be able to bootstrap another node")
}

// A bigger bootstrap
func TestDHT_Bootstrap2(t *testing.T) {

	const timeout = 10 * time.Second
	const nodesNum = 100
	const minToBoot = 25

	sim := simulator.New()

	// Create a bootstrap node
	cfg := config.DefaultConfig()
	bn, _ := simNodeWithDHT(t, cfg.SwarmConfig, sim)

	// config for other nodes
	cfg2 := config.DefaultConfig()
	cfg2.SwarmConfig.RandomConnections = minToBoot // min numbers of peers to succeed in bootstrap
	cfg2.SwarmConfig.BootstrapNodes = []string{node.StringFromNode(bn.Node)}

	booted := make(chan error)

	for i := 0; i < nodesNum; i++ {
		_, d := simNodeWithDHT(t, cfg2.SwarmConfig, sim)
		go bootAndWait(t, d, booted)
	}

	timer := time.NewTimer(timeout)

	i := 0
	for i < nodesNum-1 {
		select {
		case e := <-booted:
			if e != nil {
				t.Error("Failed to boot a node. ", e)
			}
			i++
		case <-timer.C:
			t.Error("Failed to boot within time")
		}
	}
}

func Test_filterFindNodeServers(t *testing.T) {
	//func filterFindNodeServers(nodes []node.Node, queried map[string]struct{}, alpha int) []node.Node {

	nodes := node.GenerateRandomNodesData(20)

	q := make(map[string]struct{})
	q[nodes[0].String()] = struct{}{}
	q[nodes[1].String()] = struct{}{}
	q[nodes[2].String()] = struct{}{}

	filtered := filterFindNodeServers(nodes, q, 5)

	assert.Equal(t, 5, len(filtered))

	for n := range filtered {
		if _, ok := q[filtered[n].String()]; ok {
			t.Error("It was in the filtered")
		}
	}

}
