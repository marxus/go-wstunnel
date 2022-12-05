package wstunnel // import "marxus.github.io/go/wstunnel"

import (
	"context"
	"crypto/tls"
	"net"
	"strings"

	"golang.org/x/net/websocket"
)

type TunnelError struct {
	ErrorString string
}

func (err *TunnelError) Error() string { return err.ErrorString }

var ErrUsedTunnelConfig = &TunnelError{"tunnel config already been used"}

type TunnelConfig struct {
	Proxy     net.Conn
	TlsConfig *tls.Config
	used      bool
}

func Tunnel(conn net.Conn, url string, config *TunnelConfig) error {
	defer conn.Close()

	if config.used {
		return ErrUsedTunnelConfig
	}
	config.used = true

	ws, err := websocket_Dial(url, "", "http://", config)
	if err != nil {
		return err
	}

	rxtx(ws, conn)
	return nil
}

// websocket.Dial @ golang.org/x/net/websocket/client.go
func websocket_Dial(url, protocol, origin string, config *TunnelConfig) (ws *websocket.Conn, err error) {
	config_, err := websocket.NewConfig(url, origin)
	if err != nil {
		return nil, err
	}
	if protocol != "" {
		config_.Protocol = []string{protocol}
	}

	config_.TlsConfig = config.TlsConfig

	if config.Proxy == nil {
		return websocket.DialConfig(config_)
	}
	return websocket_DialConfig(config_, config.Proxy)
}

// websocket.DialConfig @ golang.org/x/net/websocket/client.go
func websocket_DialConfig(config *websocket.Config, proxy net.Conn) (ws *websocket.Conn, err error) {
	client := proxy

	if config.Location == nil {
		return nil, &websocket.DialError{config, websocket.ErrBadWebSocketLocation}
	}
	if config.Origin == nil {
		return nil, &websocket.DialError{config, websocket.ErrBadWebSocketOrigin}
	}

	if config.Location.Scheme == "wss" {
		if config.TlsConfig.ServerName == "" {
			config.TlsConfig.ServerName = strings.Split(config.Location.Host, ":")[0]
		}
		client = tls.Client(proxy, config.TlsConfig)
		if err = client.(*tls.Conn).HandshakeContext(context.Background()); err != nil {
			proxy.Close()
			goto Error
		}
	}

	ws, err = websocket.NewClient(config, client)
	if err != nil {
		client.Close()
		goto Error
	}
	return

Error:
	return nil, &websocket.DialError{config, err}
}
