package main

import (
	"fmt"
	"net"
	"sync/atomic"
)

// PacketInfo
type PacketInfo struct {
	cmd string
	msg string
}

// SessionInfo
type SessionInfo struct {
	uid     uint64
	conn    net.Conn
	ownChan chan PacketInfo
}

var chanMap = make(map[uint64](chan PacketInfo), 10)

func sendPacketToAllChan(si SessionInfo, msg string, excludeSessUidx uint64) {
	newpkt := PacketInfo{}
	newpkt.cmd = "recv"
	newpkt.msg = msg

	fmt.Println(msg)
	for k, v := range chanMap {
		if k == excludeSessUidx {
			continue
		}
		v <- newpkt
		fmt.Println("sess_uid :", k)
	}
}

func readPacket(si SessionInfo) {
	data := make([]byte, 4096)

	//read packet context
	for {
		n, err := si.conn.Read(data)
		if err != nil {
			fmt.Println(err)

			//send close message
			newpkt := PacketInfo{}
			newpkt.cmd = "close"
			si.ownChan <- newpkt

			return
		}

		sendPacketToAllChan(si, string(data[:n]), si.uid)
	}
}

func prcMsg(si SessionInfo) {
	for {
		pi := <-si.ownChan
		if pi.cmd == "close" {
			fmt.Println("closed :", si.uid)
			delete(chanMap, si.uid)
			si.conn.Close()
			break
		}
		si.conn.Write([]byte(pi.msg))
	}
}

func requestHandler(si SessionInfo) {
	go readPacket(si)
	go prcMsg(si)
}

func main() {
	var sessuid uint64

	ln, err := net.Listen("tcp", ":5432")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		atomic.AddUint64(&sessuid, 1)

		//make a new session
		si := SessionInfo{}
		si.uid = sessuid
		si.conn = conn
		si.ownChan = make(chan PacketInfo, 10)

		//register new channel
		chanMap[sessuid] = si.ownChan
		go requestHandler(si)
	}
}
