package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"net"
	"strconv"
	"strings"
	"time"
)

type my_file struct {
	fname   string
	ver     int64
	nBytes  int64
	exptime time.Time
	content []byte
}

type my_task struct {
	cmd     string
	fname   string
	ver     int64
	nBytes  int64
	expdur  int64
	content []byte
	ret     chan []byte
}

func handleDataAccess(my_map map[string]my_file, ch chan my_task) {
	for {
		tsk := <-ch

		var output string

		switch tsk.cmd {

		case "read":

			_, ok := my_map[tsk.fname]
			if ok == false {
				output = "ERR_FILE_NOT_FOUND\r\n"
			} else {
				if my_map[tsk.fname].exptime != time.Date(1991, time.May, 4, 0, 0, 0, 0, time.UTC) && time.Now().After(my_map[tsk.fname].exptime) {
					delete(my_map, tsk.fname)
					output = "ERR_FILE_NOT_FOUND\r\n"
				} else {
					if my_map[tsk.fname].exptime == time.Date(1991, time.May, 4, 0, 0, 0, 0, time.UTC) {
						output = "CONTENTS " + strconv.FormatInt(my_map[tsk.fname].ver, 10) + " " + strconv.FormatInt(int64(len(my_map[tsk.fname].content)), 10) + " " + strconv.FormatInt(0, 10) + "\r\n" + string(my_map[tsk.fname].content) + "\r\n"
					} else {
						output = "CONTENTS " + strconv.FormatInt(my_map[tsk.fname].ver, 10) + " " + strconv.FormatInt(int64(len(my_map[tsk.fname].content)), 10) + " " + strconv.FormatInt(int64(math.Ceil(my_map[tsk.fname].exptime.Sub(time.Now()).Seconds())), 10) + "\r\n" + string(my_map[tsk.fname].content) + "\r\n"
					}
				}
			}

		case "write":

			_, ok := my_map[tsk.fname]
			if ok == false {
				if tsk.expdur == 0 {
					my_map[tsk.fname] = my_file{tsk.fname, 0, tsk.nBytes, time.Date(1991, time.May, 4, 0, 0, 0, 0, time.UTC), make([]byte, len(tsk.content))}
					copy(my_map[tsk.fname].content[:], tsk.content[:])
				} else {
					my_map[tsk.fname] = my_file{tsk.fname, 0, tsk.nBytes, time.Now().Add(time.Duration(tsk.expdur) * time.Second), make([]byte, len(tsk.content))}
					copy(my_map[tsk.fname].content[:], tsk.content[:])
				}
				output = "OK " + strconv.FormatInt(my_map[tsk.fname].ver, 10) + "\r\n"
			} else {
				if my_map[tsk.fname].exptime != time.Date(1991, time.May, 4, 0, 0, 0, 0, time.UTC) && time.Now().After(my_map[tsk.fname].exptime) {
					if tsk.expdur == 0 {
						my_map[tsk.fname] = my_file{tsk.fname, 0, tsk.nBytes, time.Date(1991, time.May, 4, 0, 0, 0, 0, time.UTC), make([]byte, len(tsk.content))}
						//my_map[tsk.fname].content = make([]byte, len(tsk.content))
						copy(my_map[tsk.fname].content[:], tsk.content[:])
						//my_map[tsk.fname].nBytes = tsk.nBytes
						//my_map[tsk.fname].exptime = time.Date(1991, time.May, 4, 0, 0, 0, 0, time.UTC)
						//my_map[tsk.fname].ver = 0
					} else {
						my_map[tsk.fname] = my_file{tsk.fname, 0, tsk.nBytes, time.Now().Add(time.Duration(tsk.expdur) * time.Second), make([]byte, len(tsk.content))}
						//my_map[tsk.fname].content = make([]byte, len(tsk.content))
						copy(my_map[tsk.fname].content[:], tsk.content[:])
						//my_map[tsk.fname].nBytes = tsk.nBytes
						//my_map[tsk.fname].exptime = time.Now().Add(time.Duration(tsk.expdur) * time.Second)
						//my_map[tsk.fname].ver = 0
					}

					output = "OK " + strconv.FormatInt(my_map[tsk.fname].ver, 10) + "\r\n"
				} else {
					if tsk.expdur == 0 {
						my_map[tsk.fname] = my_file{tsk.fname, my_map[tsk.fname].ver + 1, tsk.nBytes, time.Date(1991, time.May, 4, 0, 0, 0, 0, time.UTC), make([]byte, len(tsk.content))}
						//my_map[tsk.fname].content = make([]byte, len(tsk.content))
						copy(my_map[tsk.fname].content[:], tsk.content[:])
						//my_map[tsk.fname].nBytes = tsk.nBytes
						//my_map[tsk.fname].exptime = time.Date(1991, time.May, 4, 0, 0, 0, 0, time.UTC)
						//my_map[tsk.fname].ver++
					} else {
						my_map[tsk.fname] = my_file{tsk.fname, my_map[tsk.fname].ver + 1, tsk.nBytes, time.Now().Add(time.Duration(tsk.expdur) * time.Second), make([]byte, len(tsk.content))}
						//my_map[tsk.fname].content = make([]byte, len(tsk.content))
						copy(my_map[tsk.fname].content[:], tsk.content[:])
						//my_map[tsk.fname].nBytes = tsk.nBytes
						//my_map[tsk.fname].exptime = time.Now().Add(time.Duration(tsk.expdur) * time.Second)
						//my_map[tsk.fname].ver++
					}

					output = "OK " + strconv.FormatInt(my_map[tsk.fname].ver, 10) + "\r\n"
				}
			}

		case "cas":

			_, ok := my_map[tsk.fname]
			if ok == false {
				output = "ERR_FILE_NOT_FOUND\r\n"
			} else {
				if my_map[tsk.fname].exptime != time.Date(1991, time.May, 4, 0, 0, 0, 0, time.UTC) && time.Now().After(my_map[tsk.fname].exptime) {
					delete(my_map, tsk.fname)
					output = "ERR_FILE_NOT_FOUND\r\n"
				} else {
					if my_map[tsk.fname].ver != tsk.ver {
						output = "ERR_VERSION\r\n"
					} else {
						if tsk.expdur == 0 {
							my_map[tsk.fname] = my_file{tsk.fname, my_map[tsk.fname].ver + 1, tsk.nBytes, time.Date(1991, time.May, 4, 0, 0, 0, 0, time.UTC), make([]byte, len(tsk.content))}
							//my_map[tsk.fname].content = make([]byte, len(tsk.content))
							copy(my_map[tsk.fname].content[:], tsk.content[:])
							//my_map[tsk.fname].nBytes = tsk.nBytes
							//my_map[tsk.fname].exptime = time.Date(1991, time.May, 4, 0, 0, 0, 0, time.UTC)
							//my_map[tsk.fname].ver++
						} else {
							my_map[tsk.fname] = my_file{tsk.fname, my_map[tsk.fname].ver + 1, tsk.nBytes, time.Now().Add(time.Duration(tsk.expdur) * time.Second), make([]byte, len(tsk.content))}
							//my_map[tsk.fname].content = make([]byte, len(tsk.content))
							copy(my_map[tsk.fname].content[:], tsk.content[:])
							//my_map[tsk.fname].nBytes = tsk.nBytes
							//my_map[tsk.fname].exptime = time.Now().Add(time.Duration(tsk.expdur) * time.Second)
							//my_map[tsk.fname].ver++
						}
						output = "OK " + strconv.FormatInt(my_map[tsk.fname].ver, 10) + "\r\n"
					}
				}
			}

		case "delete":

			_, ok := my_map[tsk.fname]
			if ok == false {
				output = "ERR_FILE_NOT_FOUND\r\n"
			} else {
				if my_map[tsk.fname].exptime != time.Date(1991, time.May, 4, 0, 0, 0, 0, time.UTC) && time.Now().After(my_map[tsk.fname].exptime) {
					delete(my_map, tsk.fname)
					output = "ERR_FILE_NOT_FOUND\r\n"
				} else {
					delete(my_map, tsk.fname)
					output = "OK\r\n"
				}
			}

		}

		//fmt.Println(output)
		tsk.ret <- []byte(output)
	}
}

