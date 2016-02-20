package main

import (
	//"bufio"
	"fmt"
	//"net"
	//"strconv"
	//"strings"
	//"sync"
	"testing"
	"time"
)

func expect(t *testing.T, a string, b string) {
	if a != b {
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", b, a))
	}
}

func testEqualByteSlices(a, b []byte) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func init() {
	time.Sleep(1 * time.Second)
}

//----------------------------------------Follower Testing----------------------------------------
func TestFunc_Follower(t *testing.T) {
	var actions []Action

	//var append_ev AppendEvent
	//var timeout TimeoutEvent
	//var appendEntriesReq_ev AppendEntriesReqEvent
	var appendEntriesResp_ev AppendEntriesRespEvent
	var voteReq_ev VoteReqEvent
	var voteResp_ev VoteRespEvent

	var send_action SendAction
	var commit_action CommitAction
	//var alarm_action AlarmAction
	//var logStore_action LogStoreAction
	//var stateStore_action StateStoreAction
	var ok bool

	var sm StateMachine
	sm.initialise()

	// ---------------------------------------------------------------------
	data := []byte("abc")
	actions = sm.processEvent(AppendEvent{data})
	if len(actions) != 1 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, len(actions)))
	}
	commit_action, ok = actions[0].(CommitAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	if commit_action.err != "ERR_CONTACT_LEADER" {
		t.Error("error code in coomit action NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "ERR_CONTACT_LEADER", commit_action.err))
	}
	if !testEqualByteSlices([]byte(commit_action.data), []byte("abc")) {
		t.Error("data in commit action NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", []byte("abc"), commit_action.data))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(AppendEntriesReqEvent{4, 1, 4, 1, 1, nil, 1})
	if len(actions) != 3 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 3, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	send_action, ok = actions[2].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	appendEntriesResp_ev, ok = send_action.ev.(AppendEntriesRespEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	if appendEntriesResp_ev.success != false {
		t.Error("event in send action incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", false, appendEntriesResp_ev.success))
	}
	if sm.currentTerm != 1 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(AppendEntriesReqEvent{3, 0, 3, 1, 1, nil, 1})
	if len(actions) != 1 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, len(actions)))
	}
	send_action, ok = actions[0].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	appendEntriesResp_ev, ok = send_action.ev.(AppendEntriesRespEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	if appendEntriesResp_ev.success != false {
		t.Error("event in send action incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", false, appendEntriesResp_ev.success))
	}
	if appendEntriesResp_ev.term != 1 {
		t.Error("AppendEntriesResp term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(AppendEntriesReqEvent{4, 1, 4, 0, 0, []LogEntry{LogEntry{1, []byte("abc")}}, 1})
	if len(actions) != 3 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 3, len(actions)))
	}
	_, ok = actions[0].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(LogStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	send_action, ok = actions[2].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	appendEntriesResp_ev, ok = send_action.ev.(AppendEntriesRespEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	if appendEntriesResp_ev.success != true {
		t.Error("event in send action incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", true, appendEntriesResp_ev.success))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(VoteReqEvent{3, 0, 3, 1, 1})
	if len(actions) != 1 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, len(actions)))
	}
	send_action, ok = actions[0].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	voteResp_ev, ok = send_action.ev.(VoteRespEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	if voteResp_ev.voteGranted != false {
		t.Error("event in send action incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", false, voteResp_ev.voteGranted))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(VoteReqEvent{3, 1, 3, 0, 0})
	if len(actions) != 1 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, len(actions)))
	}
	send_action, ok = actions[0].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	voteResp_ev, ok = send_action.ev.(VoteRespEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	if voteResp_ev.voteGranted != false {
		t.Error("event in send action incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", false, voteResp_ev.voteGranted))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(VoteReqEvent{3, 1, 3, 1, 1})
	if len(actions) != 1 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, len(actions)))
	}
	send_action, ok = actions[0].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	voteResp_ev, ok = send_action.ev.(VoteRespEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	if voteResp_ev.voteGranted != true {
		t.Error("event in send action incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", true, voteResp_ev.voteGranted))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(VoteReqEvent{4, 1, 4, 1, 1})
	if len(actions) != 1 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, len(actions)))
	}
	send_action, ok = actions[0].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	voteResp_ev, ok = send_action.ev.(VoteRespEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	if voteResp_ev.voteGranted != false {
		t.Error("event in send action incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", false, voteResp_ev.voteGranted))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(TimeoutEvent{})
	if len(actions) != 6 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 6, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[2+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		voteReq_ev, ok = send_action.ev.(VoteReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if voteReq_ev.candidateId != 0 {
			t.Error("candidate Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, voteReq_ev.candidateId))
		}
	}
	if sm.state != "Candidate" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Candidate", sm.state))
	}
	// ---------------------------------------------------------------------
}

//----------------------------------------Candidate Testing----------------------------------------

func TestFunc_Candidate_1(t *testing.T) {
	var actions []Action

	//var append_ev AppendEvent
	//var timeout TimeoutEvent
	//var appendEntriesReq_ev AppendEntriesReqEvent
	var appendEntriesResp_ev AppendEntriesRespEvent
	var voteReq_ev VoteReqEvent
	//var voteResp_ev VoteRespEvent

	var send_action SendAction
	var commit_action CommitAction
	//var alarm_action AlarmAction
	//var logStore_action LogStoreAction
	//var stateStore_action StateStoreAction
	var ok bool

	var sm StateMachine
	sm.initialise()

	// ---------------------------------------------------------------------
	actions = sm.processEvent(TimeoutEvent{})
	if len(actions) != 6 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 6, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[2+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		voteReq_ev, ok = send_action.ev.(VoteReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if voteReq_ev.candidateId != 0 {
			t.Error("candidate Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, voteReq_ev.candidateId))
		}
	}
	if sm.state != "Candidate" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Candidate", sm.state))
	}
	if sm.currentTerm != 1 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	data := []byte("abc")
	actions = sm.processEvent(AppendEvent{data})
	if len(actions) != 1 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, len(actions)))
	}
	commit_action, ok = actions[0].(CommitAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	if commit_action.err != "ERR_TRY_LATER" {
		t.Error("error code in coomit action NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "ERR_TRY_LATER", commit_action.err))
	}
	if !testEqualByteSlices([]byte(commit_action.data), []byte("abc")) {
		t.Error("data in commit action NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", []byte("abc"), commit_action.data))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(TimeoutEvent{})
	if len(actions) != 6 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 6, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[2+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		voteReq_ev, ok = send_action.ev.(VoteReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if voteReq_ev.candidateId != 0 {
			t.Error("candidate Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, voteReq_ev.candidateId))
		}
	}
	if sm.currentTerm != 2 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 2, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(AppendEntriesReqEvent{3, 1, 3, 1, 1, nil, 1})
	if len(actions) != 1 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, len(actions)))
	}
	send_action, ok = actions[0].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	appendEntriesResp_ev, ok = send_action.ev.(AppendEntriesRespEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	if appendEntriesResp_ev.success != false {
		t.Error("event in send action incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", false, appendEntriesResp_ev.success))
	}
	if appendEntriesResp_ev.term != 2 {
		t.Error("AppendEntriesResp term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 2, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(AppendEntriesReqEvent{4, 2, 4, 0, 0, []LogEntry{LogEntry{1, []byte("abc")}}, 1})
	if len(actions) != 3 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 3, len(actions)))
	}
	_, ok = actions[0].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(LogStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	send_action, ok = actions[2].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	appendEntriesResp_ev, ok = send_action.ev.(AppendEntriesRespEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	if appendEntriesResp_ev.success != true {
		t.Error("event in send action incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", true, appendEntriesResp_ev.success))
	}
	if sm.state != "Follower" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Follower", sm.state))
	}
	// ---------------------------------------------------------------------
}

