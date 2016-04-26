package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"os"
	"os/exec"
	"testing"
	"time"
	"sync"
)

/*func TestRPCMain(t *testing.T) {
	go serverMain()
	time.Sleep(1 * time.Second)
}*/

func execCommand(cmd string) (error) {
	proc := exec.Command("sh", "-c", cmd)
    proc.Stdout = os.Stdout
    proc.Stdin = os.Stdin
    err := proc.Run()

    return err
}

func execCommandBg(cmd string) (error) {
	proc := exec.Command("sh", "-c", cmd)
    proc.Stdout = os.Stdout
    proc.Stdin = os.Stdin
    err := proc.Start()

    return err
}

func beginTest() {
	killCluster()
	err := execCommand("./clean.sh")
	if err != nil {
		panic("[Error] while running clean.sh")
	}
	err = execCommand("go build")
	if err != nil {
		panic("[Error] while go build")
	}
}

func endTest() {
	killCluster()
	err := execCommand("./clean.sh")
	if err != nil {
		panic("[Error] while running clean.sh")
	}
}

func startServer(server_id int, clust_no int) {
	err := execCommandBg("./assignment4 "+ strconv.Itoa(server_id) + " " + strconv.Itoa(clust_no))
	if err != nil {
		panic("[Error] while starting server - "+strconv.Itoa(server_id)+" - "+err.Error())
	}	
}

func startCluster(clust_no int) {
	/*for _, entry := range my_file_clust {
		//fmt.Println("Starting - ",entry.Id)
		startServer(entry.Id)
	}*/

	procs := make(map[int]*exec.Cmd)
	for i, entry := range my_file_clust {
		procs[i] = exec.Command("sh", "-c", "./assignment4 "+ strconv.Itoa(entry.Id) + " " + strconv.Itoa(clust_no))
		procs[i].Stdout = os.Stdout
		procs[i].Stderr = os.Stderr
		err := procs[i].Start()
		if err != nil {
			panic("[Error] while starting server - "+strconv.Itoa(entry.Id)+" - "+err.Error())		
		}
	}
}

func killServer(server_id int) {
	err := execCommand("./killServer.sh \"assignment "+strconv.Itoa(server_id)+"\"")
	if err != nil {
		panic("[Error] while killing server - "+strconv.Itoa(server_id)+" - "+err.Error())
	}	
}

func killCluster() {
	for _, entry := range my_file_clust {
		killServer(entry.Id)
	}	
}

func expect(t *testing.T, response *Msg, expected *Msg, errstr string, err error) {
	if err != nil {
		t.Fatal("Unexpected error: " + err.Error())
	}
	ok := true
	if response.Kind != expected.Kind {
		ok = false
		errstr += fmt.Sprintf(" Got kind='%c', expected '%c'", response.Kind, expected.Kind)
	}
	if expected.Version > 0 && expected.Version != response.Version {
		ok = false
		errstr += " Version mismatch"
	}
	if response.Kind == 'C' {
		if expected.Contents != nil &&
			bytes.Compare(response.Contents, expected.Contents) != 0 {
			ok = false
		}
	}
	if !ok {
		t.Fatal("Expected " + errstr)
	}
}

///*
func TestBasic(t *testing.T) {
	beginTest()
	startCluster(0)

	time.Sleep(time.Second*5)

	cl := mkClient(t, "127.0.0.1:9001")
	defer cl.close()

	m, err := cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)

	txt := "foo"
	m, err = cl.write("cs733net", txt, 0)
	expect(t, m, &Msg{Kind: 'O'}, "write success", err)


	killCluster()
	endTest()
	fmt.Println("Test End - TestBasic")
}
//*/

///*
func TestRPC_BasicSequential(t *testing.T) {
	beginTest()
	startCluster(0)

	time.Sleep(time.Second*5)

	cl := mkClient(t, "127.0.0.1:9001")
	defer cl.close()

	// Read non-existent file cs733net
	m, err := cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)

	// Read non-existent file cs733net
	m, err = cl.delete("cs733net")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)

	// Write file cs733net
	data := "Cloud fun"
	m, err = cl.write("cs733net", data, 0)
	expect(t, m, &Msg{Kind: 'O'}, "write success", err)

	// Expect to read it back
	m, err = cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(data)}, "read my write", err)

	// CAS in new value
	version1 := m.Version
	data2 := "Cloud fun 2"
	// Cas new value
	m, err = cl.cas("cs733net", version1, data2, 0)
	expect(t, m, &Msg{Kind: 'O'}, "cas success", err)

	// Expect to read it back
	m, err = cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(data2)}, "read my cas", err)

	// Expect Cas to fail with old version
	m, err = cl.cas("cs733net", version1, data, 0)
	expect(t, m, &Msg{Kind: 'V'}, "cas version mismatch", err)

	// Expect a failed cas to not have succeeded. Read should return data2.
	m, err = cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(data2)}, "failed cas to not have succeeded", err)

	// delete
	m, err = cl.delete("cs733net")
	expect(t, m, &Msg{Kind: 'O'}, "delete success", err)

	// Expect to not find the file
	m, err = cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)

	killCluster()
	endTest()
	fmt.Println("Test End - TestRPC_BasicSequential")
}
//*/

