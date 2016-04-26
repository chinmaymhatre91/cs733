package main

import (
	"fmt"
	"github.com/cs733-iitb/cluster"
	"github.com/cs733-iitb/cluster/mock"
	"strconv"
	"testing"
	"time"
	"errors"
)

var my_clust = []NetConfig{
	{Id: 101, Host: "127.0.0.1", Port: 9001},
	{Id: 102, Host: "127.0.0.1", Port: 9002},
	{Id: 103, Host: "127.0.0.1", Port: 9003},
	{Id: 104, Host: "127.0.0.1", Port: 9004},
	{Id: 105, Host: "127.0.0.1", Port: 9005}}


func makeRaft(clust []NetConfig, index int) Node {
	file_dir := "node"
	conf := Config{Cluster: clust, Id: clust[index].Id, LogFileDir: file_dir + strconv.Itoa(clust[index].Id) + "/log", StateFileDir: file_dir + strconv.Itoa(clust[index].Id) + "/state", ElectionTimeout: 800, HeartbeatTimeout: 400}
	raft_node := New(conf)
	return raft_node
}


func makeRafts(clust []NetConfig) []Node {
	var nodes []Node

	for i, _ := range clust {
		raft_node := makeRaft(clust, i)
		nodes = append(nodes, raft_node)
	}

	return nodes
}

func GetMockClusterConfig(clust []NetConfig) cluster.Config {
	var peer_config []cluster.PeerConfig

	for _, peer := range clust {
		peer_config = append(peer_config, cluster.PeerConfig{Id: peer.Id, Address: peer.Host + ":" + strconv.Itoa(peer.Port)})
	}

	return cluster.Config{Peers: peer_config}
}

func changeToMockCluster(nodes []Node, mockCluster *mock.MockCluster)  {
	for _, node := range nodes {
		node_id, _ := node.Id()
		node.ChangeToMockServer(mockCluster.Servers[node_id])
	}
}

func getLeader(nodes []Node) (Node, error) {
	var id, leader_id int
	var err error
	for _, rn := range nodes {
		id, err = rn.Id()
		if err != nil {
			continue
		}
		leader_id, err = rn.LeaderId()
		if err != nil {
			continue
		}
		if id == leader_id {
			return rn, nil
		}
	}
	return nil, errors.New("ERR_NO_LEADER_FOUND")
}


///*
func TestBasic(t *testing.T) {
	rafts := makeRafts(my_clust) // array of []raft.Node
	time.Sleep(time.Second * 2)
	
	ldr, err := getLeader(rafts)
	for err != nil {
		ldr, err = getLeader(rafts)
	}
	ldr_id, _ := ldr.Id()
	if debug_l3 {
		fmt.Println("--------------------- Leader ---------------------> " + strconv.Itoa(ldr_id))
	}
	ldr.Append([]byte("foo"))
	time.Sleep(time.Second * 2)

	if debug_l3 {
		for _, node := range rafts {
			rn_node := node.(*RaftNode)
			fmt.Println(rn_node.SM.Log, strconv.Itoa(rn_node.SM.CommitIndex))
			//fmt.Println(strconv.Itoa(rn_node.SM.Id) + ":" + string(rn_node.SM.Log[0].Data))
			//fmt.Println(strconv.Itoa(rn_node.SM.Id) + ":" + string(rn_node.SM.Log[1].Data))
		}
	}

	for _, node := range rafts {
		ch, _ := node.CommitChannel()
		rn_node := node.(*RaftNode)
		select {
		case ci := <-ch:
			if ci.Err != nil {
				t.Fatal(ci.Err)
			}
			if debug_l3 {
				fmt.Println(strconv.Itoa(rn_node.SM.Id) + "->" + string(ci.Data) + "->" + strconv.Itoa(rn_node.SM.CommitIndex))
			}
			//if ci.Data != nil {
			if string(ci.Data) != "foo" {
				t.Fatal("Got different data")
			}
		default:
			t.Fatal("Expected message on all nodes")
		}
	}

	for _, node := range rafts {
		node.Shutdown()
	}

	if debug_l3 {
		fmt.Println("---------------------------------------------------------------------------------------")
	}
}
//*/

