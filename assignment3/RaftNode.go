package main

import (
	"errors"
	//"fmt"
	"github.com/cs733-iitb/cluster"
	"github.com/cs733-iitb/log"
	//"math/rand"
	"strconv"
	"time"
	//"bufio"
	//"io"
	//"math"
	//"net"
	//"strings"
)

// -------------------- configuration structures --------------------

type NetConfig struct {
	Id   int
	Host string
	Port int
}

type Config struct {
	Cluster          []NetConfig
	Id               int
	LogFileDir       string
	StateFileDir     string
	ElectionTimeout  int
	HeartbeatTimeout int
}

// -------------------- commit info structures --------------------

type CommitInfo struct {
	Data  []byte
	Index int
	Err   error
}

// -------------------- raft node structures --------------------

type Node interface {
	// Client's message to Raft node
	Append([]byte) error

	// A channel for client to listen on. What goes into Append must come out of here at some point.
	CommitChannel() (<-chan CommitInfo, error)

	// Last known committed index in the log. This could be -1 until the system stabilizes.
	CommittedIndex() (int, error)

	// Returns the data at a log index, or an error.
	Get(index int) ([]byte, error)

	// Node's id
	Id() (int, error)

	// Id of leader. -1 if unknown
	LeaderId() (int, error)

	// Signal to shut down all goroutines, stop sockets, flush log and close it, cancel timers.
	Shutdown()
}

type RaftNode struct {
	SM             StateMachine
	IsWorking      bool
	TimeoutTimer   *time.Timer
	AppendEventCh  chan Event
	TimeoutEventCh chan Event
	CommitCh       chan CommitInfo
	NetServer      cluster.Server
	LogFile        *log.Log
	StateFile      *log.Log
}

// -------------------- raft node functions --------------------

func (RN *RaftNode) Append(data []byte) error {
	if RN.IsWorking {
		append_ev := AppendEvent{Data: data}
		RN.AppendEventCh <- append_ev
		return nil
	}
	return errors.New("ERR_RN_NOT_WORKING")
}

func (RN *RaftNode) CommitChannel() (<-chan CommitInfo, error) {
	if RN.IsWorking {
		return RN.CommitCh, nil
	}
	return nil, errors.New("ERR_RN_NOT_WORKING")
}

func (RN *RaftNode) CommittedIndex() (int, error) {
	if RN.IsWorking {
		RN.SM.Mux.Lock()
		defer RN.SM.Mux.Unlock()
		return RN.SM.CommitIndex, nil
	}
	return -1, errors.New("ERR_RN_NOT_WORKING")
}

func (RN *RaftNode) Get(index int) ([]byte, error) {
	if RN.IsWorking {
		RN.SM.Mux.Lock()
		defer RN.SM.Mux.Unlock()
		if index <= len(RN.SM.Log)-1 {
			return RN.SM.Log[index].Data, nil
		} else {
			return nil, errors.New("ERR_NO_LOG_ENTRY_FOR_INDEX")
		}
	}
	return nil, errors.New("ERR_RN_NOT_WORKING")
}

func (RN *RaftNode) Id() (int, error) {
	if RN.IsWorking {
		return RN.SM.Id, nil
	}
	return -1, errors.New("ERR_RN_NOT_WORKING")
}

func (RN *RaftNode) LeaderId() (int, error) {
	if RN.IsWorking {
		RN.SM.Mux.Lock()
		defer RN.SM.Mux.Unlock()
		return RN.SM.LeaderId, nil
	}
	return -1, errors.New("ERR_RN_NOT_WORKING")
}

func (RN *RaftNode) Shutdown() {
	RN.SM.Mux.Lock()
	defer RN.SM.Mux.Unlock()

	RN.IsWorking = false
	RN.TimeoutTimer.Stop()
	//Close(RN.TimeoutEventCh)
	//Close(RN.AppendEventCh)
	//Close(RN.CommitCh)
	RN.NetServer.Close()
	RN.LogFile.Close()
	RN.StateFile.Close()
}

func makeRafts() []Node {
	var nodes []Node

	clust := []NetConfig{
		{Id: 101, Host: "127.0.0.1", Port: 8001},
		{Id: 102, Host: "127.0.0.1", Port: 8002},
		{Id: 103, Host: "127.0.0.1", Port: 8003},
		{Id: 104, Host: "127.0.0.1", Port: 8004},
		{Id: 105, Host: "127.0.0.1", Port: 8005}}
	file_dir := "file"

	for i := 0; i < 5; i++ {
		conf := Config{Cluster: clust, Id: clust[i].Id, LogFileDir: file_dir + "_log_" + strconv.Itoa(clust[i].Id), StateFileDir: file_dir + "_state_" + strconv.Itoa(clust[i].Id), ElectionTimeout: 800, HeartbeatTimeout: 100}
		raft_node := New(conf)
		nodes = append(nodes, raft_node)
	}

	return nodes
}

func getLeader(nodes []Node) (Node, error) {
	for _, rn := range nodes {
		id, _ := rn.Id()
		leader_id, _ := rn.LeaderId()
		if id == leader_id {
			return rn, nil
		}
	}
	return nil, errors.New("ERR_NO_LEADER_FOUND")
}

