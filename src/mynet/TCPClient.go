// TCPClient
package mynet

import (
	. "core"
	. "def"
	"fmt"
	"io"
	. "message"
	"net"
	"os"
	"reflect"
)

type TCPUserConn struct {
	Conn   *net.TCPConn
	Sender *TCPSender
}

func (this *TCPUserConn) close() {
	this.Conn.CloseWrite()
}

func (this *TCPUserConn) processMessage(header *PackHeader, datas []byte) {
	LogDebug("receive cmd", header.Cmd)
	if header.Cmd == CMD_CONNECTION {
		msg := &Command{header.Cmd, datas, nil, nil}
		defer func() {
			if err := recover(); err != nil {
				LogError("User processClientMsg failed:", err, " cmd:", msg.Cmd)
			}
		}()
		netMsg := CreateNetMsg(msg)
		netMsg.Process(this.Sender)

	} else {
		msg := &Command{header.Cmd, datas, nil, nil}
		defer func() {
			if err := recover(); err != nil {
				LogError("User processClientMsg failed:", err, " cmd:", msg.Cmd)
			}
		}()
		netMsg := CreateNetMsg(msg)
		netMsg.Process(this.Sender)
	}
}

func Connect() {

	defer func() {
		if err := recover(); err != nil {
			LogError(err) //这里的err其实就是panic传入的内容
		}
	}()

	serverConfig := &ServerConfig{}
	err := serverConfig.ReadServerConfig()
	if err != nil {
		LogError("Read server config error.")
		return
	}

	CreateChanMgr()

	server := fmt.Sprintf("%v:%d", serverConfig.Host, 5006)
	LogDebug("Server is ", server)
	tcpAddr, err := net.ResolveTCPAddr("tcp4", server)
	if err != nil {
		LogError("Fatal error: %s", err.Error())
		os.Exit(1)
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		LogError("Fatal error: %s", err.Error())
		os.Exit(1)
	}

	client := &TCPUserConn{}
	client.Conn = conn
	client.Sender = CreateTCPSender(conn)

	go client.Sender.Start()
	go ProcessRecv(client)

	msgConnection := &MsgConnection{}
	client.Sender.Send(msgConnection)

	sysChan := make(chan *Command)
	RegisterChan(SYSTEM_CHAN_ID, sysChan)
	for {
		select {
		case msg := <-sysChan:
			LogInfo("system command :", msg.Cmd)
			if msg.Cmd == CMD_SYSTEM_MAIN_CLOSE {
				return
			}
		}
	}
}

func ProcessRecv(handler *TCPUserConn) {
	defer func() {
		if err := recover(); err != nil {
			LogError(err) //这里的err其实就是panic传入的内容
		}
	}()
	conn := handler.Conn
	defer conn.CloseWrite()
	defer handler.close()

	for {
		headerBytes := make([]byte, HEADER_LENGTH)
		_, err := io.ReadFull(conn, headerBytes)
		if err != nil {
			LogError("Read Data Error, maybe the socket is closed!  ")
			break
		}

		header := &PackHeader{}
		Byte2Struct(reflect.ValueOf(header), headerBytes)

		LogDebug("Header", header.Cmd, header.Length, header.Tag, header.Version)
		bodyBytes := make([]byte, header.Length-HEADER_LENGTH)
		_, err = io.ReadFull(conn, bodyBytes)
		if err != nil {
			LogError("read data error ", err.Error())
			break
		}

		handler.processMessage(header, bodyBytes)
	}
}