func TestFunc_Candidate_2(t *testing.T) {
	var actions []Action

	//var append_ev AppendEvent
	//var timeout TimeoutEvent
	//var appendEntriesReq_ev AppendEntriesReqEvent
	//var appendEntriesResp_ev AppendEntriesRespEvent
	var voteReq_ev VoteReqEvent
	var voteResp_ev VoteRespEvent

	var send_action SendAction
	//var commit_action CommitAction
	//var alarm_action AlarmAction
	//var logStore_action LogStoreAction
	//var stateStore_action StateStoreAction
	var ok bool

	var sm StateMachine
	sm.initialise()

	// ---------------------------------------------------------------------
	actions = sm.processEvent(TimeoutEvent{})
	if len(actions) != 6 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 6, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[2+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		voteReq_ev, ok = send_action.ev.(VoteReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if voteReq_ev.candidateId != 0 {
			t.Error("candidate Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, voteReq_ev.candidateId))
		}
	}
	if sm.state != "Candidate" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Candidate", sm.state))
	}
	if sm.currentTerm != 1 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(VoteReqEvent{3, 0, 3, 1, 1})
	if len(actions) != 1 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, len(actions)))
	}
	send_action, ok = actions[0].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	voteResp_ev, ok = send_action.ev.(VoteRespEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	if voteResp_ev.voteGranted != false {
		t.Error("event in send action incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", false, voteResp_ev.voteGranted))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(VoteReqEvent{4, 2, 3, 1, 1})
	if len(actions) != 2 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 2, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	send_action, ok = actions[1].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	voteResp_ev, ok = send_action.ev.(VoteRespEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	if voteResp_ev.voteGranted != true {
		t.Error("event in send action incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", true, voteResp_ev.voteGranted))
	}
	if sm.state != "Follower" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Follower", sm.state))
	}
	if sm.currentTerm != 2 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------
}

