package main

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	//"time"
	//"sync"
	//"bufio"
	//"io"
	//"math"
	//"net"
	//"strings"
)

var debug_l1 bool = false
var debug_l2 bool = false
var debug_l3 bool = false
/*
type Peer struct {
	address string
}
*/

//-------------------- function to get random number between 'min' & 'max' --------------------

func random(min int, max int) int {
	return rand.Intn(max-min) + min
}

//-------------------- function to get minimum & maximum of two integers --------------------

func min(val1 int, val2 int) int {
	if val1 < val2 {
		return val1
	}
	return val2
}

func max(val1 int, val2 int) int {
	if val1 > val2 {
		return val1
	}
	return val2
}

// -------------------- configuration structures --------------------
/*
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
*/
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
	Index 	 int
	Data  	 []byte
	LeaderId int
	Err   	 error
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

func (SM *StateMachine) getPeerIndex(serverId int) (int, error) {
	for i, peer := range SM.Conf.Cluster {
		if peer.Id == serverId {
			return i, nil
		}
	}
	return 0, errors.New("SERVER_ID_NOT_FOUND") 
}

/*func (SM *StateMachine) Initialise(conf Config) {
	// initialise state machine

}*/

// -------------------- process event while in leader state --------------------

func (SM *StateMachine) ProcessLeaderEvent(ev Event) []Action {
	var act []Action

	old_State := SM.State
	old_CurrentTerm := SM.CurrentTerm
	old_CommitIndex := SM.CommitIndex	
	var event_name, extra string


	switch ev.(type) {

	case AppendEvent: // data []byte
		evnt := ev.(AppendEvent)
		event_name = "AppendEvent                  "	
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
		event_name = "TimeoutEvent                 "
		alarm_act := AlarmAction{SM.Conf.HeartbeatTimeout}
		act = append(act, alarm_act)

		for i, peer := range SM.Conf.Cluster {
			send_log := SM.Log[SM.NextIndex[i]:]
			appendEntriesReq_ev := AppendEntriesReqEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LeaderId: SM.Id, PrevLogIndex: SM.NextIndex[i] - 1, PrevLogTerm: SM.Log[SM.NextIndex[i]-1].Term, Entries: send_log, LeaderCommitIndex: SM.CommitIndex}
			send_act := SendAction{PeerId: peer.Id, Ev: appendEntriesReq_ev}
			act = append(act, send_act)
		}

	case AppendEntriesReqEvent:
		evnt := ev.(AppendEntriesReqEvent)
		event_name = "AppendEntriesReqEvent  <- "+strconv.Itoa(evnt.SenderId)
		extra = strconv.Itoa(evnt.Term) + "," + strconv.Itoa(evnt.PrevLogIndex) + "," + strconv.Itoa(evnt.PrevLogTerm) + "," + strconv.Itoa(evnt.LeaderCommitIndex) 
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
					prev_CommitIndex := SM.CommitIndex
					SM.CommitIndex = min(evnt.LeaderCommitIndex, len(SM.Log)-1)
					for i:=prev_CommitIndex+1; i<=SM.CommitIndex; i++ {
						commit_act := CommitAction{Index: i, Data: SM.Log[i].Data, LeaderId: SM.LeaderId, Err: nil}
						act = append(act, commit_act)
					}
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
					prev_CommitIndex := SM.CommitIndex
					SM.CommitIndex = min(evnt.LeaderCommitIndex, len(SM.Log)-1)
					for i:=prev_CommitIndex+1; i<=SM.CommitIndex; i++ {
						commit_act := CommitAction{Index: i, Data: SM.Log[i].Data, LeaderId: SM.LeaderId, Err: nil}
						act = append(act, commit_act)
					}
				}

				appendEntriesResp_ev := AppendEntriesRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LastLogIndex: len(SM.Log) - 1, Success: true}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: appendEntriesResp_ev}
				act = append(act, send_act)
			}
		}

	case AppendEntriesRespEvent:
		evnt := ev.(AppendEntriesRespEvent)
		event_name = "AppendEntriesRespEvent <- "+strconv.Itoa(evnt.SenderId)
		extra = strconv.Itoa(evnt.Term) + "," + strconv.Itoa(evnt.LastLogIndex) + "," + strconv.FormatBool(evnt.Success) 
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
			peer_index, err := SM.getPeerIndex(evnt.SenderId)
			if err != nil {
				fmt.Println(err)
			}
			if evnt.Success == true {
				if evnt.LastLogIndex > SM.MatchIndex[peer_index] {
					SM.MatchIndex[peer_index] = evnt.LastLogIndex
				}
				SM.NextIndex[peer_index] = SM.MatchIndex[peer_index] + 1

				for i := SM.MatchIndex[peer_index]; i > SM.CommitIndex; i-- {
					cnt := 1
					for j, _ := range SM.Conf.Cluster {
						if SM.MatchIndex[j] >= i {
							cnt++
						}
					}
					if (cnt > len(SM.Conf.Cluster)/2) && (SM.Log[i].Term == SM.CurrentTerm) {
						prev_CommitIndex := SM.CommitIndex
						SM.CommitIndex = i
						for i:=prev_CommitIndex+1; i<=SM.CommitIndex; i++ {
							commit_act := CommitAction{Index: i, Data: SM.Log[i].Data, LeaderId: SM.LeaderId, Err: nil}
							act = append(act, commit_act)
						}
						break
					}
				}

				if SM.MatchIndex[peer_index] < (len(SM.Log) - 1) {
					send_log := SM.Log[SM.NextIndex[peer_index]:]
					appendEntriesReq_ev := AppendEntriesReqEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LeaderId: SM.Id, PrevLogIndex: SM.NextIndex[peer_index] - 1, PrevLogTerm: SM.Log[SM.NextIndex[peer_index]-1].Term, Entries: send_log, LeaderCommitIndex: SM.CommitIndex}
					send_act := SendAction{PeerId: evnt.SenderId, Ev: appendEntriesReq_ev}
					act = append(act, send_act)
				}

			} else {
				SM.NextIndex[peer_index] = SM.NextIndex[peer_index] - 1
				fmt.Println("id --> ", SM.Id)
				fmt.Println("peer_index --> ", peer_index)
				fmt.Println("NextIndex[peer_index] --> ", SM.NextIndex[peer_index])
				fmt.Println(evnt)
				send_log := SM.Log[SM.NextIndex[peer_index]:]
				appendEntriesReq_ev := AppendEntriesReqEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LeaderId: SM.Id, PrevLogIndex: SM.NextIndex[peer_index] - 1, PrevLogTerm: SM.Log[SM.NextIndex[peer_index]-1].Term, Entries: send_log, LeaderCommitIndex: SM.CommitIndex}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: appendEntriesReq_ev}
				act = append(act, send_act)
			}
		}

	case VoteReqEvent:
		evnt := ev.(VoteReqEvent)
		event_name = "VoteReqEvent           <- "+strconv.Itoa(evnt.SenderId)
		extra = strconv.Itoa(evnt.Term) + "," + strconv.Itoa(evnt.LastLogIndex) + "," + strconv.Itoa(evnt.LastLogTerm) 
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
		event_name = "VoteRespEvent          <- "+strconv.Itoa(evnt.SenderId)
		extra = strconv.Itoa(evnt.Term) + "," + strconv.FormatBool(evnt.VoteGranted) 
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

	if debug_l1 {
		fmt.Println("[" + strconv.Itoa(SM.Id) + "] " + event_name + " ---- (" + old_State + "-" + strconv.Itoa(old_CurrentTerm) + "-" + strconv.Itoa(old_CommitIndex) + ") ----> (" + SM.State + "-" + strconv.Itoa(SM.CurrentTerm) + "-" + strconv.Itoa(SM.CommitIndex)+") ---- " + extra)
	}

	return act
}

