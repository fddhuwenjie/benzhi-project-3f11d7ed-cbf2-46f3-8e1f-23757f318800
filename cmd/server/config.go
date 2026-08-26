package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type config struct {
	addr      string
	dataDir   string
	selfCheck bool
}

func parseConfig() (config, error) {
	portDefault := "127.0.0.1:19081"
	if p := os.Getenv("PORT"); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			return config{}, errors.New("PORT 必须是 1 至 65535 的端口号")
		}
		portDefault = net.JoinHostPort("127.0.0.1", p)
	}
	var c config
	flag.StringVar(&c.addr, "addr", portDefault, "回环监听地址")
	flag.StringVar(&c.dataDir, "data-dir", "./data", "SQLite 与证据目录")
	flag.BoolVar(&c.selfCheck, "self-check", false, "执行真实 HTTP 闭环自检后退出")
	flag.Parse()
	if err := validateAddress(c.addr); err != nil {
		return config{}, err
	}
	if strings.TrimSpace(c.dataDir) == "" {
		return config{}, errors.New("data-dir 不能为空")
	}
	return c, nil
}
func validateAddress(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("addr 格式无效: %w", err)
	}
	p, err := strconv.Atoi(port)
	if err != nil || p < 0 || p > 65535 {
		return errors.New("addr 端口无效")
	}
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return errors.New("安全限制：addr 必须使用回环地址")
	}
	return nil
}