///*
func TestRPC_Binary(t *testing.T) {
	beginTest()
	startCluster(0)

	time.Sleep(time.Second*5)

	cl := mkClient(t, "127.0.0.1:9001")
	defer cl.close()

	// Write binary contents
	data := "\x00\x01\r\n\x03" // some non-ascii, some crlf chars
	m, err := cl.write("binfile", data, 0)
	expect(t, m, &Msg{Kind: 'O'}, "write success", err)

	// Expect to read it back
	m, err = cl.read("binfile")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(data)}, "read my write", err)

	killCluster()
	endTest()
	fmt.Println("Test End - TestRPC_Binary")
}
//*/

///*
func TestRPC_Chunks(t *testing.T) {
	beginTest()
	startCluster(0)

	time.Sleep(time.Second*5)

	cl := mkClient(t, "127.0.0.1:9001")
	defer cl.close()

	var err error
	snd := func(chunk string) {
		if err == nil {
			err = cl.send(chunk)
		}
	}

	var m *Msg

	for {
		// Send the command "write teststream 10\r\nabcdefghij\r\n" in multiple chunks
		// Nagle's algorithm is disabled on a write, so the server should get these in separate TCP packets.
		snd("wr")
		time.Sleep(10 * time.Millisecond)
		snd("ite test")
		time.Sleep(10 * time.Millisecond)
		snd("stream 1")
		time.Sleep(10 * time.Millisecond)
		snd("0\r\nabcdefghij\r")
		time.Sleep(10 * time.Millisecond)
		snd("\n")
		m, err = cl.rcv()
		if err == nil {
			res, err := cl.checkResp(m)
			if (res == "T_KIND_MSG" || res == "R_KIND_MSG" ) && (err == nil) {
				continue
			}
			break
		}
	}
	expect(t, m, &Msg{Kind: 'O'}, "writing in chunks should work", err)

	killCluster()
	endTest()
	fmt.Println("Test End - TestRPC_Chunks")
}
//*/

///*
func TestRPC_BasicTimer(t *testing.T) {
	beginTest()
	startCluster(0)

	time.Sleep(time.Second*5)

	cl := mkClient(t, "127.0.0.1:9001")
	defer cl.close()

	// Write file cs733, with expiry time of 2 seconds
	str := "Cloud fun"
	m, err := cl.write("cs733", str, 2)
	expect(t, m, &Msg{Kind: 'O'}, "write success", err)

	// Expect to read it back immediately.
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(str)}, "read my cas", err)

	time.Sleep(3 * time.Second)

	// Expect to not find the file after expiry
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)

	// Recreate the file with expiry time of 1 second
	m, err = cl.write("cs733", str, 1)
	expect(t, m, &Msg{Kind: 'O'}, "file recreated", err)

	// Overwrite the file with expiry time of 4. This should be the new time.
	m, err = cl.write("cs733", str, 3)
	expect(t, m, &Msg{Kind: 'O'}, "file overwriten with exptime=4", err)

	// The last expiry time was 3 seconds. We should expect the file to still be around 2 seconds later
	time.Sleep(2 * time.Second)

	// Expect the file to not have expired.
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(str)}, "file to not expire until 4 sec", err)

	time.Sleep(3 * time.Second)
	// 5 seconds since the last write. Expect the file to have expired
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'F'}, "file not found after 4 sec", err)

	// Create the file with an expiry time of 1 sec. We're going to delete it
	// then immediately create it. The new file better not get deleted. 
	m, err = cl.write("cs733", str, 1)
	expect(t, m, &Msg{Kind: 'O'}, "file created for delete", err)

	m, err = cl.delete("cs733")
	expect(t, m, &Msg{Kind: 'O'}, "deleted ok", err)

	m, err = cl.write("cs733", str, 0) // No expiry
	expect(t, m, &Msg{Kind: 'O'}, "file recreated", err)

	time.Sleep(1100 * time.Millisecond) // A little more than 1 sec
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'C'}, "file should not be deleted", err)

	killCluster()
	endTest()
	fmt.Println("Test End - TestRPC_BasicTimer")
}
//*/