func handleConnection(conn net.Conn, ch chan my_task) {
	for {
		new_conn := bufio.NewReader(conn)
		message, err := new_conn.ReadString('\n')
		if err == io.EOF || err != nil {
			conn.Close()
			break
		} else {
			var tsk my_task

			msg := strings.Split(message, " ")

			switch msg[0] {
			case "read":
				// Read command implimentation
				// Command : read <filename>\r\n
				// Return  : CONTENTS <version> <numbytes> <exptime> \r\n
				//			 <content bytes>\r\n

				if len(msg) == 2 {
					tsk.cmd = "read"
					tsk.fname = msg[1][:len(msg[1])-2]
					tsk.ret = make(chan []byte, 1)

					ch <- tsk
					tmp := <-tsk.ret
					conn.Write(tmp)
				} else {
					conn.Write([]byte("ERR_CMD_ERR\r\n"))
					conn.Close()
					break
				}

			case "write":
				// Write command implimentation
				// Command : write <filename> <numbytes> [<exptime>]\r\n
				//			 <content bytes>\r\n
				// Return  : OK <version>\r\n

				if len(msg) == 3 || len(msg) == 4 {
					var num_bytes int64
					if len(msg) == 3 {
						num_bytes, _ = strconv.ParseInt(msg[2][:len(msg[2])-2], 10, 64)
					} else {
						num_bytes, _ = strconv.ParseInt(msg[2], 10, 64)
					}
					buff := make([]byte, num_bytes)
					_, err := io.ReadFull(new_conn, buff)
					if err != nil {
						conn.Write([]byte("ERR_CMD_ERR\r\n"))
						conn.Close()
						break
					}

					dummy_buff := make([]byte, 2)
					_, err = io.ReadFull(new_conn, dummy_buff)
					if err != nil {
						conn.Write([]byte("ERR_CMD_ERR\r\n"))
						conn.Close()
						break
					}

					if string(dummy_buff) == "\r\n" {
						tsk.cmd = "write"
						tsk.fname = msg[1]
						if len(msg) == 3 {
							tsk.nBytes, err = strconv.ParseInt(msg[2][:len(msg[2])-2], 10, 64)
							if err != nil {
								conn.Write([]byte("ERR_CMD_ERR\r\n"))
								conn.Close()
								break
							}
							tsk.expdur = int64(0)
						} else {
							tsk.nBytes, err = strconv.ParseInt(msg[2], 10, 64)
							if err != nil {
								conn.Write([]byte("ERR_CMD_ERR\r\n"))
								conn.Close()
								break
							}
							tsk.expdur, err = strconv.ParseInt(msg[3][:len(msg[3])-2], 10, 64)
							if err != nil {
								conn.Write([]byte("ERR_CMD_ERR\r\n"))
								conn.Close()
								break
							}
						}
						tsk.content = make([]byte, len(buff))
						copy(tsk.content[:], buff[:])
						tsk.ret = make(chan []byte, 1)

						ch <- tsk
						conn.Write(<-tsk.ret)
					} else {
						conn.Write([]byte("ERR_CMD_ERR\r\n"))
						conn.Close()
						break
					}
				} else {
					conn.Write([]byte("ERR_CMD_ERR\r\n"))
					conn.Close()
					break
				}

			case "cas":
				// Write command implimentation
				// Command : cas <filename> <version> <numbytes> [<exptime>]\r\n
				//			 <content bytes>\r\n
				// Return  : OK <version>\r\n

				if len(msg) == 4 || len(msg) == 5 {
					var num_bytes int64
					if len(msg) == 4 {
						num_bytes, _ = strconv.ParseInt(msg[3][:len(msg[3])-2], 10, 64)
					} else {
						num_bytes, _ = strconv.ParseInt(msg[3], 10, 64)
					}
					buff := make([]byte, num_bytes)
					_, err := io.ReadFull(new_conn, buff)
					if err != nil {
						conn.Write([]byte("ERR_CMD_ERR\r\n"))
						conn.Close()
						break
					}

					dummy_buff := make([]byte, 2)
					_, err = io.ReadFull(new_conn, dummy_buff)
					if err != nil {
						conn.Write([]byte("ERR_CMD_ERR\r\n"))
						conn.Close()
						break
					}

					if string(dummy_buff) == "\r\n" {
						tsk.cmd = "cas"
						tsk.fname = msg[1]
						tsk.ver, _ = strconv.ParseInt(msg[2], 10, 64)
						if len(msg) == 4 {
							tsk.nBytes, _ = strconv.ParseInt(msg[3][:len(msg[3])-2], 10, 64)
							if err != nil {
								conn.Write([]byte("ERR_CMD_ERR\r\n"))
								conn.Close()
								break
							}
							tsk.expdur = int64(0)
						} else {
							tsk.nBytes, err = strconv.ParseInt(msg[3], 10, 64)
							if err != nil {
								conn.Write([]byte("ERR_CMD_ERR\r\n"))
								conn.Close()
								break
							}
							tsk.expdur, err = strconv.ParseInt(msg[4][:len(msg[4])-2], 10, 64)
							if err != nil {
								conn.Write([]byte("ERR_CMD_ERR\r\n"))
								conn.Close()
								break
							}
						}
						tsk.content = make([]byte, len(buff))
						copy(tsk.content[:], buff[:])
						tsk.ret = make(chan []byte, 1)

						ch <- tsk
						conn.Write(<-tsk.ret)
					} else {
						conn.Write([]byte("ERR_CMD_ERR\r\n"))
						conn.Close()
						break
					}
				} else {
					conn.Write([]byte("ERR_CMD_ERR\r\n"))
					conn.Close()
					break
				}

			case "delete":
				// Command : delete <filename>\r\n
				// Return  : OK\r\n

				if len(msg) == 2 {
					tsk.cmd = "delete"
					tsk.fname = msg[1][:len(msg[1])-2]
					tsk.ret = make(chan []byte, 1)

					ch <- tsk
					conn.Write(<-tsk.ret)
				} else {
					conn.Write([]byte("ERR_CMD_ERR\r\n"))
					conn.Close()
					break
				}

			default:
				conn.Write([]byte("ERR_CMD_ERR\r\n"))
				conn.Close()
				break
			}
		}
	}
	fmt.Println("End")
}

func serverMain() {
	ch := make(chan my_task, 1)
	my_map := make(map[string]my_file)

	go handleDataAccess(my_map, ch)

	fmt.Println("Server Started.....")

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		// handle error
		fmt.Println("[Error] : Error listing on port")
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
			fmt.Println("[Error] : Error accepting connection")
		}
		go handleConnection(conn, ch)
	}
}

func main() {
	serverMain()
}
