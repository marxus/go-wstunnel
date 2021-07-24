package wstunnel // import "marxus.github.io/go/wstunnel"

import (
	"context"
	"errors"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

var (
	DEBUG = false
	debug = func(args ...interface{}) {
		if DEBUG {
			fmt.Print(args)
		}
	}
)

func ServerSide(_ws, conn net.Conn) { IOLoop(ws.StateServerSide, _ws, conn) }
func ClientSide(_ws, conn net.Conn) { IOLoop(ws.StateClientSide, _ws, conn) }
func IOLoop(side ws.State, _ws, conn net.Conn) {
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
						debug("IOLoop:ws:Ping", err)
						return
					}
					continue
				}
				debug("IOLoop:conn:Read", err)
				return
			}
			data = data[:n]
			err = wsWrite(_ws, side, ws.OpBinary, data)
			if err != nil {
				debug("IOLoop:ws:Write", err)
				return
			}
		}
	}()

	defer close()
	for { // _ws -> conn
		data, err := wsRead(_ws)
		if err != nil {
			debug("IOLoop:ws:Read", err)
			return
		}
		_, err = conn.Write(data)
		if err != nil {
			debug("IOLoop:conn:Write", err)
			return
		}
	}
}

type TunnelHandler struct {
	http.Handler
	network, address string
}

func (h *TunnelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ws, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		debug("TunnelHandler:ServeHTTP:ws:UpgradeHTTP", err)
		return
	}

	conn, err := net.Dial(h.network, h.address)
	if err != nil {
		debug("TunnelHandler:ServeHTTP:net:Dial", err)
		_ws.Close()
		return
	}

	ServerSide(_ws, conn)
}

func NewTunnelHandler(network, address string) *TunnelHandler {
	return &TunnelHandler{nil, network, address}
}

func Tunnel(conn net.Conn, urlstr string) {
	_ws, _, _, err := ws.Dial(context.Background(), urlstr)
	if err != nil {
		debug("Tunnel:ws:Dial", err)
		return
	}

	ClientSide(_ws, conn)
}
