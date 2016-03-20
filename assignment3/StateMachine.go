package main

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
	//"bufio"
	//"io"
	//"math"
	//"net"
	//"strconv"
	//"strings"
)

/*
type Peer struct {
	address string
}
*/

var election_timeout int = 100
var heartbeat_timeout int = 100

//-------------------- function to get random number between 'min' & 'max' --------------------

func random(min int, max int) int {
	return rand.Intn(max-min) + min
}

//-------------------- function to get minimum of two integers --------------------

func min(val1 int, val2 int) int {
	if val1 < val2 {
		return val1
	}
	return val2
}

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

// -------------------- event structures --------------------

type Event interface{}

type AppendEvent struct {
	Data []byte
}

type TimeoutEvent struct {
}

type AppendEntriesReqEvent struct {
	SenderId          int
	Term              int
	LeaderId          int
	PrevLogIndex      int
	PrevLogTerm       int
	Entries           []LogEntry
	LeaderCommitIndex int
}

type AppendEntriesRespEvent struct {
	SenderId     int
	Term         int
	LastLogIndex int
	Success      bool
}

type VoteReqEvent struct {
	SenderId     int
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

type VoteRespEvent struct {
	SenderId    int
	Term        int
	VoteGranted bool
}

// -------------------- action structures --------------------

type Action interface{}

type SendAction struct {
	PeerId int
	Ev     Event
}

type CommitAction struct {
	Index int
	Data  []byte
	Err   error
}

type AlarmAction struct {
	Time int
}

type LogStoreAction struct {
	Index int
	Term  int
	Data  []byte
}

type StateStoreAction struct {
	Term                 int
	VotedFor             int
	CurrTermLastLogIndex int
}

// -------------------- state machine structure --------------------

type LogEntry struct {
	Term int
	Data []byte
}

type StateEntry struct {
	Term                 int
	VotedFor             int
	CurrTermLastLogIndex int
}

type StateMachine struct {
	// ???????
	Id            int
	Conf          Config
	State         string
	NumOfVotes    int
	NumOfNegVotes int
	LeaderId      int
	Mux           sync.Mutex
	// persistannt state on all servers
	CurrentTerm          int
	VotedFor             int
	CurrTermLastLogIndex int
	Log                  []LogEntry
	// volatile state on all servers
	CommitIndex int
	LastApplied int
	// volatile state on leaders
	NextIndex  []int
	MatchIndex []int
}

// -------------------- state machine methodes --------------------

func (SM *StateMachine) getPeerIndex(serverId int) int {
	var index int

	for i, peer := range SM.Conf.Cluster {
		if peer.Id == serverId {
			index = i
			break
		}
	}

	return index
}

/*func (SM *StateMachine) Initialise(conf Config) {
	// initialise state machine

}*/

// -------------------- process event while in leader state --------------------

func (SM *StateMachine) ProcessLeaderEvent(ev Event) []Action {
	var act []Action

	switch ev.(type) {

	case AppendEvent: // data []byte
		evnt := ev.(AppendEvent)
		log_entry := LogEntry{Term: SM.CurrentTerm, Data: evnt.Data}
		SM.Log = append(SM.Log, log_entry)
		logStore_act := LogStoreAction{Index: len(SM.Log) - 1, Term: SM.CurrentTerm, Data: evnt.Data}
		act = append(act, logStore_act)

		for i, peer := range SM.Conf.Cluster {
			send_log := SM.Log[SM.NextIndex[i]:]
			appendEntriesReq_ev := AppendEntriesReqEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LeaderId: SM.Id, PrevLogIndex: SM.NextIndex[i] - 1, PrevLogTerm: SM.Log[SM.NextIndex[i]-1].Term, Entries: send_log, LeaderCommitIndex: SM.CommitIndex}
			send_act := SendAction{PeerId: peer.Id, Ev: appendEntriesReq_ev}
			act = append(act, send_act)
		}

	case TimeoutEvent:
		//evnt := ev.(TimeoutEvent)
		alarm_act := AlarmAction{SM.Conf.HeartbeatTimeout}
		act = append(act, alarm_act)

		for i, peer := range SM.Conf.Cluster {
			send_log := SM.Log[SM.NextIndex[i]:]
			appendEntriesReq_ev := AppendEntriesReqEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LeaderId: SM.Id, PrevLogIndex: SM.NextIndex[i] - 1, PrevLogTerm: SM.Log[SM.NextIndex[i]-1].Term, Entries: send_log, LeaderCommitIndex: SM.CommitIndex}
			send_act := SendAction{PeerId: peer.Id, Ev: appendEntriesReq_ev}
			act = append(act, send_act)
		}

	case AppendEntriesReqEvent: // SenderId int, Term int, LeaderId int, PrevLogIndex int, PrevLogTerm int, entries []LogEntry, LeaderCommitIndex int
		evnt := ev.(AppendEntriesReqEvent)
		if evnt.Term < SM.CurrentTerm {
			appendEntriesResp_ev := AppendEntriesRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LastLogIndex: len(SM.Log) - 1, Success: false}
			send_act := SendAction{PeerId: evnt.SenderId, Ev: appendEntriesResp_ev}
			act = append(act, send_act)
		} else {
			SM.State = "Follower"
			SM.CurrentTerm = evnt.Term
			SM.VotedFor = -1
			SM.LeaderId = evnt.SenderId
			SM.CurrTermLastLogIndex = -1

			alarm_act := AlarmAction{Time: random(SM.Conf.ElectionTimeout, 2*SM.Conf.ElectionTimeout)}
			act = append(act, alarm_act)

			stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
			act = append(act, stateStore_act)

			if (evnt.PrevLogIndex > len(SM.Log)-1) || (SM.Log[evnt.PrevLogIndex].Term != evnt.PrevLogTerm) {
				appendEntriesResp_ev := AppendEntriesRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LastLogIndex: len(SM.Log) - 1, Success: false}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: appendEntriesResp_ev}
				act = append(act, send_act)
			} else if SM.CurrTermLastLogIndex != -1 && SM.CurrTermLastLogIndex >= evnt.PrevLogIndex+len(evnt.Entries) {
				if SM.CommitIndex < (len(SM.Log)-1) && SM.CommitIndex < evnt.LeaderCommitIndex {
					SM.CommitIndex = min(evnt.LeaderCommitIndex, len(SM.Log)-1)
					commit_act := CommitAction{Index: SM.CommitIndex, Data: SM.Log[SM.CommitIndex].Data, Err: nil}
					act = append(act, commit_act)
				}
				appendEntriesResp_ev := AppendEntriesRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LastLogIndex: len(SM.Log) - 1, Success: true}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: appendEntriesResp_ev}
				act = append(act, send_act)
			} else {
				// how to delete entries in "Log" using logstore.....do we need to?????
				/*for i, log_entry := range SM.log[evnt.prevLogIndex+1 : ] {
					logStore_act := LogStoreAction{evnt.prevLogIndex+1+i, nil}
					act = append(act, logStore_act)
				}*/
				SM.Log = SM.Log[:evnt.PrevLogIndex+1]

				for _, log_entry := range evnt.Entries {
					SM.Log = append(SM.Log, log_entry)
					logStore_act := LogStoreAction{Index: len(SM.Log) - 1, Term: log_entry.Term, Data: log_entry.Data}
					act = append(act, logStore_act)
				}

				SM.CurrTermLastLogIndex = len(SM.Log) - 1

				stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
				act = append(act, stateStore_act)

				if SM.CommitIndex < (len(SM.Log)-1) && SM.CommitIndex < evnt.LeaderCommitIndex {
					SM.CommitIndex = min(evnt.LeaderCommitIndex, len(SM.Log)-1)
					commit_act := CommitAction{Index: SM.CommitIndex, Data: SM.Log[SM.CommitIndex].Data, Err: nil}
					act = append(act, commit_act)
				}

				appendEntriesResp_ev := AppendEntriesRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LastLogIndex: len(SM.Log) - 1, Success: true}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: appendEntriesResp_ev}
				act = append(act, send_act)
			}
		}

	case AppendEntriesRespEvent:
		evnt := ev.(AppendEntriesRespEvent)
		if evnt.Term > SM.CurrentTerm {
			SM.State = "Follower"
			SM.CurrentTerm = evnt.Term
			SM.VotedFor = -1
			SM.LeaderId = -1
			SM.CurrTermLastLogIndex = -1

			alarm_act := AlarmAction{Time: random(SM.Conf.ElectionTimeout, 2*SM.Conf.ElectionTimeout)}
			act = append(act, alarm_act)

			stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
			act = append(act, stateStore_act)
		} else {
			if evnt.Success == true {
				if evnt.LastLogIndex > SM.MatchIndex[SM.getPeerIndex(evnt.SenderId)] {
					SM.MatchIndex[SM.getPeerIndex(evnt.SenderId)] = evnt.LastLogIndex
				}
				SM.NextIndex[SM.getPeerIndex(evnt.SenderId)] = SM.MatchIndex[SM.getPeerIndex(evnt.SenderId)] + 1

				for i := SM.MatchIndex[SM.getPeerIndex(evnt.SenderId)]; i > SM.CommitIndex; i-- {
					cnt := 1
					for j, _ := range SM.Conf.Cluster {
						if SM.MatchIndex[j] >= i {
							cnt++
						}
					}
					if (cnt > len(SM.Conf.Cluster)/2) && (SM.Log[i].Term == SM.CurrentTerm) {
						SM.CommitIndex = i
						commit_act := CommitAction{Index: SM.CommitIndex, Data: SM.Log[SM.CommitIndex].Data, Err: nil}
						act = append(act, commit_act)
						break
					}
				}

				if SM.MatchIndex[SM.getPeerIndex(evnt.SenderId)] < (len(SM.Log) - 1) {
					send_log := SM.Log[SM.NextIndex[SM.getPeerIndex(evnt.SenderId)]:]
					appendEntriesReq_ev := AppendEntriesReqEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LeaderId: SM.Id, PrevLogIndex: SM.NextIndex[SM.getPeerIndex(evnt.SenderId)] - 1, PrevLogTerm: SM.Log[SM.NextIndex[SM.getPeerIndex(evnt.SenderId)]-1].Term, Entries: send_log, LeaderCommitIndex: SM.CommitIndex}
					send_act := SendAction{PeerId: evnt.SenderId, Ev: appendEntriesReq_ev}
					act = append(act, send_act)
				}

			} else {
				SM.NextIndex[evnt.SenderId] = SM.NextIndex[evnt.SenderId] - 1
				send_log := SM.Log[SM.NextIndex[SM.getPeerIndex(evnt.SenderId)]:]
				appendEntriesReq_ev := AppendEntriesReqEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LeaderId: SM.Id, PrevLogIndex: SM.NextIndex[SM.getPeerIndex(evnt.SenderId)] - 1, PrevLogTerm: SM.Log[SM.NextIndex[SM.getPeerIndex(evnt.SenderId)]-1].Term, Entries: send_log, LeaderCommitIndex: SM.CommitIndex}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: appendEntriesReq_ev}
				act = append(act, send_act)
			}
		}

	case VoteReqEvent:
		evnt := ev.(VoteReqEvent)
		if evnt.Term <= SM.CurrentTerm {
			voteResp_ev := VoteRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, VoteGranted: false}
			send_act := SendAction{PeerId: evnt.SenderId, Ev: voteResp_ev}
			act = append(act, send_act)
		} else {
			SM.State = "Follower"
			SM.CurrentTerm = evnt.Term
			SM.VotedFor = -1
			SM.LeaderId = -1
			SM.CurrTermLastLogIndex = -1

			alarm_act := AlarmAction{Time: random(SM.Conf.ElectionTimeout, 2*SM.Conf.ElectionTimeout)}
			act = append(act, alarm_act)

			stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
			act = append(act, stateStore_act)

			if (evnt.LastLogTerm < SM.Log[len(SM.Log)-1].Term) || ((evnt.LastLogTerm == SM.Log[len(SM.Log)-1].Term) && (evnt.LastLogIndex < len(SM.Log)-1)) {
				voteResp_ev := VoteRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, VoteGranted: false}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: voteResp_ev}
				act = append(act, send_act)
			} else {
				SM.VotedFor = evnt.SenderId
				stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
				act = append(act, stateStore_act)
				voteResp_ev := VoteRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, VoteGranted: true}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: voteResp_ev}
				act = append(act, send_act)
			}
		}

	case VoteRespEvent:
		evnt := ev.(VoteRespEvent)
		if evnt.Term > SM.CurrentTerm {
			SM.State = "Follower"
			SM.CurrentTerm = evnt.Term
			SM.VotedFor = -1
			SM.LeaderId = -1
			SM.CurrTermLastLogIndex = -1

			alarm_act := AlarmAction{Time: random(SM.Conf.ElectionTimeout, 2*SM.Conf.ElectionTimeout)}
			act = append(act, alarm_act)

			stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
			act = append(act, stateStore_act)
		}
	}

	return act
}