func getServerId(cl *Client) (int) {
	fields := strings.Split(cl.addr, ":")
	port, err := strconv.Atoi(fields[1])
	if err != nil {
		panic("[Error] Converting string integer while getting Id")
	}
	id := 100 + (port - 9000)
	return id
}

///*
func TestLeaderFailure(t *testing.T) {
	beginTest()
	startCluster(0)

	time.Sleep(time.Second*5)

	cl := mkClient(t, "127.0.0.1:9001")
	defer cl.close()

	m, err := cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)

	txt := "foo1"
	m, err = cl.write("cs733net", txt, 0)
	expect(t, m, &Msg{Kind: 'O'}, "write success", err)

	ldr_id := getServerId(cl)

	killServer(ldr_id)

	time.Sleep(time.Second * 5)

	new_ldr_id := ldr_id + 1
	if new_ldr_id > 105 {
		new_ldr_id = 101
	}
	new_ldr_port := 9000 + (new_ldr_id-100)

	cl = mkClient(t, "127.0.0.1:"+strconv.Itoa(new_ldr_port))

	txt = "foo2"
	m, err = cl.write("cs733net", txt, 0)
	expect(t, m, &Msg{Kind: 'O'}, "write success", err)

	startServer(ldr_id, 0)

	time.Sleep(time.Second * 5)

	cl = mkClient(t, "127.0.0.1:"+strconv.Itoa(9000 + (ldr_id-100)))

	m, err = cl.read("cs733net")
	if err != nil {
		t.Fatal("Unexpected error " + err.Error())
	}
	if !(string(m.Contents[:]) == txt && m.Version == 2) {
		t.Fatal("Expected msg with contents - foo2 with version 2. Received - " + string(m.Contents[:]) + strconv.Itoa(m.Version))
	}

	killCluster()
	endTest()
	fmt.Println("Test End - TestLeaderFailure")
}
//*/

// nclients write to the same file. At the end the file should be
// any one clients' last write

///*
func TestRPC_ConcurrentWrites(t *testing.T) {
	beginTest()
	startCluster(1)

	time.Sleep(time.Second*5)

	nclients := 5
	niters := 5
	clients := make([]*Client, nclients)
	for i := 0; i < nclients; i++ {
		cl := mkClient(t, "127.0.0.1:9001")
		if cl == nil {
			t.Fatalf("Unable to create client #%d", i)
		}
		defer cl.close()
		clients[i] = cl
	}

	errCh := make(chan error, nclients)
	var sem sync.WaitGroup // Used as a semaphore to coordinate goroutines to begin concurrently
	sem.Add(1)
	ch := make(chan *Msg, nclients*niters) // channel for all replies
	for i := 0; i < nclients; i++ {
		go func(i int, cl *Client) {
			sem.Wait()
			for j := 0; j < niters; j++ {
				str := fmt.Sprintf("cl %d %d", i, j)
				m, err := cl.write("concWrite", str, 0)
				if err != nil {
					errCh <- err
					break
				} else {
					ch <- m
				}
			}
		}(i, clients[i])
	}
	time.Sleep(100 * time.Millisecond) // give goroutines a chance
	sem.Done()                         // Go!

	// There should be no errors
	for i := 0; i < nclients*niters; i++ {
		select {
		case m := <-ch:
			if m.Kind != 'O' {
				t.Fatalf("Concurrent write failed with kind=%c", m.Kind)
			}
		case err := <- errCh:
			t.Fatal(err)
		}
	}
	m, _ := clients[0].read("concWrite")
	// Ensure the contents are of the form "cl <i> 4"
	// The last write of any client ends with " 4"
	if !(m.Kind == 'C' && strings.HasSuffix(string(m.Contents), " 4")) {
		t.Fatalf("Expected to be able to read after 1000 writes. Got msg = %v", m)
	}

	killCluster()
	endTest()
	fmt.Println("Test End - TestRPC_ConcurrentWrites")
}
//*/

// nclients cas to the same file. At the end the file should be any one clients' last write.
// The only difference between this test and the ConcurrentWrite test above is that each
// client loops around until each CAS succeeds. The number of concurrent clients has been
// reduced to keep the testing time within limits.