///*
func TestMultipleAppend(t *testing.T) {
	var strs []string 
	strs = append(strs, "foo1")
	strs = append(strs, "foo2") 


	rafts := makeRafts(my_clust) // array of []raft.Node
	time.Sleep(time.Second * 2)
	
	ldr, err := getLeader(rafts)
	for err != nil {
		ldr, err = getLeader(rafts)
	}
	ldr_id, _ := ldr.Id()
	if debug_l3 {
		fmt.Println("--------------------- Leader ---------------------> " + strconv.Itoa(ldr_id))
	}
	ldr.Append([]byte(strs[0]))
	time.Sleep(time.Second * 4)

	ldr, err = getLeader(rafts)
	for err != nil {
		ldr, err = getLeader(rafts)
	}
	ldr_id, _ = ldr.Id()
	if debug_l3 {
		fmt.Println("--------------------- Leader ---------------------> " + strconv.Itoa(ldr_id))
	}
	ldr.Append([]byte(strs[1]))
	time.Sleep(time.Second * 4)

	if debug_l3 {
		for _, node := range rafts {
			rn_node := node.(*RaftNode)
			fmt.Println(rn_node.SM.Log, strconv.Itoa(rn_node.SM.CommitIndex))
			//fmt.Println(strconv.Itoa(rn_node.SM.Id) + ":" + string(rn_node.SM.Log[0].Data))
			//fmt.Println(strconv.Itoa(rn_node.SM.Id) + ":" + string(rn_node.SM.Log[1].Data))
		}
	}

	for _, str := range strs {
		for _, node := range rafts {
			ch, _ := node.CommitChannel()
			rn_node := node.(*RaftNode)
			select {
			case ci := <-ch:
				if ci.Err != nil {
					t.Fatal(ci.Err)
				}
				if debug_l3 {
					fmt.Println(strconv.Itoa(rn_node.SM.Id) + "->" + string(ci.Data) + "->" + strconv.Itoa(rn_node.SM.CommitIndex))
				}
				//if ci.Data != nil {
				if string(ci.Data) != str {
					t.Fatal("Got different data")
				}
			default:
				t.Fatal("Expected message on all nodes")
			}
		}
	}

	for _, node := range rafts {
		node.Shutdown()
	}

	if debug_l3 {
		fmt.Println("---------------------------------------------------------------------------------------")
	}
}
//*/


///*
func TestLeaderFail(t *testing.T) {
	rafts := makeRafts(my_clust) // array of []raft.Node
	time.Sleep(time.Second * 2)
	
	old_ldr, err := getLeader(rafts)
	for err != nil {
		old_ldr, err = getLeader(rafts)
	}
	old_ldr_id, _ := old_ldr.Id()
	if debug_l3 {
		fmt.Println("--------------------- Leader ---------------------> " + strconv.Itoa(old_ldr_id))
	}

	old_ldr.Shutdown()
	time.Sleep(time.Second * 2)

	ldr, err := getLeader(rafts)
	for err != nil {
		ldr, err = getLeader(rafts)
	}
	ldr_id, _ := ldr.Id()
	if debug_l3 {
		fmt.Println("--------------------- Leader ---------------------> " + strconv.Itoa(ldr_id))
	}

	ldr.Append([]byte("foo"))
	time.Sleep(time.Second * 4)

	if debug_l3 {
		for _, node := range rafts {
			rn_node := node.(*RaftNode)
			fmt.Println(rn_node.SM.Log, strconv.Itoa(rn_node.SM.CommitIndex))
			//fmt.Println(strconv.Itoa(rn_node.SM.Id) + ":" + string(rn_node.SM.Log[0].Data))
			//fmt.Println(strconv.Itoa(rn_node.SM.Id) + ":" + string(rn_node.SM.Log[1].Data))
		}
	}

	for _, node := range rafts {
		ch, err := node.CommitChannel()
		if err != nil {
			continue
		}
		rn_node := node.(*RaftNode)
		select {
		case ci := <-ch:
			if ci.Err != nil {
				t.Fatal(ci.Err)
			}
			if debug_l3 {
				fmt.Println(strconv.Itoa(rn_node.SM.Id) + "->" + string(ci.Data) + "->" + strconv.Itoa(rn_node.SM.CommitIndex))
			}
			//if ci.Data != nil {
			if string(ci.Data) != "foo" {
				t.Fatal("Got different data")
			}
		default:
			t.Fatal("Expected message on all nodes")
		}
	}

	for _, node := range rafts {
		node.Shutdown()
	}

	if debug_l3 {
		fmt.Println("---------------------------------------------------------------------------------------")
	}
}
//*/

