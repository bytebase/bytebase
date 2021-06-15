package server

import (
	"context"
	"fmt"
	"time"

	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

const (
	SCHEMA_SYNC_INTERVAL = time.Duration(1) * time.Second
)

func NewSchemaSyncer(logger *zap.Logger, server *Server) *SchemaSyncer {
	return &SchemaSyncer{
		l:      logger,
		server: server,
	}
}

type SchemaSyncer struct {
	l      *zap.Logger
	server *Server
}

func (s *SchemaSyncer) Run() error {
	go func() {
		for {
			time.Sleep(SCHEMA_SYNC_INTERVAL)

			func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
						s.l.Error("Schema syncer PANIC RECOVER", zap.Error(err))
					}
				}()

				rowStatus := api.Normal
				instanceFind := &api.InstanceFind{
					RowStatus: &rowStatus,
				}
				list, err := s.server.InstanceService.FindInstanceList(context.Background(), instanceFind)
				if err != nil {
					s.l.Info(fmt.Sprintf("Failed to retrieve instances: %v\n", err))
				}

				for _, instance := range list {
					s.l.Info("sync instance " + instance.Name)
				}
			}()
		}
	}()

	return nil
}