///*
func TestRPC_ConcurrentCas(t *testing.T) {
	beginTest()
	startCluster(1)

	time.Sleep(time.Second*5)

	nclients := 5
	niters := 5

	clients := make([]*Client, nclients)
	for i := 0; i < nclients; i++ {
		cl := mkClient(t, "127.0.0.1:9001")
		if cl == nil {
			t.Fatalf("Unable to create client #%d", i)
		}
		defer cl.close()
		clients[i] = cl
	}

	var sem sync.WaitGroup // Used as a semaphore to coordinate goroutines to *begin* concurrently
	sem.Add(1)

	m, _ := clients[0].write("concCas", "first", 0)
	ver := m.Version
	if m.Kind != 'O' || ver == 0 {
		t.Fatalf("Expected write to succeed and return version")
	}

	var wg sync.WaitGroup
	wg.Add(nclients)

	errorCh := make(chan error, nclients)
	
	for i := 0; i < nclients; i++ {
		go func(i int, ver int, cl *Client) {
			sem.Wait()
			defer wg.Done()
			for j := 0; j < niters; j++ {
				str := fmt.Sprintf("cl %d %d", i, j)
				for {
					m, err := cl.cas("concCas", ver, str, 0)
					if err != nil {
						errorCh <- err
						return
					} else if m.Kind == 'O' {
						break
					} else if m.Kind != 'V' {
						errorCh <- errors.New(fmt.Sprintf("Expected 'V' msg, got %c", m.Kind))
						return
					}
					ver = m.Version // retry with latest version
				}
			}
		}(i, ver, clients[i])
	}

	time.Sleep(100 * time.Millisecond) // give goroutines a chance
	sem.Done()                         // Start goroutines
	wg.Wait()                          // Wait for them to finish
	select {
	case e := <- errorCh:
		t.Fatalf("Error received while doing cas: %v", e)
	default: // no errors
	}
	m, _ = clients[0].read("concCas")
	if !(m.Kind == 'C' && strings.HasSuffix(string(m.Contents), " 4")) {
		t.Fatalf("Expected to be able to read after 1000 writes. Got msg.Kind = %d, msg.Contents=%s", m.Kind, m.Contents)
	}

	killCluster()
	endTest()
	fmt.Println("Test End - TestRPC_ConcurrentCas")
}
//*/

//----------------------------------------------------------------------
// Utility functions

type Msg struct {
	// Kind = the first character of the command. For errors, it
	// is the first letter after "ERR_", ('V' for ERR_VERSION, for
	// example), except for "ERR_CMD_ERR", for which the kind is 'M'
	Kind      byte
	Filename  string
	Contents  []byte
	Numbytes  int
	Exptime   int // expiry time in seconds
	Version   int
	ReDirAddr string
}

func mkConn(Addr string) (*net.TCPConn, error) {
	var conn *net.TCPConn
	raddr, err := net.ResolveTCPAddr("tcp", Addr)
	if err == nil {
		conn, err = net.DialTCP("tcp", nil, raddr)
	}

	return conn, err
}

func (cl *Client) checkResp(msg *Msg) (res string, err error) {
	if msg.Kind == 'T' {
		time.Sleep(time.Millisecond * time.Duration(ELECTION_TIMEOUT))
		return "T_KIND_MSG", nil
	} else if msg.Kind == 'R' {
		cl.close()
		cl.conn, err = mkConn(msg.ReDirAddr)
		if err == nil {
			cl.reader = bufio.NewReader(cl.conn)
			cl.addr = msg.ReDirAddr
		}

		return "R_KIND_MSG", err
	}

	return "NOT_T_R_KIND_MSG", nil
}

func (cl *Client) read(filename string) (*Msg, error) {
	cmd := "read " + filename + "\r\n"
	return cl.sendRcv(cmd)
}

func (cl *Client) write(filename string, contents string, exptime int) (*Msg, error) {
	var cmd string
	if exptime == 0 {
		cmd = fmt.Sprintf("write %s %d\r\n", filename, len(contents))
	} else {
		cmd = fmt.Sprintf("write %s %d %d\r\n", filename, len(contents), exptime)
	}
	cmd += contents + "\r\n"
	msg, err := cl.sendRcv(cmd)
	if err == nil {
		res, err := cl.checkResp(msg)
		if (res == "T_KIND_MSG" || res == "R_KIND_MSG" ) && (err == nil) {
			return cl.write(filename, contents, exptime)
		}
	}
	return msg, err
}

func (cl *Client) cas(filename string, version int, contents string, exptime int) (*Msg, error) {
	var cmd string
	if exptime == 0 {
		cmd = fmt.Sprintf("cas %s %d %d\r\n", filename, version, len(contents))
	} else {
		cmd = fmt.Sprintf("cas %s %d %d %d\r\n", filename, version, len(contents), exptime)
	}
	cmd += contents + "\r\n"
	msg, err := cl.sendRcv(cmd)
	if err == nil {
		res, err := cl.checkResp(msg)
		if (res == "T_KIND_MSG" || res == "R_KIND_MSG" ) && (err == nil) {
			return cl.cas(filename, version, contents, exptime)
		}
	}
	return msg, err
}

