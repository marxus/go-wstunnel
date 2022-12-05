package wstunnel // import "marxus.github.io/go/wstunnel"

import (
	"net"
	"net/http"

	"golang.org/x/net/websocket"
)

type handler struct {
	open func() net.Conn
}

func handshake(*websocket.Config, *http.Request) error { return nil }

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	websocket.Server{
		Handshake: handshake,
		Handler: func(ws *websocket.Conn) {
			conn := h.open()
			if conn == nil {
				ws.Close()
				return
			}

			rxtx(ws, conn)
			conn.Close()
		},
	}.ServeHTTP(w, r)
}

func NewHandler(open func() net.Conn) *handler {
	return &handler{open}
}
