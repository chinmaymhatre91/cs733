package main

import (
	"bufio"
	"fmt"
	"github.com/chinmaymhatre91/cs733/assignment4/fs"
	"encoding/gob"
	"bytes"
	"net"
	"os"
	"strconv"
	"sync"
)

var crlf = []byte{'\r', '\n'}

type FileServerConfig struct {
	Id int
	Host string
	RaftPort int
	ClientPort int
	ElectionTimeout int
	HeartbeatTimeout int
}

var ELECTION_TIMEOUT int = 800
var HEARTBEAT_TIMEOUT int = 400

var my_file_clust = []FileServerConfig{
	{Id: 101, Host: "127.0.0.1", RaftPort: 8001, ClientPort: 9001, ElectionTimeout: ELECTION_TIMEOUT, HeartbeatTimeout: HEARTBEAT_TIMEOUT},
	{Id: 102, Host: "127.0.0.1", RaftPort: 8002, ClientPort: 9002, ElectionTimeout: ELECTION_TIMEOUT, HeartbeatTimeout: HEARTBEAT_TIMEOUT},
	{Id: 103, Host: "127.0.0.1", RaftPort: 8003, ClientPort: 9003, ElectionTimeout: ELECTION_TIMEOUT, HeartbeatTimeout: HEARTBEAT_TIMEOUT},
	{Id: 104, Host: "127.0.0.1", RaftPort: 8004, ClientPort: 9004, ElectionTimeout: ELECTION_TIMEOUT, HeartbeatTimeout: HEARTBEAT_TIMEOUT},
	{Id: 105, Host: "127.0.0.1", RaftPort: 8005, ClientPort: 9005, ElectionTimeout: ELECTION_TIMEOUT, HeartbeatTimeout: HEARTBEAT_TIMEOUT}}

func check(obj interface{}) {
	if obj != nil {
		fmt.Println(obj)
		os.Exit(1)
	}
}

type FileServer struct {
	Id int
	RN Node
	Host string
	ClientPort int
	ConnMapMux sync.Mutex
	ConnMap map[int]*net.TCPConn
}

func getFileServerIndex(server_id int) (int) {
	var Index int 

	for i, entry := range my_file_clust {
		if entry.Id == server_id {
			Index = i
			break 
		}
	}

	return Index
}

func getCluster() ([]NetConfig) {
	var clust []NetConfig 

	for _, entry := range my_file_clust {
		clust = append(clust, NetConfig{Id: entry.Id, Host: entry.Host, Port: entry.RaftPort})
	}

	return clust
}

func getConfig(server_id int, clust_no int) (Config) {
	var conf Config
	file_dir := "node"

	for _, entry := range my_file_clust {
		if entry.Id == server_id {
			conf.Id = server_id
			conf.Cluster = getCluster()
			conf.LogFileDir = file_dir + strconv.Itoa(server_id) + "/log"
			conf.StateFileDir = file_dir + strconv.Itoa(server_id) + "/state"			
			conf.ElectionTimeout = my_file_clust[getFileServerIndex(server_id)].ElectionTimeout
			conf.HeartbeatTimeout = my_file_clust[getFileServerIndex(server_id)].HeartbeatTimeout
			if clust_no == 1 && entry.Id != 101 {
				conf.ElectionTimeout = conf.ElectionTimeout * 10
				conf.HeartbeatTimeout = conf.HeartbeatTimeout * 10
			}
		}
	}

	return conf
}

func getHost(server_id int) (string) {
	var Host string 

	for _, entry := range my_file_clust {
		if entry.Id == server_id {
			Host = entry.Host
			break 
		}
	}

	return Host
}

func getClientPort(server_id int) (int) {
	var ClientPort int 

	for _, entry := range my_file_clust {
		if entry.Id == server_id {
			ClientPort = entry.ClientPort
			break 
		}
	}

	return ClientPort
}

func NewFileServer(server_id int, clust_no int) (FileServer) {
	var FSR FileServer

	FSR.Id = server_id
	conf := getConfig(server_id, clust_no)
	//fmt.Println(conf)
	FSR.RN = New(conf)
	FSR.Host = getHost(server_id)
	FSR.ClientPort = getClientPort(server_id)
	FSR.ConnMap = make(map[int]*net.TCPConn)

	return FSR
}

func (FSR *FileServer) mapAdd(clientId int, tcp_conn *net.TCPConn) {
	FSR.ConnMapMux.Lock()
	defer FSR.ConnMapMux.Unlock()
	FSR.ConnMap[clientId] = tcp_conn
}

func (FSR *FileServer) mapDel(clientId int) {
	FSR.ConnMapMux.Lock()
	defer FSR.ConnMapMux.Unlock()
	delete(FSR.ConnMap, clientId)
}

func (FSR *FileServer) mapFind(clientId int) (*net.TCPConn, bool) {
	FSR.ConnMapMux.Lock()
	defer FSR.ConnMapMux.Unlock()
	tcp_conn, ok :=  FSR.ConnMap[clientId]
	return tcp_conn, ok
}

