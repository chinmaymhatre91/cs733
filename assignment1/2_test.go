package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

var cnt1, cnt2 int

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

	cnt1 = 0
	cnt2 = 0
	for i := 0; i < 10; i++ {
		go test_cas(t, conn, scanner, name, version, exptime, contents)
	}

	//if cnt1 != 1 || cnt2 != 9 {
	//	t.Error(fmt.Sprintf("invalid cas operation cnt1-%v, cnt2-%v", cnt1, cnt2))
	//}

}

func test_cas(t *testing.T, conn net.Conn, scanner *bufio.Scanner, name string, version int64, exptime int, contents string) {
	fmt.Fprintf(conn, "cas %v %v %v %v\r\n%v\r\n", name, version, len(contents), exptime, contents)
	scanner.Scan()
	resp := scanner.Text()
	arr := strings.Split(resp, " ")
	if arr[0] == "OK" {
		ver, err := strconv.ParseInt(arr[1], 10, 64)
		if err != nil {
			t.Error("Non-numeric version found")
		}
		if ver != version+1 {
			t.Error("cas version error")
		}
		cnt1 = cnt1 + 1
	} else {
		cnt2 = cnt2 + 1
	}
}

func expect(t *testing.T, a string, b string) {
	if a != b {
		t.Error(fmt.Sprintf("Expected %v, found %v", b, a))
	}
}
