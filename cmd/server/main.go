package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}
func run() error {
	c, err := parseConfig()
	if err != nil {
		return err
	}
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if c.selfCheck {
		return runSelfCheck(c, log)
	}
	rt, err := buildRuntime(c, log)
	if err != nil {
		return err
	}
	done := rt.serve()
	log.Info("服务已启动", "addr", rt.listener.Addr().String(), "data_dir", c.dataDir)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err = <-done:
		if err != nil {
			return err
		}
	case sig := <-signals:
		log.Info("收到关闭信号", "signal", sig.String())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return rt.close(ctx)
}
