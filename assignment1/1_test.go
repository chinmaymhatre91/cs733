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

	fmt.Fprintf(conn, "read random\r\n")
	scanner.Scan()
	resp = scanner.Text()
	expect(t, resp, "ERR_FILE_NOT_FOUND")

	fmt.Fprintf(conn, "write %v %v 2\r\n%v\r\n", name, len(contents), contents)
	scanner.Scan()
	resp = scanner.Text()
	arr = strings.Split(resp, " ")
	expect(t, arr[0], "OK")
	version, err = strconv.ParseInt(arr[1], 10, 64)
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

	time.Sleep(3 * time.Second)

	fmt.Fprintf(conn, "read %v\r\n", name)
	scanner.Scan()
	resp = scanner.Text()
	expect(t, resp, "ERR_FILE_NOT_FOUND")

	fmt.Fprintf(conn, "write %v\r\n", name)
	scanner.Scan()
	resp = scanner.Text()
	expect(t, resp, "ERR_CMD_ERR")
}

func expect(t *testing.T, a string, b string) {
	if a != b {
		t.Error(fmt.Sprintf("Expected %v, found %v", b, a))
	}
}