func TestFunc_Candidate_3(t *testing.T) {
	var actions []Action

	//var append_ev AppendEvent
	//var timeout TimeoutEvent
	//var appendEntriesReq_ev AppendEntriesReqEvent
	//var appendEntriesResp_ev AppendEntriesRespEvent
	var voteReq_ev VoteReqEvent
	//var voteResp_ev VoteRespEvent

	var send_action SendAction
	//var commit_action CommitAction
	//var alarm_action AlarmAction
	//var logStore_action LogStoreAction
	//var stateStore_action StateStoreAction
	var ok bool

	var sm StateMachine
	sm.initialise()

	// ---------------------------------------------------------------------
	actions = sm.processEvent(TimeoutEvent{})
	if len(actions) != 6 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 6, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[2+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		voteReq_ev, ok = send_action.ev.(VoteReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if voteReq_ev.candidateId != 0 {
			t.Error("candidate Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, voteReq_ev.candidateId))
		}
	}
	if sm.state != "Candidate" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Candidate", sm.state))
	}
	if sm.currentTerm != 1 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(VoteRespEvent{3, 2, false})
	if len(actions) != 1 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	if sm.state != "Follower" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Follower", sm.state))
	}
	if sm.currentTerm != 2 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 2, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

}

func TestFunc_Candidate_4(t *testing.T) {
	var actions []Action

	//var append_ev AppendEvent
	//var timeout TimeoutEvent
	var appendEntriesReq_ev AppendEntriesReqEvent
	//var appendEntriesResp_ev AppendEntriesRespEvent
	var voteReq_ev VoteReqEvent
	//var voteResp_ev VoteRespEvent

	var send_action SendAction
	//var commit_action CommitAction
	//var alarm_action AlarmAction
	//var logStore_action LogStoreAction
	//var stateStore_action StateStoreAction
	var ok bool

	var sm StateMachine
	sm.initialise()

	// ---------------------------------------------------------------------
	actions = sm.processEvent(TimeoutEvent{})
	if len(actions) != 6 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 6, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[2+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		voteReq_ev, ok = send_action.ev.(VoteReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if voteReq_ev.candidateId != 0 {
			t.Error("candidate Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, voteReq_ev.candidateId))
		}
	}
	if sm.state != "Candidate" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Candidate", sm.state))
	}
	if sm.currentTerm != 1 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(VoteRespEvent{3, 1, true})
	if len(actions) != 0 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
	}
	actions = sm.processEvent(VoteRespEvent{4, 1, true})
	if len(actions) != 5 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 5, len(actions)))
	}
	_, ok = actions[0].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[1+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		appendEntriesReq_ev, ok = send_action.ev.(AppendEntriesReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if appendEntriesReq_ev.senderId != 0 {
			t.Error("sender Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
		}
	}
	if sm.state != "Leader" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Leader", sm.state))
	}
	if sm.currentTerm != 1 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

}

//----------------------------------------Leader Testing----------------------------------------