///*
func TestPartition(t *testing.T) {
	rafts := makeRafts(my_clust) // array of []raft.Node
	time.Sleep(time.Second)
	MockClustConfig := GetMockClusterConfig(my_clust)
	mockCluster, _ := mock.NewCluster(MockClustConfig)
	changeToMockCluster(rafts, mockCluster)
	time.Sleep(time.Second * 2)

	var part1 []Node
	var part2 []Node
	var part1_Ids []int
	var part2_Ids []int

	ldr, err := getLeader(rafts)
	for err != nil {
		ldr, err = getLeader(rafts)
	}
	for _, node := range rafts {
		node_id, _ := node.Id()
		node_leader, _ := node.LeaderId()
		if node_id == node_leader {
			part1 = append(part1, node)
			part1_Ids = append(part1_Ids, node_id)
		} else {
			part2 = append(part2, node)
			part2_Ids = append(part2_Ids, node_id)	
		}	 
	}

	mockCluster.Partition(part1_Ids, part2_Ids)
	ldr = part1[0]
	ldr_id, _ := ldr.Id()
	if debug_l3 {
		fmt.Println("--------------------- Leader ---------------------> " + strconv.Itoa(ldr_id))
 	}
 	ldr.Append([]byte("foo1"))

	time.Sleep(time.Second * 2)
	mockCluster.Heal()
	time.Sleep(time.Second * 2)

	ldr, err = getLeader(rafts)
	for err != nil {
		ldr, err = getLeader(rafts)
	}
	ldr_id, _ = ldr.Id()
	if debug_l3 {
		fmt.Println("--------------------- Leader ---------------------> " + strconv.Itoa(ldr_id))
	}
	ldr.Append([]byte("foo2"))
	time.Sleep(time.Second * 4)

	if debug_l3 {
		for _, node := range rafts {
			rn_node := node.(*RaftNode)
			fmt.Println(rn_node.SM.Log, strconv.Itoa(rn_node.SM.CommitIndex))
			//fmt.Println(strconv.Itoa(rn_node.SM.Id) + ":" + string(rn_node.SM.Log[0].Data))
			//fmt.Println(strconv.Itoa(rn_node.SM.Id) + ":" + string(rn_node.SM.Log[1].Data))
		}
	}

	for _, node := range rafts {
		ch, _ := node.CommitChannel()
		rn_node := node.(*RaftNode)
		select {
		case ci := <-ch:
			if ci.Err != nil {
				t.Fatal(ci.Err)
			}
			if debug_l3 {
				fmt.Println(strconv.Itoa(rn_node.SM.Id) + "->" + string(ci.Data))
			}
			//if ci.Data != nil {
			if string(ci.Data) != "foo2" {
				t.Fatal("Got different data")
			}
		default:
			t.Fatal("Expected message on all nodes")
		}
	}

	for _, node := range rafts {
		node.Shutdown()
	}

	if debug_l3 {
		fmt.Println("---------------------------------------------------------------------------------------")
	}
}
//*/
