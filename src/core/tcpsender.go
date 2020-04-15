package core

import (
	"fmt"
	"net"
)

type TCPSender struct {
	conn        net.TCPConn
	dataChan    chan []byte
	exit        chan bool
	UserEncrypt *Encrypt
}

func CreateTCPSender(conn *net.TCPConn) *TCPSender {
	sender := &TCPSender{*conn, make(chan []byte), make(chan bool, 1), nil}
	sender.UserEncrypt = &Encrypt{}
	sender.UserEncrypt.InitEncrypt(164, 29, 30, 60, 241, 79, 251, 107)
	if sender == nil {
		return nil
	}
	return sender
}

func (sender *TCPSender) Send(msg NetMsg) {
	bytes, ok := msg.GetNetBytes()
	if !ok {
		return
	}

	sender.UserEncrypt.Encrypt(bytes, 0, len(bytes), false)
	sender.SendBytes(bytes)
}

func (sender *TCPSender) SendBytes(bytes []byte) {
	defer func() {
		if err := recover(); err != nil {
			LogError(err)
		}
	}()
	sender.dataChan <- bytes
}

func (sender *TCPSender) send(datas []byte) bool {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	n, err := sender.conn.Write(datas)
	if err != nil {
		LogError(fmt.Sprintf("TcpSender send error:", n, "reason:", err))
		sender.conn.CloseWrite()
		return false
	}
	return true
}

func (sender *TCPSender) Start() {
	defer func() {
		if err := recover(); err != nil {
			LogError(err)
		}
	}()
	for {
		select {
		case data := <-sender.dataChan:
			if !sender.send(data) {
				return
			}
		case <-sender.exit:
			for data := range sender.dataChan {
				sender.send(data)
			}
			close(sender.dataChan)
			close(sender.exit)
			sender.conn.CloseWrite()
		}
	}
}

func (sender *TCPSender) Close() {
	defer func() {
		if x := recover(); x != nil {
			LogError("TcpSender Close failed", x)
		}
	}()
	sender.exit <- true
}
