package wstunnel // import "marxus.github.io/go/wstunnel"

import (
	"errors"
	"net"
	"os"
	"time"

	"golang.org/x/net/websocket"
)

var rs = websocket.Codec{
	func(v interface{}) ([]byte, byte, error) {
		if v == nil {
			return nil, websocket.PingFrame, nil
		}
		return v.([]byte), websocket.BinaryFrame, nil
	},

	func(msg []byte, _ byte, v interface{}) error {
		*v.(*[]byte) = msg
		return nil
	},
}

func rx(ws *websocket.Conn, conn net.Conn) {
	for {
		data := make([]byte, 32768)
		n, err := conn.Read(data)
		if err != nil {
			break
		}
		rs.Send(ws, data[:n])
	}
	ws.Close()
}

func tx(ws *websocket.Conn, conn net.Conn) {
	for { // ws -> conn
		data := make([]byte, 32768)
		ws.SetReadDeadline(time.Now().Add(10 * time.Second)) // TODO: move deadline to a seperate timer
		err := rs.Receive(ws, &data)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				rs.Send(ws, nil)
				continue
			}
			break
		}
		conn.Write(data)
	}
	ws.Close()
}

func rxtx(ws *websocket.Conn, conn net.Conn) {
	go rx(ws, conn)
	tx(ws, conn)
}
