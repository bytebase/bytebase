package lsp

import (
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"

	wsjsonrpc2 "github.com/sourcegraph/jsonrpc2/websocket"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/common/stacktrace"
	"github.com/bytebase/bytebase/backend/store"
)

var (
	upgrader   = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	newHandler = func(s *store.Store) (jsonrpc2.Handler, io.Closer) {
		return NewHandler(s), io.NopCloser(strings.NewReader(""))
	}
)

func (s *Server) Router(c echo.Context) error {
	connection, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("Failed to upgrade websocket connection", log.BBError(errors.Errorf("errors: %v\n%s", err, stacktrace.TakeStacktrace(20 /* n */, 5 /* skip */))))
		return errors.Wrap(err, "could not upgrade to WebSocket")
	}
	defer connection.Close()
	// Register the connection to be closed when the server shuts down.
	c.Echo().Server.RegisterOnShutdown(func() {
		err := connection.Close()
		if err != nil {
			slog.Error("Failed to close websocket connection", log.BBError(err))
		}
	})
	connectionID := s.connectionCount.Add(1)

	handler, closer := newHandler(s.store)
	ctx := c.Request().Context()
	<-jsonrpc2.NewConn(ctx, wsjsonrpc2.NewObjectStream(connection), handler, nil /* connOpt */).DisconnectNotify()
	err = closer.Close()
	if err != nil {
		return errors.Wrapf(err, "failed to close LSP connection: %v", connectionID)
	}
	return nil
}
