package main

import (
	//"fmt"
	"encoding/gob"
	"errors"
	"github.com/cs733-iitb/cluster"
	"github.com/cs733-iitb/cluster/mock"
	"github.com/cs733-iitb/log"
	"strconv"
	"time"
	"sync"
	//"math/rand"
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
	Data  	 []byte
	Index 	 int
	LeaderId int
	Err   	 error
}

// -------------------- raft node structures --------------------

type Node interface {
	Append([]byte) error
	CommitChannel() (<-chan CommitInfo, error)
	CommittedIndex() (int, error)
	Get(int) ([]byte, error)
	Id() (int, error)
	LeaderId() (int, error)
	ChangeToMockServer(*mock.MockServer)
	Shutdown()
}

type RaftNode struct {
	SM             StateMachine
	Mux            sync.Mutex
	IsWorking      bool
	TimeoutTimer   *time.Timer
	AppendEventCh  chan Event
	//TimeoutEventCh chan Event
	CommitCh       chan CommitInfo
	NetServer      cluster.Server
	LogFile        *log.Log
	StateFile      *log.Log
}

// -------------------- raft node functions --------------------

func (RN *RaftNode) Append(data []byte) error {
	RN.Mux.Lock()
	defer RN.Mux.Unlock()
	if RN.IsWorking {
		append_ev := AppendEvent{Data: data}
		RN.AppendEventCh <- append_ev
		return nil
	}
	return errors.New("ERR_RN_NOT_WORKING")
}

func (RN *RaftNode) CommitChannel() (<-chan CommitInfo, error) {
	RN.Mux.Lock()
	defer RN.Mux.Unlock()
	if RN.IsWorking {
		return RN.CommitCh, nil
	}
	return nil, errors.New("ERR_RN_NOT_WORKING")
}

func (RN *RaftNode) CommittedIndex() (int, error) {
	RN.Mux.Lock()
	defer RN.Mux.Unlock()
	if RN.IsWorking {
		return RN.SM.CommitIndex, nil
	}
	return -1, errors.New("ERR_RN_NOT_WORKING")
}

func (RN *RaftNode) Get(index int) ([]byte, error) {
	RN.Mux.Lock()
	defer RN.Mux.Unlock()
	if RN.IsWorking {
		if index <= len(RN.SM.Log)-1 {
			return RN.SM.Log[index].Data, nil
		} else {
			return nil, errors.New("ERR_NO_LOG_ENTRY_FOR_INDEX")
		}
	}
	return nil, errors.New("ERR_RN_NOT_WORKING")
}

func (RN *RaftNode) Id() (int, error) {
	RN.Mux.Lock()
	defer RN.Mux.Unlock()
	if RN.IsWorking {
		return RN.SM.Id, nil
	}
	return -1, errors.New("ERR_RN_NOT_WORKING")
}

func (RN *RaftNode) LeaderId() (int, error) {
	RN.Mux.Lock()
	defer RN.Mux.Unlock()
	if RN.IsWorking {
		return RN.SM.LeaderId, nil
	}
	return -1, errors.New("ERR_RN_NOT_WORKING")
}

func (RN *RaftNode) ChangeToMockServer(mockServer *mock.MockServer) {
	RN.Mux.Lock()
	defer RN.Mux.Unlock()
	if RN.IsWorking {
		RN.NetServer.Close()
		RN.NetServer = mockServer
	}
}

func (RN *RaftNode) Shutdown() {
	RN.Mux.Lock()
	defer RN.Mux.Unlock()
 
 	if RN.IsWorking {
		RN.IsWorking = false
		RN.TimeoutTimer.Stop()
		//Close(RN.TimeoutEventCh)
		//Close(RN.AppendEventCh)
		//Close(RN.CommitCh)
		RN.NetServer.Close()
		RN.LogFile.Close()
		RN.StateFile.Close()
	}
}

