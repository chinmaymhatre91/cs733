package main

import (
	//"bufio"
	"fmt"
	//"io"
	//"math"
	"math/rand"
	//"net"
	//"strconv"
	//"strings"
	"time"
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

// -------------------- event structures --------------------

type Event interface{}

type AppendEvent struct {
	data []byte
}

type TimeoutEvent struct {
}

type AppendEntriesReqEvent struct {
	senderId     int
	term         int
	leaderId     int
	prevLogIndex int
	prevLogTerm  int
	entries      []LogEntry
	leaderCommit int
}

type AppendEntriesRespEvent struct {
	senderId     int
	term         int
	lastLogIndex int
	success      bool
}

type VoteReqEvent struct {
	senderId     int
	term         int
	candidateId  int
	lastLogIndex int
	lastLogTerm  int
}

type VoteRespEvent struct {
	senderId    int
	term        int
	voteGranted bool
}

// -------------------- action structures --------------------

type Action interface{}

type SendAction struct {
	peerId int
	ev     Event
}

type CommitAction struct {
	index int
	data  []byte
	err   string
}

type AlarmAction struct {
	time int
}

type LogStoreAction struct {
	index int
	term  int
	data  []byte
}

type StateStoreAction struct {
	term                 int
	votedFor             int
	currTermLastLogIndex int
}

// -------------------- state machine structure --------------------

type LogEntry struct {
	term int
	data []byte
}

type StateMachine struct {
	// ???????
	state         string
	peers         []int
	numOfVotes    int
	numOfNegVotes int
	leaderId      int
	// persistannt state on all servers
	id                   int
	currentTerm          int
	votedFor             int
	currTermLastLogIndex int
	log                  []LogEntry
	// volatile state on all servers
	commitIndex int
	lastApplied int
	// volatile state on leaders
	nextIndex  []int
	matchIndex []int
}

// -------------------- state machine methodes --------------------

func (sm *StateMachine) getPeerIndex(serverId int) int {
	var index int

	for i, peerId := range sm.peers {
		if peerId == serverId {
			index = i
			break
		}
	}

	return index
}

func getPersistantId() int {
	return 0
}

func getPersistantPeers() []int {
	peers := []int{1, 2, 3, 4}
	return peers
}

func getPersistantCurrentTerm() int {
	return 0
}

func getPersistantVotedFor() int {
	return -1
}

func getPersistantCurrTermLastLogIndex() int {
	return -1
}

func getPersistantLog() []LogEntry {
	log := []LogEntry{}
	log = append(log, LogEntry{0, nil})
	return log
}

func (sm *StateMachine) initialise() {
	sm.state = "Follower"
	sm.id = getPersistantId()
	sm.peers = getPersistantPeers()
	sm.currentTerm = getPersistantCurrentTerm()
	sm.votedFor = getPersistantVotedFor()
	sm.currTermLastLogIndex = getPersistantCurrTermLastLogIndex()
	sm.log = getPersistantLog()
	sm.commitIndex = 0
	sm.lastApplied = 0
	sm.numOfVotes = 0
	sm.numOfNegVotes = 0
	sm.leaderId = -1
	for _, _ = range sm.peers {
		sm.nextIndex = append(sm.nextIndex, len(sm.log))
		sm.matchIndex = append(sm.matchIndex, 0)
	}
}

// -------------------- process event while in leader state --------------------

func processLeaderEvent(sm *StateMachine, ev Event) []Action {
	var act []Action

	switch ev.(type) {

	case AppendEvent: // data []byte
		evnt := ev.(AppendEvent)
		log_entry := LogEntry{sm.currentTerm, evnt.data}
		sm.log = append(sm.log, log_entry)
		logStore_act := LogStoreAction{len(sm.log) - 1, sm.currentTerm, evnt.data}
		act = append(act, logStore_act)
		for i, peerId := range sm.peers {
			send_log := sm.log[sm.nextIndex[i]:]
			appendEntriesReq_ev := AppendEntriesReqEvent{sm.id, sm.currentTerm, sm.id, sm.nextIndex[i] - 1, sm.log[sm.nextIndex[i]-1].term, send_log, sm.commitIndex}
			send_act := SendAction{peerId, appendEntriesReq_ev}
			act = append(act, send_act)
		}

	case TimeoutEvent:
		//evnt := ev.(TimeoutEvent)
		alarm_act := AlarmAction{heartbeat_timeout}
		act = append(act, alarm_act)
		for i, peerId := range sm.peers {
			appendEntriesReq_ev := AppendEntriesReqEvent{sm.id, sm.currentTerm, sm.id, sm.nextIndex[i] - 1, sm.log[sm.nextIndex[i]-1].term, nil, sm.commitIndex}
			send_act := SendAction{peerId, appendEntriesReq_ev}
			act = append(act, send_act)
		}

	case AppendEntriesReqEvent: // senderId int, term int, leaderId int, prevLogIndex int, prevLogTerm int, entries []LogEntry, leaderCommit int
		evnt := ev.(AppendEntriesReqEvent)
		if evnt.term < sm.currentTerm {
			appendEntriesResp_ev := AppendEntriesRespEvent{sm.id, sm.currentTerm, len(sm.log) - 1, false}
			send_act := SendAction{evnt.senderId, appendEntriesResp_ev}
			act = append(act, send_act)
		} else {
			sm.state = "Follower"
			sm.currentTerm = evnt.term
			sm.votedFor = -1
			sm.leaderId = evnt.senderId
			sm.currTermLastLogIndex = -1

			stateStore_act := StateStoreAction{sm.currentTerm, sm.votedFor, sm.currTermLastLogIndex}
			act = append(act, stateStore_act)

			alarm_act := AlarmAction{random(election_timeout, 2*election_timeout)}
			act = append(act, alarm_act)

			if (evnt.prevLogIndex > len(sm.log)-1) || (sm.log[evnt.prevLogIndex].term != evnt.prevLogTerm) {
				appendEntriesResp_ev := AppendEntriesRespEvent{sm.id, sm.currentTerm, len(sm.log) - 1, false}
				send_act := SendAction{evnt.senderId, appendEntriesResp_ev}
				act = append(act, send_act)
			} else if sm.currTermLastLogIndex != -1 && sm.currTermLastLogIndex >= evnt.prevLogIndex+len(evnt.entries) {
				appendEntriesResp_ev := AppendEntriesRespEvent{sm.id, sm.currentTerm, len(sm.log) - 1, true}
				send_act := SendAction{evnt.senderId, appendEntriesResp_ev}
				act = append(act, send_act)
			} else {
				// how to delete entries in "Log" using logstore.....do we need to?????
				/*for i, log_entry := range sm.log[evnt.prevLogIndex+1 : ] {
					logStore_act := LogStoreAction{evnt.prevLogIndex+1+i, nil}
					act = append(act, logStore_act)
				}*/
				sm.log = sm.log[:evnt.prevLogIndex+1]

				for _, log_entry := range evnt.entries {
					sm.log = append(sm.log, log_entry)
					logStore_act := LogStoreAction{len(sm.log) - 1, log_entry.term, log_entry.data}
					act = append(act, logStore_act)
				}

				sm.currTermLastLogIndex = len(sm.log) - 1
				sm.commitIndex = min(evnt.leaderCommit, len(sm.log)-1)

				appendEntriesResp_ev := AppendEntriesRespEvent{sm.id, sm.currentTerm, len(sm.log) - 1, true}
				send_act := SendAction{evnt.senderId, appendEntriesResp_ev}
				act = append(act, send_act)
			}
		}

	case AppendEntriesRespEvent:
		evnt := ev.(AppendEntriesRespEvent)
		if evnt.term > sm.currentTerm {
			sm.state = "Follower"
			sm.currentTerm = evnt.term
			sm.votedFor = -1
			sm.leaderId = -1
			sm.currTermLastLogIndex = -1

			stateStore_act := StateStoreAction{sm.currentTerm, sm.votedFor, sm.currTermLastLogIndex}
			act = append(act, stateStore_act)
		} else {
			if evnt.success == true {
				if evnt.lastLogIndex+1 > sm.nextIndex[sm.getPeerIndex(evnt.senderId)] {
					sm.nextIndex[sm.getPeerIndex(evnt.senderId)] = evnt.lastLogIndex + 1
				}
				if evnt.lastLogIndex > sm.matchIndex[sm.getPeerIndex(evnt.senderId)] {
					sm.matchIndex[sm.getPeerIndex(evnt.senderId)] = evnt.lastLogIndex
				}

				for i := sm.matchIndex[sm.getPeerIndex(evnt.senderId)]; i > sm.commitIndex; i-- {
					cnt := 1
					for _, peerId := range sm.peers {
						if sm.matchIndex[peerId] >= i {
							cnt++
						}
					}
					if (cnt > (len(sm.peers)+1)/2) && (sm.log[i].term == sm.currentTerm) {
						sm.commitIndex = i
						break
					}
				}

				if sm.matchIndex[sm.getPeerIndex(evnt.senderId)] < (len(sm.log) - 1) {
					send_log := sm.log[sm.nextIndex[sm.getPeerIndex(evnt.senderId)]:]
					appendEntriesReq_ev := AppendEntriesReqEvent{sm.id, sm.currentTerm, sm.id, sm.nextIndex[sm.getPeerIndex(evnt.senderId)] - 1, sm.log[sm.nextIndex[sm.getPeerIndex(evnt.senderId)]-1].term, send_log, sm.commitIndex}
					send_act := SendAction{evnt.senderId, appendEntriesReq_ev}
					act = append(act, send_act)
				}

			} else {
				sm.nextIndex[evnt.senderId] = sm.nextIndex[evnt.senderId] - 1
				send_log := sm.log[sm.nextIndex[sm.getPeerIndex(evnt.senderId)]:]
				appendEntriesReq_ev := AppendEntriesReqEvent{sm.id, sm.currentTerm, sm.id, sm.nextIndex[sm.getPeerIndex(evnt.senderId)] - 1, sm.log[sm.nextIndex[sm.getPeerIndex(evnt.senderId)]-1].term, send_log, sm.commitIndex}
				send_act := SendAction{evnt.senderId, appendEntriesReq_ev}
				act = append(act, send_act)
			}
		}

	case VoteReqEvent:
		evnt := ev.(VoteReqEvent)
		if evnt.term <= sm.currentTerm {
			voteResp_ev := VoteRespEvent{sm.id, sm.currentTerm, false}
			send_act := SendAction{evnt.senderId, voteResp_ev}
			act = append(act, send_act)
		} else {
			sm.state = "Follower"
			sm.currentTerm = evnt.term
			sm.votedFor = -1
			sm.leaderId = -1
			sm.currTermLastLogIndex = -1

			stateStore_act := StateStoreAction{sm.currentTerm, sm.votedFor, sm.currTermLastLogIndex}
			act = append(act, stateStore_act)

			alarm_act := AlarmAction{random(election_timeout, 2*election_timeout)}
			act = append(act, alarm_act)

			if (evnt.lastLogTerm < sm.log[len(sm.log)-1].term) || ((evnt.lastLogTerm == sm.log[len(sm.log)-1].term) && (evnt.lastLogIndex < len(sm.log)-1)) {
				voteResp_ev := VoteRespEvent{sm.id, sm.currentTerm, false}
				send_act := SendAction{evnt.senderId, voteResp_ev}
				act = append(act, send_act)
			} else {
				sm.votedFor = evnt.senderId
				voteResp_ev := VoteRespEvent{sm.id, sm.currentTerm, true}
				send_act := SendAction{evnt.senderId, voteResp_ev}
				act = append(act, send_act)
			}
		}

	case VoteRespEvent:
		evnt := ev.(VoteRespEvent)
		if evnt.term > sm.currentTerm {
			sm.state = "Follower"
			sm.currentTerm = evnt.term
			sm.votedFor = -1
			sm.currTermLastLogIndex = -1

			stateStore_act := StateStoreAction{sm.currentTerm, sm.votedFor, sm.currTermLastLogIndex}
			act = append(act, stateStore_act)

			alarm_act := AlarmAction{random(election_timeout, 2*election_timeout)}
			act = append(act, alarm_act)
		}
	}

	return act
}

// -------------------- process event while in follower state --------------------

func processFollowerEvent(sm *StateMachine, ev Event) []Action {
	var act []Action

	switch ev.(type) {

	case AppendEvent:
		evnt := ev.(AppendEvent)
		commit_act := CommitAction{0, evnt.data, "ERR_CONTACT_LEADER"}
		act = append(act, commit_act)

	case TimeoutEvent:
		//evnt := ev.(TimeoutEvent)
		sm.state = "Candidate"
		sm.currentTerm = sm.currentTerm + 1
		sm.votedFor = sm.id
		sm.numOfVotes = 1
		sm.currTermLastLogIndex = -1
		sm.numOfNegVotes = 0
		sm.leaderId = -1

		stateStore_act := StateStoreAction{sm.currentTerm, sm.votedFor, sm.currTermLastLogIndex}
		act = append(act, stateStore_act)

		alarm_act := AlarmAction{random(election_timeout, 2*election_timeout)}
		act = append(act, alarm_act)

		for _, peerId := range sm.peers {
			voteReq_ev := VoteReqEvent{sm.id, sm.currentTerm, sm.id, len(sm.log) - 1, sm.log[len(sm.log)-1].term}
			send_act := SendAction{peerId, voteReq_ev}
			act = append(act, send_act)
		}

	case AppendEntriesReqEvent:
		evnt := ev.(AppendEntriesReqEvent)
		if evnt.term < sm.currentTerm {
			appendEntriesResp_ev := AppendEntriesRespEvent{sm.id, sm.currentTerm, len(sm.log) - 1, false}
			send_act := SendAction{evnt.senderId, appendEntriesResp_ev}
			act = append(act, send_act)
		} else {
			if evnt.term > sm.currentTerm {
				sm.currentTerm = evnt.term
				sm.votedFor = -1
				sm.currTermLastLogIndex = -1

				stateStore_act := StateStoreAction{sm.currentTerm, sm.votedFor, sm.currTermLastLogIndex}
				act = append(act, stateStore_act)
			}
			sm.leaderId = evnt.senderId

			alarm_act := AlarmAction{random(election_timeout, 2*election_timeout)}
			act = append(act, alarm_act)

			if (evnt.prevLogIndex > len(sm.log)-1) || (sm.log[evnt.prevLogIndex].term != evnt.prevLogTerm) {
				appendEntriesResp_ev := AppendEntriesRespEvent{sm.id, sm.currentTerm, len(sm.log) - 1, false}
				send_act := SendAction{evnt.senderId, appendEntriesResp_ev}
				act = append(act, send_act)
			} else if sm.currTermLastLogIndex != -1 && sm.currTermLastLogIndex >= evnt.prevLogIndex+len(evnt.entries) {
				appendEntriesResp_ev := AppendEntriesRespEvent{sm.id, sm.currentTerm, len(sm.log) - 1, true}
				send_act := SendAction{evnt.senderId, appendEntriesResp_ev}
				act = append(act, send_act)
			} else {
				// how to delete entries in "Log" using logstore.....do we need to?????
				/*for i, log_entry := range sm.log[evnt.prevLogIndex+1 : ] {
					logStore_act := LogStoreAction{evnt.prevLogIndex+1+i, nil}
					act = append(act, logStore_act)
				}*/
				sm.log = sm.log[:evnt.prevLogIndex+1]

				for _, log_entry := range evnt.entries {
					sm.log = append(sm.log, log_entry)
					logStore_act := LogStoreAction{len(sm.log) - 1, log_entry.term, log_entry.data}
					act = append(act, logStore_act)
				}

				sm.currTermLastLogIndex = len(sm.log) - 1
				sm.commitIndex = min(evnt.leaderCommit, len(sm.log)-1)

				appendEntriesResp_ev := AppendEntriesRespEvent{sm.id, sm.currentTerm, len(sm.log) - 1, true}
				send_act := SendAction{evnt.senderId, appendEntriesResp_ev}
				act = append(act, send_act)
			}
		}

	case AppendEntriesRespEvent:
		evnt := ev.(AppendEntriesRespEvent)
		if evnt.term > sm.currentTerm {
			// Don't know whether this case will occur or not.....when it will occur???.....what to do????
			sm.currentTerm = evnt.term
			sm.votedFor = -1
			sm.leaderId = -1
			sm.currTermLastLogIndex = -1

			stateStore_act := StateStoreAction{sm.currentTerm, sm.votedFor, sm.currTermLastLogIndex}
			act = append(act, stateStore_act)
		}

	case VoteReqEvent:
		evnt := ev.(VoteReqEvent)
		if evnt.term < sm.currentTerm {
			voteResp_ev := VoteRespEvent{sm.id, sm.currentTerm, false}
			send_act := SendAction{evnt.senderId, voteResp_ev}
			act = append(act, send_act)
		} else {
			if evnt.term > sm.currentTerm {
				sm.currentTerm = evnt.term
				sm.votedFor = -1
				sm.leaderId = -1
				sm.currTermLastLogIndex = -1

				stateStore_act := StateStoreAction{sm.currentTerm, sm.votedFor, sm.currTermLastLogIndex}
				act = append(act, stateStore_act)
			}

			if (evnt.lastLogTerm < sm.log[len(sm.log)-1].term) || ((evnt.lastLogTerm == sm.log[len(sm.log)-1].term) && (evnt.lastLogIndex < len(sm.log)-1)) {
				voteResp_ev := VoteRespEvent{sm.id, sm.currentTerm, false}
				send_act := SendAction{evnt.senderId, voteResp_ev}
				act = append(act, send_act)
			} else if sm.votedFor != -1 && sm.votedFor != evnt.senderId {
				voteResp_ev := VoteRespEvent{sm.id, sm.currentTerm, false}
				send_act := SendAction{evnt.senderId, voteResp_ev}
				act = append(act, send_act)
			} else {
				sm.votedFor = evnt.senderId
				voteResp_ev := VoteRespEvent{sm.id, sm.currentTerm, true}
				send_act := SendAction{evnt.senderId, voteResp_ev}
				act = append(act, send_act)
			}
		}

	case VoteRespEvent:
		evnt := ev.(VoteRespEvent)
		if evnt.term > sm.currentTerm {
			// Don't know whether this case will occur or not.....when it will occur???.....what to do????
			sm.currentTerm = evnt.term
			sm.votedFor = -1
			sm.leaderId = -1
			sm.currTermLastLogIndex = -1

			stateStore_act := StateStoreAction{sm.currentTerm, sm.votedFor, sm.currTermLastLogIndex}
			act = append(act, stateStore_act)
		}
	}

	return act
}

// -------------------- process event while in candidate state --------------------

func processCandidateEvent(sm *StateMachine, ev Event) []Action {
	var act []Action

	switch ev.(type) {

	case AppendEvent:
		evnt := ev.(AppendEvent)
		commit_act := CommitAction{0, evnt.data, "ERR_TRY_LATER"}
		act = append(act, commit_act)

	case TimeoutEvent:
		//evnt := ev.(TimeoutEvent)
		sm.currentTerm = sm.currentTerm + 1
		sm.votedFor = sm.id
		sm.numOfVotes = 1
		sm.numOfNegVotes = 0
		sm.leaderId = -1
		sm.currTermLastLogIndex = -1

		stateStore_act := StateStoreAction{sm.currentTerm, sm.votedFor, sm.currTermLastLogIndex}
		act = append(act, stateStore_act)

		alarm_act := AlarmAction{random(election_timeout, 2*election_timeout)}
		act = append(act, alarm_act)

		for _, peerId := range sm.peers {
			voteReq_ev := VoteReqEvent{sm.id, sm.currentTerm, sm.id, len(sm.log) - 1, sm.log[len(sm.log)-1].term}
			send_act := SendAction{peerId, voteReq_ev}
			act = append(act, send_act)
		}

	case AppendEntriesReqEvent:
		evnt := ev.(AppendEntriesReqEvent)
		if evnt.term < sm.currentTerm {
			appendEntriesResp_ev := AppendEntriesRespEvent{sm.id, sm.currentTerm, len(sm.log) - 1, false}
			send_act := SendAction{evnt.senderId, appendEntriesResp_ev}
			act = append(act, send_act)
		} else {
			sm.state = "Follower"
			if evnt.term > sm.currentTerm {
				sm.currentTerm = evnt.term
				sm.votedFor = -1
				sm.currTermLastLogIndex = -1

				stateStore_act := StateStoreAction{sm.currentTerm, sm.votedFor, sm.currTermLastLogIndex}
				act = append(act, stateStore_act)
			}
			sm.leaderId = evnt.senderId

			alarm_act := AlarmAction{random(election_timeout, 2*election_timeout)}
			act = append(act, alarm_act)

			if (evnt.prevLogIndex > len(sm.log)-1) || (sm.log[evnt.prevLogIndex].term != evnt.prevLogTerm) {
				appendEntriesResp_ev := AppendEntriesRespEvent{sm.id, sm.currentTerm, len(sm.log) - 1, false}
				send_act := SendAction{evnt.senderId, appendEntriesResp_ev}
				act = append(act, send_act)
			} else if sm.currTermLastLogIndex != -1 && sm.currTermLastLogIndex >= evnt.prevLogIndex+len(evnt.entries) {
				appendEntriesResp_ev := AppendEntriesRespEvent{sm.id, sm.currentTerm, len(sm.log) - 1, true}
				send_act := SendAction{evnt.senderId, appendEntriesResp_ev}
				act = append(act, send_act)
			} else {
				// how to delete entries in "Log" using logstore.....do we need to?????
				/*for i, log_entry := range sm.log[evnt.prevLogIndex+1 : ] {
					logStore_act := LogStoreAction{evnt.prevLogIndex+1+i, nil}
					act = append(act, logStore_act)
				}*/
				sm.log = sm.log[:evnt.prevLogIndex+1]

				for _, log_entry := range evnt.entries {
					sm.log = append(sm.log, log_entry)
					logStore_act := LogStoreAction{len(sm.log) - 1, log_entry.term, log_entry.data}
					act = append(act, logStore_act)
				}

				sm.currTermLastLogIndex = -1
				sm.commitIndex = min(evnt.leaderCommit, len(sm.log)-1)

				appendEntriesResp_ev := AppendEntriesRespEvent{sm.id, sm.currentTerm, len(sm.log) - 1, true}
				send_act := SendAction{evnt.senderId, appendEntriesResp_ev}
				act = append(act, send_act)
			}
		}

	case AppendEntriesRespEvent:
		evnt := ev.(AppendEntriesRespEvent)
		if evnt.term > sm.currentTerm {
			sm.state = "Follower"
			sm.currentTerm = evnt.term
			sm.votedFor = -1
			sm.leaderId = -1
			sm.currTermLastLogIndex = -1

			stateStore_act := StateStoreAction{sm.currentTerm, sm.votedFor, sm.currTermLastLogIndex}
			act = append(act, stateStore_act)
		}

	case VoteReqEvent:
		evnt := ev.(VoteReqEvent)
		if evnt.term <= sm.currentTerm {
			voteResp_ev := VoteRespEvent{sm.id, sm.currentTerm, false}
			send_act := SendAction{evnt.senderId, voteResp_ev}
			act = append(act, send_act)
		} else {
			sm.state = "Follower"
			sm.currentTerm = evnt.term
			sm.votedFor = -1
			sm.leaderId = -1
			sm.currTermLastLogIndex = -1

			stateStore_act := StateStoreAction{sm.currentTerm, sm.votedFor, sm.currTermLastLogIndex}
			act = append(act, stateStore_act)

			if (evnt.lastLogTerm < sm.log[len(sm.log)-1].term) || ((evnt.lastLogTerm == sm.log[len(sm.log)-1].term) && (evnt.lastLogIndex < len(sm.log)-1)) {
				voteResp_ev := VoteRespEvent{sm.id, sm.currentTerm, false}
				send_act := SendAction{evnt.senderId, voteResp_ev}
				act = append(act, send_act)
			} else {
				sm.votedFor = evnt.senderId
				voteResp_ev := VoteRespEvent{sm.id, sm.currentTerm, true}
				send_act := SendAction{evnt.senderId, voteResp_ev}
				act = append(act, send_act)
			}
		}

	case VoteRespEvent:
		evnt := ev.(VoteRespEvent)
		if evnt.term > sm.currentTerm {
			sm.state = "Follower"
			sm.currentTerm = evnt.term
			sm.votedFor = -1
			sm.leaderId = -1
			sm.currTermLastLogIndex = -1

			stateStore_act := StateStoreAction{sm.currentTerm, sm.votedFor, sm.currTermLastLogIndex}
			act = append(act, stateStore_act)
		} else if evnt.term == sm.currentTerm {
			if evnt.voteGranted == true {
				sm.numOfVotes = sm.numOfVotes + 1
				if sm.numOfVotes > (len(sm.peers)+1)/2 {
					sm.state = "Leader"
					sm.leaderId = sm.id

					for i, _ := range sm.peers {
						sm.nextIndex[i] = len(sm.log)
						sm.matchIndex[i] = 0
					}

					alarm_act := AlarmAction{heartbeat_timeout}
					act = append(act, alarm_act)

					for i, peerId := range sm.peers {
						appendEntriesReq_ev := AppendEntriesReqEvent{sm.id, sm.currentTerm, sm.id, sm.nextIndex[i] - 1, sm.log[sm.nextIndex[i]-1].term, nil, sm.commitIndex}
						send_act := SendAction{peerId, appendEntriesReq_ev}
						act = append(act, send_act)
					}
				}
			} else {
				sm.numOfNegVotes = sm.numOfNegVotes + 1
				if sm.numOfVotes > (len(sm.peers)+1)/2 {
					sm.state = "Follower"
					sm.leaderId = -1
				}
			}
		}
	}

	return act
}

// -------------------- processEvent function --------------------

func (sm *StateMachine) processEvent(ev Event) []Action {
	//fmt.Println("In Process Event")
	var act []Action

	switch sm.state {
	case "Leader":
		act = processLeaderEvent(sm, ev)
	case "Follower":
		act = processFollowerEvent(sm, ev)
	case "Candidate":
		act = processCandidateEvent(sm, ev)
	}

	return act
}

// -------------------- main function --------------------

func main() {
	rand.Seed(time.Now().Unix())

	fmt.Println("abc")
}
