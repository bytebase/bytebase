package lsp

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"

	wsjsonrpc2 "github.com/sourcegraph/jsonrpc2/websocket"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/common/stacktrace"
)

func (s *Server) ConfigLSPRouters(
	ctx context.Context,
) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	newHandler := func() (jsonrpc2.Handler, io.Closer) {
		return NewHandler(), io.NopCloser(strings.NewReader(""))
	}

	s.server = &http.Server{
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/lsp", func(w http.ResponseWriter, r *http.Request) {
		connection, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("Failed to upgrade websocket connection", log.BBError(errors.Errorf("errors: %v\n%s", err, stacktrace.TakeStacktrace(20 /* n */, 5 /* skip */))))
			http.Error(w, errors.Wrap(err, "could not upgrade to WebSocket").Error(), http.StatusBadRequest)
			return
		}
		defer connection.Close()
		// Register the connection to be closed when the server shuts down.
		s.server.RegisterOnShutdown(func() {
			err := connection.Close()
			if err != nil {
				slog.Error("Failed to close websocket connection", log.BBError(err))
			}
		})
		connectionID := s.connectionCount.Add(1)

		slog.Debug("New LSP connection", slog.Uint64("connectionID", connectionID))
		handler, closer := newHandler()
		<-jsonrpc2.NewConn(ctx, wsjsonrpc2.NewObjectStream(connection), handler, nil /* connOpt */).DisconnectNotify()
		err = closer.Close()
		if err != nil {
			slog.Error("Failed to close LSP connection", slog.Uint64("connectionID", connectionID), log.BBError(errors.Errorf("errors: %v\n%s", err, stacktrace.TakeStacktrace(20 /* n */, 5 /* skip */))))
		}
		slog.Debug("LSP connection closed", slog.Uint64("connectionID", connectionID))
	})

	s.server.Handler = mux
}