// -------------------- process event while in follower state --------------------

func (SM *StateMachine) ProcessFollowerEvent(ev Event) []Action {
	var act []Action

	switch ev.(type) {

	case AppendEvent:
		evnt := ev.(AppendEvent)
		commit_act := CommitAction{Index: 0, Data: evnt.Data, Err: errors.New("ERR_CONTACT_LEADER")}
		act = append(act, commit_act)

	case TimeoutEvent:
		//evnt := ev.(TimeoutEvent)
		SM.State = "Candidate"
		SM.CurrentTerm = SM.CurrentTerm + 1
		SM.VotedFor = SM.Id
		SM.NumOfVotes = 1
		SM.CurrTermLastLogIndex = -1
		SM.NumOfNegVotes = 0
		SM.LeaderId = -1

		alarm_act := AlarmAction{Time: random(SM.Conf.ElectionTimeout, 2*SM.Conf.ElectionTimeout)}
		act = append(act, alarm_act)

		stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
		act = append(act, stateStore_act)

		for _, peer := range SM.Conf.Cluster {
			voteReq_ev := VoteReqEvent{SenderId: SM.Id, Term: SM.CurrentTerm, CandidateId: SM.Id, LastLogIndex: len(SM.Log) - 1, LastLogTerm: SM.Log[len(SM.Log)-1].Term}
			send_act := SendAction{PeerId: peer.Id, Ev: voteReq_ev}
			act = append(act, send_act)
		}

	case AppendEntriesReqEvent:
		evnt := ev.(AppendEntriesReqEvent)
		if evnt.Term < SM.CurrentTerm {
			appendEntriesResp_ev := AppendEntriesRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LastLogIndex: len(SM.Log) - 1, Success: false}
			send_act := SendAction{PeerId: evnt.SenderId, Ev: appendEntriesResp_ev}
			act = append(act, send_act)
		} else {
			alarm_act := AlarmAction{Time: random(SM.Conf.ElectionTimeout, 2*SM.Conf.ElectionTimeout)}
			act = append(act, alarm_act)

			if evnt.Term > SM.CurrentTerm {
				SM.CurrentTerm = evnt.Term
				SM.VotedFor = -1
				SM.CurrTermLastLogIndex = -1

				stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
				act = append(act, stateStore_act)
			}
			SM.LeaderId = evnt.SenderId

			if (evnt.PrevLogIndex > len(SM.Log)-1) || (SM.Log[evnt.PrevLogIndex].Term != evnt.PrevLogTerm) {
				appendEntriesResp_ev := AppendEntriesRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LastLogIndex: len(SM.Log) - 1, Success: false}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: appendEntriesResp_ev}
				act = append(act, send_act)
			} else if SM.CurrTermLastLogIndex != -1 && SM.CurrTermLastLogIndex >= evnt.PrevLogIndex+len(evnt.Entries) {
				if SM.CommitIndex < (len(SM.Log)-1) && SM.CommitIndex < evnt.LeaderCommitIndex {
					SM.CommitIndex = min(evnt.LeaderCommitIndex, len(SM.Log)-1)
					commit_act := CommitAction{Index: SM.CommitIndex, Data: SM.Log[SM.CommitIndex].Data, Err: nil}
					act = append(act, commit_act)
				}
				appendEntriesResp_ev := AppendEntriesRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LastLogIndex: len(SM.Log) - 1, Success: true}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: appendEntriesResp_ev}
				act = append(act, send_act)
			} else {
				// how to delete entries in "Log" using logstore.....do we need to?????
				/*for i, log_entry := range SM.log[evnt.prevLogIndex+1 : ] {
					logStore_act := LogStoreAction{evnt.prevLogIndex+1+i, nil}
					act = append(act, logStore_act)
				}*/
				SM.Log = SM.Log[:evnt.PrevLogIndex+1]

				for _, log_entry := range evnt.Entries {
					SM.Log = append(SM.Log, log_entry)
					logStore_act := LogStoreAction{Index: len(SM.Log) - 1, Term: log_entry.Term, Data: log_entry.Data}
					act = append(act, logStore_act)
				}

				SM.CurrTermLastLogIndex = len(SM.Log) - 1

				stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
				act = append(act, stateStore_act)

				if SM.CommitIndex < (len(SM.Log)-1) && SM.CommitIndex < evnt.LeaderCommitIndex {
					SM.CommitIndex = min(evnt.LeaderCommitIndex, len(SM.Log)-1)
					commit_act := CommitAction{Index: SM.CommitIndex, Data: SM.Log[SM.CommitIndex].Data, Err: nil}
					act = append(act, commit_act)
				}

				appendEntriesResp_ev := AppendEntriesRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LastLogIndex: len(SM.Log) - 1, Success: true}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: appendEntriesResp_ev}
				act = append(act, send_act)
			}
		}

	case AppendEntriesRespEvent:
		evnt := ev.(AppendEntriesRespEvent)
		if evnt.Term > SM.CurrentTerm {
			// Don't know whether this case will occur or not.....when it will occur???.....what to do????
			SM.CurrentTerm = evnt.Term
			SM.VotedFor = -1
			SM.LeaderId = -1
			SM.CurrTermLastLogIndex = -1

			alarm_act := AlarmAction{Time: random(SM.Conf.ElectionTimeout, 2*SM.Conf.ElectionTimeout)}
			act = append(act, alarm_act)

			stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
			act = append(act, stateStore_act)
		}

	case VoteReqEvent:
		evnt := ev.(VoteReqEvent)
		if evnt.Term < SM.CurrentTerm {
			voteResp_ev := VoteRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, VoteGranted: false}
			send_act := SendAction{PeerId: evnt.SenderId, Ev: voteResp_ev}
			act = append(act, send_act)
		} else {
			if evnt.Term > SM.CurrentTerm {
				SM.CurrentTerm = evnt.Term
				SM.VotedFor = -1
				SM.LeaderId = -1
				SM.CurrTermLastLogIndex = -1

				alarm_act := AlarmAction{Time: random(SM.Conf.ElectionTimeout, 2*SM.Conf.ElectionTimeout)}
				act = append(act, alarm_act)

				stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
				act = append(act, stateStore_act)
			}

			if (evnt.LastLogTerm < SM.Log[len(SM.Log)-1].Term) || ((evnt.LastLogTerm == SM.Log[len(SM.Log)-1].Term) && (evnt.LastLogIndex < len(SM.Log)-1)) {
				voteResp_ev := VoteRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, VoteGranted: false}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: voteResp_ev}
				act = append(act, send_act)
			} else if SM.VotedFor != -1 && SM.VotedFor != evnt.SenderId {
				voteResp_ev := VoteRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, VoteGranted: false}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: voteResp_ev}
				act = append(act, send_act)
			} else {
				alarm_act := AlarmAction{Time: random(SM.Conf.ElectionTimeout, 2*SM.Conf.ElectionTimeout)}
				act = append(act, alarm_act)
				SM.VotedFor = evnt.SenderId
				stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
				act = append(act, stateStore_act)
				voteResp_ev := VoteRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, VoteGranted: true}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: voteResp_ev}
				act = append(act, send_act)
			}
		}

	case VoteRespEvent:
		evnt := ev.(VoteRespEvent)
		if evnt.Term > SM.CurrentTerm {
			// Don't know whether this case will occur or not.....when it will occur???.....what to do????
			SM.CurrentTerm = evnt.Term
			SM.VotedFor = -1
			SM.LeaderId = -1
			SM.CurrTermLastLogIndex = -1

			alarm_act := AlarmAction{Time: random(SM.Conf.ElectionTimeout, 2*SM.Conf.ElectionTimeout)}
			act = append(act, alarm_act)

			stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
			act = append(act, stateStore_act)
		}
	}

	return act
}