func reply(conn *net.TCPConn, msg *fs.Msg) bool {
	var err error
	write := func(data []byte) {
		if err != nil {
			return
		}
		_, err = conn.Write(data)
	}
	var resp string
	switch msg.Kind {
	case 'C': // read response
		resp = fmt.Sprintf("CONTENTS %d %d %d", msg.Version, msg.Numbytes, msg.Exptime)
	case 'O':
		resp = "OK "
		if msg.Version > 0 {
			resp += strconv.Itoa(msg.Version)
		}
	case 'F':
		resp = "ERR_FILE_NOT_FOUND"
	case 'V':
		resp = "ERR_VERSION " + strconv.Itoa(msg.Version)
	case 'M':
		resp = "ERR_CMD_ERR"
	case 'I':
		resp = "ERR_INTERNAL"
	case 'R':
		resp = "ERR_REDIRECT " + msg.ReDirAddr
	case 'T':
		resp = "ERR_TRY_LATER"
	default:
		fmt.Printf("Unknown response kind '%c'", msg.Kind)
		return false
	}
	resp += "\r\n"
	write([]byte(resp))
	if msg.Kind == 'C' {
		write(msg.Contents)
		write(crlf)
	}
	return err == nil
}

func encode(msg *fs.Msg) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	err := enc.Encode(msg)
	return buf.Bytes(), err
}

func decode(data []byte) (*fs.Msg, error) {
	msg := &fs.Msg{}
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(msg)
	return msg, err
}

func (FSR *FileServer) serve(clientId int, conn *net.TCPConn) {
	reader := bufio.NewReader(conn)
	for {
		msg, msgerr, fatalerr := fs.GetMsg(reader)
		if fatalerr != nil || msgerr != nil {
			reply(conn, &fs.Msg{Kind: 'M'})
			FSR.mapDel(clientId)
			conn.Close()
			break
		}

		if msgerr != nil {
			if (!reply(conn, &fs.Msg{Kind: 'M'})) {
				FSR.mapDel(clientId)
				conn.Close()
				break
			}
		}

		//fmt.Println("["+strconv.Itoa(FSR.Id)+"] - Msg Kind ", string(msg.Kind))
		if msg.Kind == 'r' {
			response := fs.ProcessMsg(msg)
			if !reply(conn, response) {
				FSR.mapDel(clientId)
				conn.Close()
				break
			}
		} else {
			msg.ClientId = clientId
			data, err := encode(msg)
			if err != nil {
				reply(conn, &fs.Msg{Kind: 'I'})
				FSR.mapDel(clientId)
				conn.Close()
				break
			}
			err = FSR.RN.Append(data)
			if err != nil {
				reply(conn, &fs.Msg{Kind: 'I'})
				FSR.mapDel(clientId)
				conn.Close()
				break
			}
		}
	}
}

func (FSR *FileServer) serverFile() {
	ch, err := FSR.RN.CommitChannel()
	if err != nil {
		panic("[Error] No CommitChannel Received")
	}

	lastAppliedIndex := 0
	for {
		commitInfo := <- ch 

		if commitInfo.Err != nil {
			msg, err := decode(commitInfo.Data)
			if err !=nil {
				panic("[Error] Decode error")
			}
			conn, ok := FSR.mapFind(msg.ClientId)
			if !ok {
				panic("[Error] Client not found for this msg")
			}

			switch commitInfo.Err.Error() {
			case "ERR_CONTACT_LEADER":
				ldrHost := getHost(commitInfo.LeaderId)
				ldrClientPort := getClientPort(commitInfo.LeaderId)
				reply(conn, &fs.Msg{Kind: 'R', ReDirAddr: ldrHost+":"+strconv.Itoa(ldrClientPort)})
			case "ERR_TRY_LATER":
				reply(conn, &fs.Msg{Kind: 'T'})
			default:
				panic("[Error] Unknown error - "+commitInfo.Err.Error())
			}
		} else {
			for i:=lastAppliedIndex+1; i<=commitInfo.Index; i++ {
				data, err := FSR.RN.Get(i)
				if err != nil {
					panic("[Error] Error from FS.RN.Get - "+err.Error())
				}

				msg, err := decode(data)
				if err !=nil {
					panic("[Error] Decode error")
				}

				response := fs.ProcessMsg(msg)

				conn, ok := FSR.mapFind(msg.ClientId)
				if ok {
					if commitInfo.LeaderId != FSR.Id {
						panic("[Error] Got CommitInfo & Client found, but node is not leader now")
					}
					reply(conn, response)
				}
			}
			lastAppliedIndex = max(lastAppliedIndex, commitInfo.Index)
		}
	}
}

func (FSR *FileServer) serverMain() {
	clientId := FSR.Id * 1000
	tcpaddr, err := net.ResolveTCPAddr("tcp", FSR.Host+":"+strconv.Itoa(FSR.ClientPort))
	check(err)
	tcp_acceptor, err := net.ListenTCP("tcp", tcpaddr)
	check(err)

	for {
		tcp_conn, err := tcp_acceptor.AcceptTCP()
		check(err)
		FSR.mapAdd(clientId, tcp_conn)
		go FSR.serve(clientId, tcp_conn)
		clientId = clientId + 1
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println(os.Args)
		panic("[Error] Wrong number of arguments")
	}
	
	server_id, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic("[Error] Non-nummeric server id")
	}
	clust_no, err := strconv.Atoi(os.Args[2])
	if err != nil {
		panic("[Error] Non-nummeric cluster no")
	} 

	FSR := NewFileServer(server_id, clust_no)
	go FSR.serverFile()
	fmt.Println(server_id, " Looks good")
	FSR.serverMain()

}