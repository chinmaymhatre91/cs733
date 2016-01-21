package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	cnt1 int
	cnt2 int
	mux  sync.Mutex
)

func TestTCPSimple(t *testing.T) {
	go serverMain()
	time.Sleep(1 * time.Second)
	name := "hi.txt"
	contents := "bye"
	exptime := 300000
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Error(err.Error())
	}

	scanner := bufio.NewScanner(conn)

	// Write a file
	fmt.Fprintf(conn, "write %v %v %v\r\n%v\r\n", name, len(contents), exptime, contents)
	scanner.Scan()
	resp := scanner.Text()
	arr := strings.Split(resp, " ")
	expect(t, arr[0], "OK")
	version, err := strconv.ParseInt(arr[1], 10, 64)
	if err != nil {
		t.Error("Non-numeric version found")
	}

	fmt.Fprintf(conn, "read %v\r\n", name)
	scanner.Scan()
	arr = strings.Split(scanner.Text(), " ")
	expect(t, arr[0], "CONTENTS")
	expect(t, arr[1], fmt.Sprintf("%v", version))
	expect(t, arr[2], fmt.Sprintf("%v", len(contents)))
	scanner.Scan()
	expect(t, contents, scanner.Text())

	for i := 0; i < 100; i++ {
		go test_write(t, name, exptime, contents)
	}

	time.Sleep(2 * time.Second)

	fmt.Fprintf(conn, "read %v\r\n", name)
	scanner.Scan()
	arr = strings.Split(scanner.Text(), " ")
	expect(t, arr[0], "CONTENTS")
	if arr[1] != fmt.Sprintf("%v", version+100) {
		t.Error("write operation error")
	}
	expect(t, arr[2], fmt.Sprintf("%v", len(contents)))
	scanner.Scan()
	expect(t, contents, scanner.Text())

}

func test_write(t *testing.T, name string, exptime int, contents string) {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Error(err.Error())
	}

	scanner := bufio.NewScanner(conn)

	fmt.Fprintf(conn, "write %v %v %v\r\n%v\r\n", name, len(contents), exptime, contents)
	scanner.Scan()
	resp := scanner.Text()
	arr := strings.Split(resp, " ")
	expect(t, arr[0], "OK")
	_, err = strconv.ParseInt(arr[1], 10, 64)
	if err != nil {
		t.Error("Non-numeric version found")
	}
}

func expect(t *testing.T, a string, b string) {
	if a != b {
		t.Error(fmt.Sprintf("Expected %v, found %v", b, a))
	}
}