// -------------------- process event while in candidate state --------------------

func (SM *StateMachine) ProcessCandidateEvent(ev Event) []Action {
	var act []Action

	switch ev.(type) {

	case AppendEvent:
		evnt := ev.(AppendEvent)
		commit_act := CommitAction{Index: 0, Data: evnt.Data, Err: errors.New("ERR_TRY_LATER")}
		act = append(act, commit_act)

	case TimeoutEvent:
		//evnt := ev.(TimeoutEvent)
		SM.CurrentTerm = SM.CurrentTerm + 1
		SM.VotedFor = SM.Id
		SM.NumOfVotes = 1
		SM.NumOfNegVotes = 0
		SM.LeaderId = -1
		SM.CurrTermLastLogIndex = -1

		alarm_act := AlarmAction{Time: random(SM.Conf.ElectionTimeout, 2*SM.Conf.ElectionTimeout)}
		act = append(act, alarm_act)

		stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
		act = append(act, stateStore_act)

		for _, peer := range SM.Conf.Cluster {
			voteReq_ev := VoteReqEvent{SenderId: SM.Id, Term: SM.CurrentTerm, CandidateId: SM.Id, LastLogIndex: len(SM.Log) - 1, LastLogTerm: SM.Log[len(SM.Log)-1].Term}
			send_act := SendAction{PeerId: peer.Id, Ev: voteReq_ev}
			act = append(act, send_act)
		}

	case AppendEntriesReqEvent:
		evnt := ev.(AppendEntriesReqEvent)
		if evnt.Term < SM.CurrentTerm {
			appendEntriesResp_ev := AppendEntriesRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LastLogIndex: len(SM.Log) - 1, Success: false}
			send_act := SendAction{PeerId: evnt.SenderId, Ev: appendEntriesResp_ev}
			act = append(act, send_act)
		} else {
			SM.State = "Follower"

			alarm_act := AlarmAction{Time: random(SM.Conf.ElectionTimeout, 2*SM.Conf.ElectionTimeout)}
			act = append(act, alarm_act)

			if evnt.Term > SM.CurrentTerm {
				SM.CurrentTerm = evnt.Term
				SM.VotedFor = -1
				SM.CurrTermLastLogIndex = -1

				stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
				act = append(act, stateStore_act)
			}
			SM.LeaderId = evnt.SenderId

			if (evnt.PrevLogIndex > len(SM.Log)-1) || (SM.Log[evnt.PrevLogIndex].Term != evnt.PrevLogTerm) {
				appendEntriesResp_ev := AppendEntriesRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LastLogIndex: len(SM.Log) - 1, Success: false}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: appendEntriesResp_ev}
				act = append(act, send_act)
			} else if SM.CurrTermLastLogIndex != -1 && SM.CurrTermLastLogIndex >= evnt.PrevLogIndex+len(evnt.Entries) {
				if SM.CommitIndex < (len(SM.Log)-1) && SM.CommitIndex < evnt.LeaderCommitIndex {
					SM.CommitIndex = min(evnt.LeaderCommitIndex, len(SM.Log)-1)
					commit_act := CommitAction{Index: SM.CommitIndex, Data: SM.Log[SM.CommitIndex].Data, Err: nil}
					act = append(act, commit_act)
				}
				appendEntriesResp_ev := AppendEntriesRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LastLogIndex: len(SM.Log) - 1, Success: true}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: appendEntriesResp_ev}
				act = append(act, send_act)
			} else {
				// how to delete entries in "Log" using logstore.....do we need to?????
				/*for i, log_entry := range SM.log[evnt.prevLogIndex+1 : ] {
					logStore_act := LogStoreAction{evnt.prevLogIndex+1+i, nil}
					act = append(act, logStore_act)
				}*/
				SM.Log = SM.Log[:evnt.PrevLogIndex+1]

				for _, log_entry := range evnt.Entries {
					SM.Log = append(SM.Log, log_entry)
					logStore_act := LogStoreAction{Index: len(SM.Log) - 1, Term: log_entry.Term, Data: log_entry.Data}
					act = append(act, logStore_act)
				}

				SM.CurrTermLastLogIndex = len(SM.Log) - 1

				stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
				act = append(act, stateStore_act)

				if SM.CommitIndex < (len(SM.Log)-1) && SM.CommitIndex < evnt.LeaderCommitIndex {
					SM.CommitIndex = min(evnt.LeaderCommitIndex, len(SM.Log)-1)
					commit_act := CommitAction{Index: SM.CommitIndex, Data: SM.Log[SM.CommitIndex].Data, Err: nil}
					act = append(act, commit_act)
				}

				appendEntriesResp_ev := AppendEntriesRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LastLogIndex: len(SM.Log) - 1, Success: true}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: appendEntriesResp_ev}
				act = append(act, send_act)
			}
		}

	case AppendEntriesRespEvent:
		evnt := ev.(AppendEntriesRespEvent)
		if evnt.Term > SM.CurrentTerm {
			// Don't know whether this case will occur or not.....when it will occur???.....what to do????
			SM.State = "Follower"
			SM.CurrentTerm = evnt.Term
			SM.VotedFor = -1
			SM.LeaderId = -1
			SM.CurrTermLastLogIndex = -1

			alarm_act := AlarmAction{Time: random(SM.Conf.ElectionTimeout, 2*SM.Conf.ElectionTimeout)}
			act = append(act, alarm_act)

			stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
			act = append(act, stateStore_act)
		}

	case VoteReqEvent:
		evnt := ev.(VoteReqEvent)
		if evnt.Term <= SM.CurrentTerm {
			voteResp_ev := VoteRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, VoteGranted: false}
			send_act := SendAction{PeerId: evnt.SenderId, Ev: voteResp_ev}
			act = append(act, send_act)
		} else {
			SM.State = "Follower"
			SM.CurrentTerm = evnt.Term
			SM.VotedFor = -1
			SM.LeaderId = -1
			SM.CurrTermLastLogIndex = -1

			alarm_act := AlarmAction{Time: random(SM.Conf.ElectionTimeout, 2*SM.Conf.ElectionTimeout)}
			act = append(act, alarm_act)

			stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
			act = append(act, stateStore_act)

			if (evnt.LastLogTerm < SM.Log[len(SM.Log)-1].Term) || ((evnt.LastLogTerm == SM.Log[len(SM.Log)-1].Term) && (evnt.LastLogIndex < len(SM.Log)-1)) {
				voteResp_ev := VoteRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, VoteGranted: false}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: voteResp_ev}
				act = append(act, send_act)
			} else {
				SM.VotedFor = evnt.SenderId
				stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
				act = append(act, stateStore_act)
				voteResp_ev := VoteRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, VoteGranted: true}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: voteResp_ev}
				act = append(act, send_act)
			}
		}

	case VoteRespEvent:
		evnt := ev.(VoteRespEvent)
		if evnt.Term > SM.CurrentTerm {
			SM.State = "Follower"
			SM.CurrentTerm = evnt.Term
			SM.VotedFor = -1
			SM.LeaderId = -1
			SM.CurrTermLastLogIndex = -1

			alarm_act := AlarmAction{Time: random(SM.Conf.ElectionTimeout, 2*SM.Conf.ElectionTimeout)}
			act = append(act, alarm_act)

			stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
			act = append(act, stateStore_act)
		} else if evnt.Term == SM.CurrentTerm {
			if evnt.VoteGranted == true {
				SM.NumOfVotes = SM.NumOfVotes + 1
				if SM.NumOfVotes > len(SM.Conf.Cluster)/2 {
					SM.State = "Leader"
					SM.LeaderId = SM.Id

					for i, _ := range SM.Conf.Cluster {
						SM.NextIndex[i] = len(SM.Log)
						SM.MatchIndex[i] = 0
					}

					alarm_act := AlarmAction{SM.Conf.HeartbeatTimeout}
					act = append(act, alarm_act)

					for i, peer := range SM.Conf.Cluster {
						send_log := SM.Log[SM.NextIndex[i]:]
						appendEntriesReq_ev := AppendEntriesReqEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LeaderId: SM.Id, PrevLogIndex: SM.NextIndex[i] - 1, PrevLogTerm: SM.Log[SM.NextIndex[i]-1].Term, Entries: send_log, LeaderCommitIndex: SM.CommitIndex}
						send_act := SendAction{PeerId: peer.Id, Ev: appendEntriesReq_ev}
						act = append(act, send_act)
					}
				}
			} else {
				SM.NumOfNegVotes = SM.NumOfNegVotes + 1
				if SM.NumOfNegVotes > len(SM.Conf.Cluster)/2 {
					SM.State = "Follower"
					SM.VotedFor = -1
					SM.LeaderId = -1
					SM.CurrTermLastLogIndex = -1

					alarm_act := AlarmAction{Time: random(SM.Conf.ElectionTimeout, 2*SM.Conf.ElectionTimeout)}
					act = append(act, alarm_act)

					stateStore_act := StateStoreAction{Term: SM.CurrentTerm, VotedFor: SM.VotedFor, CurrTermLastLogIndex: SM.CurrTermLastLogIndex}
					act = append(act, stateStore_act)
				}
			}
		}
	}

	return act
}

// -------------------- processEvent function --------------------

func (SM *StateMachine) ProcessEvent(ev Event) []Action {
	//fmt.Println("In Process Event")
	var act []Action

	SM.Mux.Lock()
	defer SM.Mux.Unlock()

	switch SM.State {
	case "Leader":
		act = SM.ProcessLeaderEvent(ev)
	case "Follower":
		act = SM.ProcessFollowerEvent(ev)
	case "Candidate":
		act = SM.ProcessCandidateEvent(ev)
	}

	return act
}

// -------------------- main function --------------------

func main() {
	rand.Seed(time.Now().Unix())

	fmt.Println("abc")
}
