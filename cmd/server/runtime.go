package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"manuscript-conservation-gate/internal/application"
	"manuscript-conservation-gate/internal/evidence"
	"manuscript-conservation-gate/internal/httpapi"
	"manuscript-conservation-gate/internal/sqlstore"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type runtime struct {
	server   *http.Server
	listener net.Listener
	store    *sqlstore.Store
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	x := hex.EncodeToString(b[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", x[:8], x[8:12], x[12:16], x[16:20], x[20:])
}
func buildRuntime(c config, log *slog.Logger) (*runtime, error) {
	if err := os.MkdirAll(c.dataDir, 0700); err != nil {
		return nil, err
	}
	store, err := sqlstore.Open(filepath.Join(c.dataDir, "conservation.db"))
	if err != nil {
		return nil, err
	}
	manager := evidence.New(filepath.Join(c.dataDir, "evidence"))
	service := application.NewService(store, manager, application.RealClock{}, newID)
	handler := httpapi.New(service, log)
	listener, err := net.Listen("tcp", c.addr)
	if err != nil {
		store.Close()
		return nil, err
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10}
	return &runtime{server: server, listener: listener, store: store}, nil
}
func (r *runtime) serve() <-chan error {
	done := make(chan error, 1)
	go func() {
		err := r.server.Serve(r.listener)
		if err == http.ErrServerClosed {
			err = nil
		}
		done <- err
	}()
	return done
}
func (r *runtime) close(ctx context.Context) error {
	err := r.server.Shutdown(ctx)
	closeErr := r.store.Close()
	if err != nil {
		return err
	}
	return closeErr
}