func (cl *Client) delete(filename string) (*Msg, error) {
	cmd := "delete " + filename + "\r\n"
	msg, err := cl.sendRcv(cmd)
	if err == nil {
		res, err := cl.checkResp(msg)
		if (res == "T_KIND_MSG" || res == "R_KIND_MSG" ) && (err == nil) {
			return cl.delete(filename)
		}
	}
	return msg, err
}

var errNoConn = errors.New("Connection is closed")

type Client struct {
	conn   *net.TCPConn
	reader *bufio.Reader // a bufio Reader wrapper over conn
	addr   string
}

func mkClient(t *testing.T, Addr string) *Client {
	var client *Client
	raddr, err := net.ResolveTCPAddr("tcp", Addr)
	if err == nil {
		conn, err := net.DialTCP("tcp", nil, raddr)
		if err == nil {
			client = &Client{conn: conn, reader: bufio.NewReader(conn), addr: Addr}
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func (cl *Client) send(str string) error {
	if cl.conn == nil {
		return errNoConn
	}
	_, err := cl.conn.Write([]byte(str))
	if err != nil {
		err = fmt.Errorf("Write error in SendRaw: %v", err)
		cl.conn.Close()
		cl.conn = nil
	}
	return err
}

func (cl *Client) sendRcv(str string) (msg *Msg, err error) {
	if cl.conn == nil {
		return nil, errNoConn
	}
	err = cl.send(str)
	if err == nil {
		msg, err = cl.rcv()
	}
	return msg, err
}

func (cl *Client) close() {
	if cl != nil && cl.conn != nil {
		cl.conn.Close()
		cl.conn = nil
	}
}

func (cl *Client) rcv() (msg *Msg, err error) {
	// we will assume no errors in server side formatting
	line, err := cl.reader.ReadString('\n')
	if err == nil {
		msg, err = parseFirst(line)
		if err != nil {
			return nil, err
		}
		if msg.Kind == 'C' {
			contents := make([]byte, msg.Numbytes)
			var c byte
			for i := 0; i < msg.Numbytes; i++ {
				if c, err = cl.reader.ReadByte(); err != nil {
					break
				}
				contents[i] = c
			}
			if err == nil {
				msg.Contents = contents
				cl.reader.ReadByte() // \r
				cl.reader.ReadByte() // \n
			}
		}
	}
	if err != nil {
		cl.close()
	}
	return msg, err
}

func parseFirst(line string) (msg *Msg, err error) {
	fields := strings.Fields(line)
	msg = &Msg{}

	// Utility function fieldNum to int
	toInt := func(fieldNum int) int {
		var i int
		if err == nil {
			if fieldNum >=  len(fields) {
				err = errors.New(fmt.Sprintf("Not enough fields. Expected field #%d in %s\n", fieldNum, line))
				return 0
			}
			i, err = strconv.Atoi(fields[fieldNum])
		}
		return i
	}

	// Utility function fieldNum to string
	toString := func(fieldNum int) string {
		var str string
		if err == nil {
			if fieldNum >=  len(fields) {
				err = errors.New(fmt.Sprintf("Not enough fields. Expected field #%d in %s\n", fieldNum, line))
				return str
			}
			str = fields[fieldNum]
		}
		return str
	}

	if len(fields) == 0 {
		return nil, errors.New("Empty line. The previous command is likely at fault")
	}
	switch fields[0] {
	case "OK": // OK [version]
		msg.Kind = 'O'
		if len(fields) > 1 {
			msg.Version = toInt(1)
		}
	case "CONTENTS": // CONTENTS <version> <numbytes> <exptime> \r\n
		msg.Kind = 'C'
		msg.Version = toInt(1)
		msg.Numbytes = toInt(2)
		msg.Exptime = toInt(3)
	case "ERR_VERSION":
		msg.Kind = 'V'
		msg.Version = toInt(1)
	case "ERR_FILE_NOT_FOUND":
		msg.Kind = 'F'
	case "ERR_CMD_ERR":
		msg.Kind = 'M'
	case "ERR_INTERNAL":
		msg.Kind = 'I'
	case "ERR_REDIRECT":
		msg.Kind = 'R'
		msg.ReDirAddr = toString(1)
	case "ERR_TRY_LATER":
		msg.Kind = 'T'
	default:
		err = errors.New("Unknown response " + fields[0])
	}
	if err != nil {
		return nil, err
	} else {
		return msg, nil
	}
}