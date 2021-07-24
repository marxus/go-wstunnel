package wstunnel // import "marxus.github.io/go/wstunnel"

import (
	"errors"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"io"
	"net"
	"os"
	"time"
)

var (
	DEBUG = false
	debug = func(args ...interface{}) { if DEBUG { fmt.Print(args) } }
)

func TunnelServer(_ws, conn net.Conn) { Tunnel(ws.StateServerSide, _ws, conn) }
func TunnelClient(_ws, conn net.Conn) { Tunnel(ws.StateClientSide, _ws, conn) }
func Tunnel(side ws.State, _ws, conn net.Conn) {
	close := func() { _ws.Close(); conn.Close() }
	wsRead, wsWrite := map[ws.State]func(rw io.ReadWriter) ([]byte, error){
		ws.StateServerSide: wsutil.ReadClientBinary,
		ws.StateClientSide: wsutil.ReadServerBinary,
	}[side], wsutil.WriteMessage

	go func() { // conn -> _ws
		defer close()
		for {
			data := make([]byte, 32768)
			conn.SetReadDeadline(time.Now().Add(10 * time.Second))
			n, err := conn.Read(data)
			if err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					err := wsWrite(_ws, side, ws.OpPing, nil)
					if err != nil {
						debug("ws:Ping", err)
						return
					}
					continue
				}
				debug("conn:Read", err)
				return
			}
			data = data[:n]
			err = wsWrite(_ws, side, ws.OpBinary, data)
			if err != nil {
				debug("ws:Write", err)
				return
			}
		}
	}()

	defer close()
	for { // _ws -> conn
		data, err := wsRead(_ws)
		if err != nil {
			debug("ws:Read", err)
			return
		}
		_, err = conn.Write(data)
		if err != nil {
			debug("conn:Write", err)
			return
		}
	}
}