func RegisterStructs() {
	gob.Register(LogEntry{})
	gob.Register(StateEntry{})
	gob.Register(AppendEntriesReqEvent{})
	gob.Register(AppendEntriesRespEvent{})
	gob.Register(VoteReqEvent{})
	gob.Register(VoteRespEvent{})
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
	rn.AppendEventCh = make(chan Event, 100)
	//rn.TimeoutEventCh = make(chan Event, 100)
	rn.CommitCh = make(chan CommitInfo, 100)
	RegisterStructs()
	//fmt.Println("["+strconv.Itoa(conf.Id)+"] 2.2.1")
	ClustConfig := GetClusterConfig(conf)               // ?????
	rn.NetServer, _ = cluster.New(conf.Id, ClustConfig) // ?????
	rn.LogFile, _ = log.Open(conf.LogFileDir)
	//fmt.Println("["+strconv.Itoa(conf.Id)+"] 2.2.2")
	rn.StateFile, _ = log.Open(conf.StateFileDir)

	//fmt.Println("["+strconv.Itoa(conf.Id)+"] 2.2.2")
	// initilisation of state machine
	rn.SM.Id = conf.Id
	rn.SM.Conf = conf
	rn.SM.State = "Follower"
	rn.SM.NumOfVotes = 0
	rn.SM.NumOfNegVotes = 0
	rn.SM.LeaderId = -1
	//fmt.Println("["+strconv.Itoa(conf.Id)+"] 2.2.3")
	// if rn.StateFile.GetLastIndex() != -1 {
	// 	entry, err := rn.StateFile.Get(0)
	// 	state_entry := entry.(StateEntry)

	// 	rn.SM.CurrentTerm = state_entry.Term
	// 	rn.SM.VotedFor = state_entry.VotedFor
	// 	rn.SM.CurrTermLastLogIndex = state_entry.CurrTermLastLogIndex
	// } else {
	rn.StateFile.TruncateToEnd(0)
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
	rn.LogFile.TruncateToEnd(0)
	rn.SM.Log = append(rn.SM.Log, LogEntry{Term: 0, Data: nil})
	rn.LogFile.Append(LogEntry{Term: 0, Data: nil})
	//}
	//fmt.Println("["+strconv.Itoa(conf.Id)+"] 2.2.4")
	rn.SM.CommitIndex = 0
	rn.SM.LastApplied = 0
	for _, _ = range conf.Cluster {
		rn.SM.NextIndex = append(rn.SM.NextIndex, len(rn.SM.Log))
		rn.SM.MatchIndex = append(rn.SM.MatchIndex, 0)
	}

	//fmt.Println("["+strconv.Itoa(conf.Id)+"] 2.2.5")
	//go rn.ProcessTimers()
	go rn.ProcessNodeEvents()
	//fmt.Println("["+strconv.Itoa(conf.Id)+"] 2.2.6")

	rn.TimeoutTimer.Reset(time.Millisecond * time.Duration(conf.ElectionTimeout))
	//fmt.Println("["+strconv.Itoa(conf.Id)+"] 2.2.7")

	return &rn
}

/*func (RN *RaftNode) ProcessTimers() {
	for {
		RN.Mux.Lock()
		if RN.IsWorking {
			select {
			case <-RN.TimeoutTimer.C:
				RN.TimeoutEventCh <- TimeoutEvent{}
			default:
			}
		}
		RN.Mux.Unlock()
	}
}*/

func (RN *RaftNode) ProcessActions(actions []Action) {
	for _, act := range actions {
		switch act.(type) {

		case SendAction:
			send_act := act.(SendAction)
			// send_act.ev to outbox of send_act.peerid
			RN.NetServer.Outbox() <- &cluster.Envelope{Pid: send_act.PeerId, Msg: send_act.Ev}

		case CommitAction:
			commit_act := act.(CommitAction)
			RN.CommitCh <- CommitInfo{Data: commit_act.Data, Index: commit_act.Index, LeaderId: commit_act.LeaderId, Err: commit_act.Err}

		case AlarmAction:
			alarm_act := act.(AlarmAction)
			ret := RN.TimeoutTimer.Reset(time.Millisecond * time.Duration(alarm_act.Time))
			if !ret {
				RN.TimeoutTimer = time.NewTimer(time.Millisecond * time.Duration(alarm_act.Time))
			}

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
	for {
		RN.Mux.Lock()
		if RN.IsWorking {
			var ev Event

			select {
			case ev = <-RN.AppendEventCh:
				// Append Event
				actions := RN.SM.ProcessEvent(ev)
				RN.ProcessActions(actions)

			case <-RN.TimeoutTimer.C:
				// Timeout Event
				ev = TimeoutEvent{}
				actions := RN.SM.ProcessEvent(ev)
				RN.ProcessActions(actions)

			case inboxEvent := <-RN.NetServer.Inbox():
				// Network Event
				ev = inboxEvent.Msg
				actions := RN.SM.ProcessEvent(ev)
				RN.ProcessActions(actions)
			default:
			}
			//actions := RN.SM.ProcessEvent(ev)
			//RN.ProcessActions(actions)
		}
		RN.Mux.Unlock()
	}
}