func TestFunc_Leader_1(t *testing.T) {
	var actions []Action

	//var append_ev AppendEvent
	//var timeout TimeoutEvent
	var appendEntriesReq_ev AppendEntriesReqEvent
	var appendEntriesResp_ev AppendEntriesRespEvent
	var voteReq_ev VoteReqEvent
	//var voteResp_ev VoteRespEvent

	var send_action SendAction
	//var commit_action CommitAction
	//var alarm_action AlarmAction
	var logStore_action LogStoreAction
	//var stateStore_action StateStoreAction
	var ok bool

	var sm StateMachine
	sm.initialise()

	// ---------------------------------------------------------------------
	actions = sm.processEvent(TimeoutEvent{})
	if len(actions) != 6 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 6, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[2+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		voteReq_ev, ok = send_action.ev.(VoteReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if voteReq_ev.candidateId != 0 {
			t.Error("candidate Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, voteReq_ev.candidateId))
		}
	}
	if sm.state != "Candidate" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Candidate", sm.state))
	}
	if sm.currentTerm != 1 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(VoteRespEvent{3, 1, true})
	if len(actions) != 0 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
	}
	actions = sm.processEvent(VoteRespEvent{4, 1, true})
	if len(actions) != 1+len(sm.peers) {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1+len(sm.peers), len(actions)))
	}
	_, ok = actions[0].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[1+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		appendEntriesReq_ev, ok = send_action.ev.(AppendEntriesReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if appendEntriesReq_ev.senderId != 0 {
			t.Error("sender Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
		}
	}
	if sm.state != "Leader" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Leader", sm.state))
	}
	if sm.currentTerm != 1 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	data := []byte("abc")
	actions = sm.processEvent(AppendEvent{data})
	if len(actions) != 1+len(sm.peers) {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1+len(sm.peers), len(actions)))
	}
	_, ok = actions[0].(LogStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[1+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		appendEntriesReq_ev, ok = send_action.ev.(AppendEntriesReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if appendEntriesReq_ev.senderId != 0 {
			t.Error("sender Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
		}
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(TimeoutEvent{})
	if len(actions) != 1+len(sm.peers) {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1+len(sm.peers), len(actions)))
	}
	_, ok = actions[0].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[1+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		appendEntriesReq_ev, ok = send_action.ev.(AppendEntriesReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if appendEntriesReq_ev.senderId != 0 {
			t.Error("sender Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
		}
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(AppendEntriesReqEvent{3, 0, 3, 1, 1, nil, 1})
	if len(actions) != 1 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, len(actions)))
	}
	send_action, ok = actions[0].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	appendEntriesResp_ev, ok = send_action.ev.(AppendEntriesRespEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	if appendEntriesResp_ev.success != false {
		t.Error("event in send action incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", false, appendEntriesResp_ev.success))
	}
	if appendEntriesResp_ev.term != 1 {
		t.Error("AppendEntriesResp term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(AppendEntriesReqEvent{4, 2, 4, 1, 2, nil, 1})
	if len(actions) != 3 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 3, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	send_action, ok = actions[2].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	appendEntriesResp_ev, ok = send_action.ev.(AppendEntriesRespEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	if appendEntriesResp_ev.success != false {
		t.Error("event in send action incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", false, appendEntriesResp_ev.success))
	}
	if appendEntriesResp_ev.term != 2 {
		t.Error("AppendEntriesResp term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 2, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(AppendEntriesReqEvent{4, 3, 4, 0, 0, []LogEntry{LogEntry{1, []byte("abc")}, LogEntry{2, []byte("xyz")}}, 1})
	if len(actions) != 5 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 5, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	logStore_action, ok = actions[2].(LogStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	if !testEqualByteSlices(logStore_action.data, []byte("abc")) {
		t.Error("data in logStore action NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", []byte("abc"), logStore_action.data))
	}
	logStore_action, ok = actions[3].(LogStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	if !testEqualByteSlices(logStore_action.data, []byte("xyz")) {
		t.Error("data in logStore action NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", []byte("xyz"), logStore_action.data))
	}
	send_action, ok = actions[4].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	appendEntriesResp_ev, ok = send_action.ev.(AppendEntriesRespEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	if appendEntriesResp_ev.success != true {
		t.Error("event in send action incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", false, appendEntriesResp_ev.success))
	}
	if appendEntriesResp_ev.term != 3 {
		t.Error("AppendEntriesResp term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 3, sm.currentTerm))
	}
	if sm.state != "Follower" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Follower", sm.state))
	}
	// ---------------------------------------------------------------------
}

func TestFunc_Leader_2(t *testing.T) {
	var actions []Action

	//var append_ev AppendEvent
	//var timeout TimeoutEvent
	var appendEntriesReq_ev AppendEntriesReqEvent
	//var appendEntriesResp_ev AppendEntriesRespEvent
	var voteReq_ev VoteReqEvent
	//var voteResp_ev VoteRespEvent

	var send_action SendAction
	//var commit_action CommitAction
	//var alarm_action AlarmAction
	//var logStore_action LogStoreAction
	//var stateStore_action StateStoreAction
	var ok bool

	var sm StateMachine
	sm.initialise()

	// ---------------------------------------------------------------------
	actions = sm.processEvent(TimeoutEvent{})
	if len(actions) != 6 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 6, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[2+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		voteReq_ev, ok = send_action.ev.(VoteReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if voteReq_ev.candidateId != 0 {
			t.Error("candidate Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, voteReq_ev.candidateId))
		}
	}
	if sm.state != "Candidate" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Candidate", sm.state))
	}
	if sm.currentTerm != 1 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(VoteRespEvent{3, 1, true})
	if len(actions) != 0 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
	}
	actions = sm.processEvent(VoteRespEvent{4, 1, true})
	if len(actions) != 1+len(sm.peers) {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1+len(sm.peers), len(actions)))
	}
	_, ok = actions[0].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[1+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		appendEntriesReq_ev, ok = send_action.ev.(AppendEntriesReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if appendEntriesReq_ev.senderId != 0 {
			t.Error("sender Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
		}
	}
	if sm.state != "Leader" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Leader", sm.state))
	}
	if sm.currentTerm != 1 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(AppendEntriesRespEvent{3, 2, 1, false})
	if len(actions) != 1 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	if sm.state != "Follower" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Follower", sm.state))
	}
	if sm.currentTerm != 2 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 2, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

}

func TestFunc_Leader_3(t *testing.T) {
	var actions []Action

	//var append_ev AppendEvent
	//var timeout TimeoutEvent
	var appendEntriesReq_ev AppendEntriesReqEvent
	//var appendEntriesResp_ev AppendEntriesRespEvent
	var voteReq_ev VoteReqEvent
	//var voteResp_ev VoteRespEvent

	var send_action SendAction
	//var commit_action CommitAction
	//var alarm_action AlarmAction
	//var logStore_action LogStoreAction
	//var stateStore_action StateStoreAction
	var ok bool

	var sm StateMachine
	sm.initialise()

	// ---------------------------------------------------------------------
	actions = sm.processEvent(TimeoutEvent{})
	if len(actions) != 6 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 6, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[2+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		voteReq_ev, ok = send_action.ev.(VoteReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if voteReq_ev.candidateId != 0 {
			t.Error("candidate Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, voteReq_ev.candidateId))
		}
	}
	if sm.state != "Candidate" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Candidate", sm.state))
	}
	if sm.currentTerm != 1 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(VoteRespEvent{3, 1, true})
	if len(actions) != 0 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
	}
	actions = sm.processEvent(VoteRespEvent{4, 1, true})
	if len(actions) != 1+len(sm.peers) {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1+len(sm.peers), len(actions)))
	}
	_, ok = actions[0].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[1+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		appendEntriesReq_ev, ok = send_action.ev.(AppendEntriesReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if appendEntriesReq_ev.senderId != 0 {
			t.Error("sender Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
		}
	}
	if sm.state != "Leader" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Leader", sm.state))
	}
	if sm.currentTerm != 1 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	data := []byte("abc")
	actions = sm.processEvent(AppendEvent{data})
	if len(actions) != 1+len(sm.peers) {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1+len(sm.peers), len(actions)))
	}
	_, ok = actions[0].(LogStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[1+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		appendEntriesReq_ev, ok = send_action.ev.(AppendEntriesReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if appendEntriesReq_ev.senderId != 0 {
			t.Error("sender Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
		}
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(AppendEntriesRespEvent{3, 1, 1, false})
	if len(actions) != 1 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, len(actions)))
	}
	send_action, ok = actions[0].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	appendEntriesReq_ev, ok = send_action.ev.(AppendEntriesReqEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	if appendEntriesReq_ev.senderId != 0 {
		t.Error("sender Id is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
	}
	// ---------------------------------------------------------------------

}

func TestFunc_Leader_4(t *testing.T) {
	var actions []Action

	//var append_ev AppendEvent
	//var timeout TimeoutEvent
	var appendEntriesReq_ev AppendEntriesReqEvent
	//var appendEntriesResp_ev AppendEntriesRespEvent
	var voteReq_ev VoteReqEvent
	var voteResp_ev VoteRespEvent

	var send_action SendAction
	//var commit_action CommitAction
	//var alarm_action AlarmAction
	//var logStore_action LogStoreAction
	//var stateStore_action StateStoreAction
	var ok bool

	var sm StateMachine
	sm.initialise()

	// ---------------------------------------------------------------------
	actions = sm.processEvent(TimeoutEvent{})
	if len(actions) != 6 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 6, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[2+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		voteReq_ev, ok = send_action.ev.(VoteReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if voteReq_ev.candidateId != 0 {
			t.Error("candidate Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, voteReq_ev.candidateId))
		}
	}
	if sm.state != "Candidate" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Candidate", sm.state))
	}
	if sm.currentTerm != 1 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(VoteRespEvent{3, 1, true})
	if len(actions) != 0 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
	}
	actions = sm.processEvent(VoteRespEvent{4, 1, true})
	if len(actions) != 1+len(sm.peers) {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1+len(sm.peers), len(actions)))
	}
	_, ok = actions[0].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[1+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		appendEntriesReq_ev, ok = send_action.ev.(AppendEntriesReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if appendEntriesReq_ev.senderId != 0 {
			t.Error("sender Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
		}
	}
	if sm.state != "Leader" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Leader", sm.state))
	}
	if sm.currentTerm != 1 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	data := []byte("abc")
	actions = sm.processEvent(AppendEvent{data})
	if len(actions) != 1+len(sm.peers) {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1+len(sm.peers), len(actions)))
	}
	_, ok = actions[0].(LogStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[1+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		appendEntriesReq_ev, ok = send_action.ev.(AppendEntriesReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if appendEntriesReq_ev.senderId != 0 {
			t.Error("sender Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
		}
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	data = []byte("xyz")
	actions = sm.processEvent(AppendEvent{data})
	if len(actions) != 1+len(sm.peers) {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1+len(sm.peers), len(actions)))
	}
	_, ok = actions[0].(LogStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[1+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		appendEntriesReq_ev, ok = send_action.ev.(AppendEntriesReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if appendEntriesReq_ev.senderId != 0 {
			t.Error("sender Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
		}
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(AppendEntriesRespEvent{3, 1, 1, true})
	if len(actions) != 1 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, len(actions)))
	}
	send_action, ok = actions[0].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = send_action.ev.(AppendEntriesReqEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	actions = sm.processEvent(AppendEntriesRespEvent{4, 1, 1, true})
	if len(actions) != 1 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, len(actions)))
	}
	send_action, ok = actions[0].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = send_action.ev.(AppendEntriesReqEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	if sm.commitIndex != 1 {
		t.Error("state machine commit index is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.commitIndex))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(VoteReqEvent{2, 1, 2, 1, 1})
	if len(actions) != 1 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, len(actions)))
	}
	send_action, ok = actions[0].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	voteResp_ev, ok = send_action.ev.(VoteRespEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	if voteResp_ev.voteGranted != false {
		t.Error("voteGranted in voteResponce is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", false, voteResp_ev.voteGranted))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(VoteReqEvent{1, 2, 1, 1, 1})
	if len(actions) != 3 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 3, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	send_action, ok = actions[2].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	voteResp_ev, ok = send_action.ev.(VoteRespEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	if voteResp_ev.voteGranted != false {
		t.Error("voteGranted in voteResponce is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", false, voteResp_ev.voteGranted))
	}
	if sm.state != "Follower" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Follower", sm.state))
	}
	if sm.currentTerm != 2 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 2, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

}

func TestFunc_Leader_5(t *testing.T) {
	var actions []Action

	//var append_ev AppendEvent
	//var timeout TimeoutEvent
	var appendEntriesReq_ev AppendEntriesReqEvent
	//var appendEntriesResp_ev AppendEntriesRespEvent
	var voteReq_ev VoteReqEvent
	var voteResp_ev VoteRespEvent

	var send_action SendAction
	//var commit_action CommitAction
	//var alarm_action AlarmAction
	//var logStore_action LogStoreAction
	//var stateStore_action StateStoreAction
	var ok bool

	var sm StateMachine
	sm.initialise()

	// ---------------------------------------------------------------------
	actions = sm.processEvent(TimeoutEvent{})
	if len(actions) != 6 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 6, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[2+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		voteReq_ev, ok = send_action.ev.(VoteReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if voteReq_ev.candidateId != 0 {
			t.Error("candidate Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, voteReq_ev.candidateId))
		}
	}
	if sm.state != "Candidate" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Candidate", sm.state))
	}
	if sm.currentTerm != 1 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(VoteRespEvent{3, 1, true})
	if len(actions) != 0 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
	}
	actions = sm.processEvent(VoteRespEvent{4, 1, true})
	if len(actions) != 1+len(sm.peers) {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1+len(sm.peers), len(actions)))
	}
	_, ok = actions[0].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[1+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		appendEntriesReq_ev, ok = send_action.ev.(AppendEntriesReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if appendEntriesReq_ev.senderId != 0 {
			t.Error("sender Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
		}
	}
	if sm.state != "Leader" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Leader", sm.state))
	}
	if sm.currentTerm != 1 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(VoteReqEvent{1, 2, 1, 0, 0})
	if len(actions) != 3 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 3, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	send_action, ok = actions[2].(SendAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	voteResp_ev, ok = send_action.ev.(VoteRespEvent)
	if ok == false {
		t.Error("Type of event NOT matching")
	}
	if voteResp_ev.voteGranted != true {
		t.Error("voteGranted in voteResponce is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", true, voteResp_ev.voteGranted))
	}
	if sm.state != "Follower" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Follower", sm.state))
	}
	if sm.currentTerm != 2 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 2, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

}

