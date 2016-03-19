package main

import (
	"fmt"
	"github.com/cs733-iitb/cluster"
	"github.com/cs733-iitb/log"
	"math/rand"
	"time"
	//"bufio"
	//"io"
	//"math"
	//"net"
	//"strconv"
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
	Append([]byte)

	// A channel for client to listen on. What goes into Append must come out of here at some point.
	CommitChannel() <-chan CommitInfo

	// Last known committed index in the log. This could be -1 until the system stabilizes.
	CommittedIndex() int

	// Returns the data at a log index, or an error.
	Get(index int) (Err, []byte)

	// Node's id
	Id()

	// Id of leader. -1 if unknown
	LeaderId() int

	// Signal to shut down all goroutines, stop sockets, flush log and close it, cancel timers.
	Shutdown()
}

type RaftNode struct {
	SM             StateMachine
	TimeoutTimer   *time.Timer
	AppendEventCh  chan Event
	TimeoutEventCh chan Event
	CommitCh       chan CommitInfo
	NetServer      cluster.Server
	LogFile        *log.Log
	StateFile      *log.Log
}

// -------------------- raft node functions --------------------

func (RN *RaftNode) Append(data []byte) {
	append_ev := AppendEvent{Data: data}
	RN.AppendEventCh <- append_ev
}

func (RN *RaftNode) CommitChannel() <-chan CommitInfo {
	return RN.CommitCh
}

func (RN *RaftNode) CommittedIndex() int {
	RN.SM.Mux.Lock()
	defer RN.SM.Mux.Unlock()
	return RN.SM.CommitIndex
}

func (RN *RaftNode) Get(index int) (Err, []byte) {

}

func (RN *RaftNode) Id() {
	return RN.SM.Id
}

func (RN *RaftNode) LeaderId() int {
	RN.SM.Mux.Lock()
	defer RN.SM.Mux.Unlock()
	return RN.SM.LeaderId
}

func (RN *RaftNode) Shutdown() {

	// stop timer goroutine ??????
	RN.TimeoutTimer.Stop()
	// stop processNodeEvents goroutine ????
}

func makeRafts() []Node {
	var nodes []Node

	return nodes
}

func getLeader(nodes []Node) Node {

}

func (RN *RaftNode) New(conf Config) Node {
	// initlisation of other raft node variables
	RN.TimeoutTimer = time.NewTimer(0)
	<-RN.TimeoutTimer.C

	AppendEventCh = make(chan Event)
	TimeoutEventCh = make(chan Event)
	CommitCh = make(chan CommitInfo)

	ClustConfig := getClusterConfig(conf) // ?????
	//RN.NetServer = cluster.New(, ClustConfig) // ?????

	RN.LogFile = log.Open(conf.LogFileDir)
	RN.StateFile = log.Open(conf.StateFileDir)

	// initilisation of state machine
	RN.SM.Id = conf.Id
	RN.SM.Conf = conf
	RN.SM.State = "Follower"
	RN.SM.NumOfVotes = 0
	RN.SM.NumOfNegVotes = 0
	RN.SM.LeaderId = -1
	if RN.StateFile.GetLastIndex() != -1 {
		entry, err := RN.StateFile.Get(0)
		state_entry := entry.(StateEntry)

		RN.SM.CurrentTerm = state_entry.Term
		RN.SM.VotedFor = state_entry.VotedFor
		RN.SM.CurrTermLastLogIndex = state_entry.CurrTermLastLogIndex
	} else {
		RN.SM.CurrentTerm = 0
		RN.SM.VotedFor = -1
		RN.SM.CurrTermLastLogIndex = -1
		RN.StateFile.Append(StateEntry{Term: 0, VotedFor: -1, CurrTermLastLogIndex: -1})
	}
	if RN.StateFile.GetLastIndex() != -1 {
		last_index := RN.StateFile.GetLastIndex()
		for i := 0; i <= last_index; i++ {
			entry, err := RN.LogFile.Get(i)
			log_entry := entry.(LogEntry)
			RN.SM.Log = Append(RN.SM.Log, log_entry)
		}
	} else {
		RN.SM.Log = Append(RN.SM.Log, LogEntry{Term: 0, Data: nil})
		RN.LogFile.Append(LogEntry{Term: 0, Data: nil})
	}
	RN.SM.CommitIndex = 0
	RN.SM.LastApplied = 0
	for _, _ = range conf.Cluster {
		sm.nextIndex = append(sm.nextIndex, len(sm.log))
		sm.matchIndex = append(sm.matchIndex, 0)
	}

	go RN.ProcessTimers()
	go RN.ProcessNodeEvents()
}

func (RN *RaftNode) ProcessTimers() {
	for {
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
			RN.TimeoutTimer.Reset(alarm_act.Time * time.Millisecond)

		case LogStoreAction:
			log_store_act := act.(LogStoreAction)
			RN.LogFile.TruncateToEnd(log_store_act.Index)
			RN.LogFile.Append(LogEntry{Term: log_store_act.Term, Data: log_store_act.Data})

		case StateStoreAction:
			state_store_act := act.(StateStoreAction)
			RN.StateFile.TruncateToEnd(0)
			RN.StateFile.Append(StateEntry{Term: state_store_act.Term, VotedFor: state_store_act.VotedFor, CurrTermLastLogIndex: state_store_act.CurrTermLastLogIndex})
		}
	}
}

func (RN *RaftNode) processNodeEvents() {
	for {
		var ev Event

		select {
		case ev <- RN.AppendEventCh:
			// Append Event

		case ev <- RN.TimeoutCh:
			// Timeout Event

		case inboxEvent := <-RN.NetServer.Inbox():
			// Network Event
			ev = inboxEvent.Msg
		}

		actions := RN.SM.processEvent(ev)
		RN.processActions(actions)
	}
}