func GetClusterConfig(conf Config) cluster.Config {
	var peer_config []cluster.PeerConfig

	for _, peer := range conf.Cluster {
		peer_config = append(peer_config, cluster.PeerConfig{Id: peer.Id, Address: peer.Host + ":" + strconv.Itoa(peer.Port)})
	}

	return cluster.Config{Peers: peer_config, InboxSize: 1000, OutboxSize: 1000}
}

func New(conf Config) Node {
	var rn RaftNode

	// initlisation of other raft node variables
	rn.IsWorking = true
	rn.TimeoutTimer = time.NewTimer(0)
	<-rn.TimeoutTimer.C
	rn.AppendEventCh = make(chan Event)
	rn.TimeoutEventCh = make(chan Event)
	rn.CommitCh = make(chan CommitInfo)
	ClustConfig := GetClusterConfig(conf)               // ?????
	rn.NetServer, _ = cluster.New(conf.Id, ClustConfig) // ?????
	rn.LogFile, _ = log.Open(conf.LogFileDir)
	rn.StateFile, _ = log.Open(conf.StateFileDir)

	// initilisation of state machine
	rn.SM.Id = conf.Id
	rn.SM.Conf = conf
	rn.SM.State = "Follower"
	rn.SM.NumOfVotes = 0
	rn.SM.NumOfNegVotes = 0
	rn.SM.LeaderId = -1
	// if rn.StateFile.GetLastIndex() != -1 {
	// 	entry, err := rn.StateFile.Get(0)
	// 	state_entry := entry.(StateEntry)

	// 	rn.SM.CurrentTerm = state_entry.Term
	// 	rn.SM.VotedFor = state_entry.VotedFor
	// 	rn.SM.CurrTermLastLogIndex = state_entry.CurrTermLastLogIndex
	// } else {
	rn.SM.CurrentTerm = 0
	rn.SM.VotedFor = -1
	rn.SM.CurrTermLastLogIndex = -1
	rn.StateFile.Append(StateEntry{Term: 0, VotedFor: -1, CurrTermLastLogIndex: -1})
	//}
	// if rn.StateFile.GetLastIndex() != -1 {
	// 	last_index := rn.StateFile.GetLastIndex()
	// 	for i := 0; i <= last_index; i++ {
	// 		entry, err := rn.LogFile.Get(i)
	// 		log_entry := entry.(LogEntry)
	// 		rn.SM.Log = append(rn.SM.Log, log_entry)
	// 	}
	// } else {
	rn.SM.Log = append(rn.SM.Log, LogEntry{Term: 0, Data: nil})
	rn.LogFile.Append(LogEntry{Term: 0, Data: nil})
	//}
	rn.SM.CommitIndex = 0
	rn.SM.LastApplied = 0
	for _, _ = range conf.Cluster {
		rn.SM.NextIndex = append(rn.SM.NextIndex, len(rn.SM.Log))
		rn.SM.MatchIndex = append(rn.SM.MatchIndex, 0)
	}

	go rn.ProcessTimers()
	go rn.ProcessNodeEvents()

	return &rn
}

func (RN *RaftNode) ProcessTimers() {
	for RN.IsWorking {
		<-RN.TimeoutTimer.C
		RN.TimeoutEventCh <- TimeoutEvent{}
	}
}

func (RN *RaftNode) ProcessActions(actions []Action) {
	for i, act := range actions {
		switch act.(type) {

		case SendAction:
			send_act := act.(SendAction)
			// send_act.ev to outbox of send_act.peerid
			RN.NetServer.Outbox() <- &cluster.Envelope{Pid: send_act.PeerId, Msg: send_act.Ev}

		case CommitAction:
			commit_act := act.(CommitAction)
			RN.CommitCh <- CommitInfo{Data: commit_act.Data, Index: commit_act.Index, Err: commit_act.Err}

		case AlarmAction:
			alarm_act := act.(AlarmAction)
			RN.TimeoutTimer.Reset(time.Millisecond * time.Duration(alarm_act.Time))

		case LogStoreAction:
			log_store_act := act.(LogStoreAction)
			RN.LogFile.TruncateToEnd(int64(log_store_act.Index))
			RN.LogFile.Append(LogEntry{Term: log_store_act.Term, Data: log_store_act.Data})

		case StateStoreAction:
			state_store_act := act.(StateStoreAction)
			RN.StateFile.TruncateToEnd(0)
			RN.StateFile.Append(StateEntry{Term: state_store_act.Term, VotedFor: state_store_act.VotedFor, CurrTermLastLogIndex: state_store_act.CurrTermLastLogIndex})
		}
	}
}

func (RN *RaftNode) ProcessNodeEvents() {
	for RN.IsWorking {
		var ev Event

		select {
		case ev = <-RN.AppendEventCh:
			// Append Event

		case ev = <-RN.TimeoutEventCh:
			// Timeout Event

		case inboxEvent := <-RN.NetServer.Inbox():
			// Network Event
			ev = inboxEvent.Msg
		}

		actions := RN.SM.ProcessEvent(ev)
		RN.ProcessActions(actions)
	}
}