func TestFunc_Leader_6(t *testing.T) {
	var actions []Action

	//var append_ev AppendEvent
	//var timeout TimeoutEvent
	var appendEntriesReq_ev AppendEntriesReqEvent
	//var appendEntriesResp_ev AppendEntriesRespEvent
	var voteReq_ev VoteReqEvent
	//var voteResp_ev VoteRespEvent

	var send_action SendAction
	//var commit_action CommitAction
	//var alarm_action AlarmAction
	//var logStore_action LogStoreAction
	//var stateStore_action StateStoreAction
	var ok bool

	var sm StateMachine
	sm.initialise()

	// ---------------------------------------------------------------------
	actions = sm.processEvent(TimeoutEvent{})
	if len(actions) != 6 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 6, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[2+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		voteReq_ev, ok = send_action.ev.(VoteReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if voteReq_ev.candidateId != 0 {
			t.Error("candidate Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, voteReq_ev.candidateId))
		}
	}
	if sm.state != "Candidate" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Candidate", sm.state))
	}
	if sm.currentTerm != 1 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(VoteRespEvent{3, 1, true})
	if len(actions) != 0 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
	}
	actions = sm.processEvent(VoteRespEvent{4, 1, true})
	if len(actions) != 1+len(sm.peers) {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1+len(sm.peers), len(actions)))
	}
	_, ok = actions[0].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	for i, _ := range sm.peers {
		send_action, ok = actions[1+i].(SendAction)
		if ok == false {
			t.Error("Type of actions NOT matching")
		}
		appendEntriesReq_ev, ok = send_action.ev.(AppendEntriesReqEvent)
		if ok == false {
			t.Error("Type of event NOT matching")
		}
		if appendEntriesReq_ev.senderId != 0 {
			t.Error("sender Id is incorrect")
			t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
		}
	}
	if sm.state != "Leader" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Leader", sm.state))
	}
	if sm.currentTerm != 1 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 1, sm.currentTerm))
	}
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	actions = sm.processEvent(VoteRespEvent{3, 2, false})
	if len(actions) != 2 {
		t.Error("number of actions NOT matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 0, len(actions)))
	}
	_, ok = actions[0].(StateStoreAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	_, ok = actions[1].(AlarmAction)
	if ok == false {
		t.Error("Type of actions NOT matching")
	}
	if sm.state != "Follower" {
		t.Error("state is incorrect")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", "Follower", sm.state))
	}
	if sm.currentTerm != 2 {
		t.Error("state machine term not matching")
		t.Error(fmt.Sprintf("Expected - %v, Found - %v", 2, sm.currentTerm))
	}
	// ---------------------------------------------------------------------
}
