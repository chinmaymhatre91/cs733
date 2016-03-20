package main

import (
	"fmt"
	"testing"
	"time"
)

func TestBasic(t *testing.T) {
	rafts := makeRafts() // array of []raft.Node
	time.Sleep(time.Second * 2)
	ldr, err := getLeader(rafts)
	if err != nil {
		fmt.Println(err)
		return
	}
	ldr.Append([]byte("foo"))
	time.Sleep(time.Second * 3)

	for _, node := range rafts {
		ch, _ := node.CommitChannel()
		select {
		case ci := <-ch:
			if ci.Err != nil {
				t.Fatal(ci.Err)
			}
			if string(ci.Data) != "foo" {
				t.Fatal("Got different data")
			}
		default:
			t.Fatal("Expected message on all nodes")
		}
	}
}