// -------------------- process event while in follower state --------------------

func (SM *StateMachine) ProcessFollowerEvent(ev Event) []Action {
	var act []Action

	old_State := SM.State
	old_CurrentTerm := SM.CurrentTerm
	old_CommitIndex := SM.CommitIndex	
	var event_name, extra string

	switch ev.(type) {

	case AppendEvent:
		evnt := ev.(AppendEvent)
		event_name = "AppendEvent                  "
		commit_act := CommitAction{Index: 0, Data: evnt.Data, LeaderId: SM.LeaderId, Err: errors.New("ERR_CONTACT_LEADER")}
		act = append(act, commit_act)

	case TimeoutEvent:
		//evnt := ev.(TimeoutEvent)
		event_name = "TimeoutEvent                 "
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
		event_name = "AppendEntriesReqEvent  <- "+strconv.Itoa(evnt.SenderId)
		extra = strconv.Itoa(evnt.Term) + "," + strconv.Itoa(evnt.PrevLogIndex) + "," + strconv.Itoa(evnt.PrevLogTerm) + "," + strconv.Itoa(evnt.LeaderCommitIndex) 
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
					prev_CommitIndex := SM.CommitIndex
					SM.CommitIndex = min(evnt.LeaderCommitIndex, len(SM.Log)-1)
					for i:=prev_CommitIndex+1; i<=SM.CommitIndex; i++ {
						commit_act := CommitAction{Index: i, Data: SM.Log[i].Data, LeaderId: SM.LeaderId, Err: nil}
						act = append(act, commit_act)
					}
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
					prev_CommitIndex := SM.CommitIndex
					SM.CommitIndex = min(evnt.LeaderCommitIndex, len(SM.Log)-1)
					for i:=prev_CommitIndex+1; i<=SM.CommitIndex; i++ {
						commit_act := CommitAction{Index: i, Data: SM.Log[i].Data, LeaderId: SM.LeaderId, Err: nil}
						act = append(act, commit_act)
					}
				}

				appendEntriesResp_ev := AppendEntriesRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LastLogIndex: len(SM.Log) - 1, Success: true}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: appendEntriesResp_ev}
				act = append(act, send_act)
			}
		}

	case AppendEntriesRespEvent:
		evnt := ev.(AppendEntriesRespEvent)
		event_name = "AppendEntriesRespEvent <- "+strconv.Itoa(evnt.SenderId)
		extra = strconv.Itoa(evnt.Term) + "," + strconv.Itoa(evnt.LastLogIndex) + "," + strconv.FormatBool(evnt.Success) 
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
		event_name = "VoteReqEvent           <- "+strconv.Itoa(evnt.SenderId)
		extra = strconv.Itoa(evnt.Term) + "," + strconv.Itoa(evnt.LastLogIndex) + "," + strconv.Itoa(evnt.LastLogTerm) 
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
		event_name = "VoteRespEvent          <- "+strconv.Itoa(evnt.SenderId)
		extra = strconv.Itoa(evnt.Term) + "," + strconv.FormatBool(evnt.VoteGranted) 
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

	if debug_l1 {
		fmt.Println("[" + strconv.Itoa(SM.Id) + "] " + event_name + " ---- (" + old_State + "-" + strconv.Itoa(old_CurrentTerm) + "-" + strconv.Itoa(old_CommitIndex) + ") ----> (" + SM.State + "-" + strconv.Itoa(SM.CurrentTerm) + "-" + strconv.Itoa(SM.CommitIndex)+") ---- " + extra)
	}

	return act
}

