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

func expect(t *testing.T, a string, b string) {
	if a != b {
		t.Error(fmt.Sprintf("Expected %v, found %v", b, a))
	}
}

func init() {
	go serverMain()
	time.Sleep(1 * time.Second)
}

//-------------------------------------------------------------------------------------------------------
//---------------------------------------------------1---------------------------------------------------
//-------------------------------------------------------------------------------------------------------

func TestMessages(t *testing.T) {
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

//-------------------------------------------------------------------------------------------------------
//---------------------------------------------------2---------------------------------------------------
//-------------------------------------------------------------------------------------------------------

var (
	cnt1 int
	cnt2 int
	mux  sync.Mutex
)

func TestCAS(t *testing.T) {
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
		go test_cas(t, name, version, exptime, contents)
	}

	time.Sleep(2 * time.Second)

	if cnt1 != 1 && cnt2 != 9 {
		t.Error("cas operation error")
	}

	fmt.Fprintf(conn, "read %v\r\n", name)
	scanner.Scan()
	arr = strings.Split(scanner.Text(), " ")
	expect(t, arr[0], "CONTENTS")
	expect(t, arr[1], fmt.Sprintf("%v", version+1))
	expect(t, arr[2], fmt.Sprintf("%v", len(contents)))
	scanner.Scan()
	expect(t, contents, scanner.Text())

}

func test_cas(t *testing.T, name string, version int64, exptime int, contents string) {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Error(err.Error())
	}

	scanner := bufio.NewScanner(conn)

	fmt.Fprintf(conn, "cas %v %v %v %v\r\n%v\r\n", name, version, len(contents), exptime, contents)
	scanner.Scan()
	resp := scanner.Text()
	//t.Error(string(resp))
	arr := strings.Split(resp, " ")
	if arr[0] == "OK" {
		ver, err := strconv.ParseInt(arr[1], 10, 64)
		if err != nil {
			t.Error("Non-numeric version found")
		}
		if ver != version+1 {
			t.Error("cas version error")
		}
		mux.Lock()
		cnt1 = cnt1 + 1
		mux.Unlock()
	} else {
		mux.Lock()
		cnt2 = cnt2 + 1
		mux.Unlock()
	}

}

//-------------------------------------------------------------------------------------------------------
//---------------------------------------------------3---------------------------------------------------
//-------------------------------------------------------------------------------------------------------

func TestWrite(t *testing.T) {
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

//-------------------------------------------------------------------------------------------------------
//---------------------------------------------------4---------------------------------------------------
//-------------------------------------------------------------------------------------------------------

func TestConcurrentCAS(t *testing.T) {
	name := "abc"
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

	contents = "abcdefghij"

	for _, r := range contents {
		go test_concurrent_cas(t, name, version, exptime, string(r))
		//contents++
	}

	time.Sleep(2 * time.Second)

	fmt.Fprintf(conn, "read %v\r\n", name)
	scanner.Scan()
	arr = strings.Split(scanner.Text(), " ")
	expect(t, arr[0], "CONTENTS")
	expect(t, arr[1], fmt.Sprintf("%v", version+10))
	scanner.Scan()
	if strings.Contains(scanner.Text(), "10") != true {
		t.Error("error in concurrent cas operation working")
	}

}

func test_concurrent_cas(t *testing.T, name string, version int64, exptime int, contents string) {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Error(err.Error())
	}

	scanner := bufio.NewScanner(conn)

	for i := 1; i <= 10; i++ {
		tmp_contents := contents + strconv.Itoa(i)
		fmt.Fprintf(conn, "cas %v %v %v %v\r\n%v\r\n", name, version, len(tmp_contents), exptime, tmp_contents)
		version++
		scanner.Scan()
		resp := scanner.Text()
		//t.Error(string(resp))
		arr := strings.Split(resp, " ")
		if arr[0] == "OK" {
			_, err := strconv.ParseInt(arr[1], 10, 64)
			if err != nil {
				t.Error("Non-numeric version found")
			}
		} else if arr[0] != "ERR_VERSION" {
			t.Error(arr[0])
		}
	}
}