// -------------------- process event while in candidate state --------------------

func (SM *StateMachine) ProcessCandidateEvent(ev Event) []Action {
	var act []Action

	old_State := SM.State
	old_CurrentTerm := SM.CurrentTerm
	old_CommitIndex := SM.CommitIndex
	var event_name, extra string	

	switch ev.(type) {

	case AppendEvent:
		evnt := ev.(AppendEvent)
		event_name = "AppendEvent                  "
		commit_act := CommitAction{Index: 0, Data: evnt.Data, LeaderId: SM.LeaderId, Err: errors.New("ERR_TRY_LATER")}
		act = append(act, commit_act)

	case TimeoutEvent:
		//evnt := ev.(TimeoutEvent)
		event_name = "TimeoutEvent                 "
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
		event_name = "AppendEntriesReqEvent  <- "+strconv.Itoa(evnt.SenderId)
		extra = strconv.Itoa(evnt.Term) + "," + strconv.Itoa(evnt.PrevLogIndex) + "," + strconv.Itoa(evnt.PrevLogTerm) + "," + strconv.Itoa(evnt.LeaderCommitIndex) 
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
					prev_CommitIndex := SM.CommitIndex
					SM.CommitIndex = min(evnt.LeaderCommitIndex, len(SM.Log)-1)
					for i:=prev_CommitIndex+1; i<=SM.CommitIndex; i++ {
						commit_act := CommitAction{Index: i, Data: SM.Log[i].Data, LeaderId: SM.LeaderId, Err: nil}
						act = append(act, commit_act)
					}
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
					prev_CommitIndex := SM.CommitIndex
					SM.CommitIndex = min(evnt.LeaderCommitIndex, len(SM.Log)-1)
					for i:=prev_CommitIndex+1; i<=SM.CommitIndex; i++ {
						commit_act := CommitAction{Index: i, Data: SM.Log[i].Data, LeaderId: SM.LeaderId, Err: nil}
						act = append(act, commit_act)
					}
				}

				appendEntriesResp_ev := AppendEntriesRespEvent{SenderId: SM.Id, Term: SM.CurrentTerm, LastLogIndex: len(SM.Log) - 1, Success: true}
				send_act := SendAction{PeerId: evnt.SenderId, Ev: appendEntriesResp_ev}
				act = append(act, send_act)
			}
		}
		
	case AppendEntriesRespEvent:
		evnt := ev.(AppendEntriesRespEvent)
		event_name = "AppendEntriesRespEvent <- "+strconv.Itoa(evnt.SenderId)
		extra = strconv.Itoa(evnt.Term) + "," + strconv.Itoa(evnt.LastLogIndex) + "," + strconv.FormatBool(evnt.Success) 
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
		event_name = "VoteReqEvent           <- "+strconv.Itoa(evnt.SenderId)
		extra = strconv.Itoa(evnt.Term) + "," + strconv.Itoa(evnt.LastLogIndex) + "," + strconv.Itoa(evnt.LastLogTerm) 
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
		event_name = "VoteRespEvent          <- "+strconv.Itoa(evnt.SenderId)
		extra = strconv.Itoa(evnt.Term) + "," + strconv.FormatBool(evnt.VoteGranted) 
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

	if debug_l1 {
		fmt.Println("[" + strconv.Itoa(SM.Id) + "] " + event_name + " ---- (" + old_State + "-" + strconv.Itoa(old_CurrentTerm) + "-" + strconv.Itoa(old_CommitIndex) + ") ----> (" + SM.State + "-" + strconv.Itoa(SM.CurrentTerm) + "-" + strconv.Itoa(SM.CommitIndex)+") ---- " + extra)
	}

	return act
}

// -------------------- processEvent function --------------------

func (SM *StateMachine) ProcessEvent(ev Event) []Action {
	//fmt.Println("In Process Event")
	var act []Action

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

/*func main() {
	rand.Seed(time.Now().Unix())
}*/
