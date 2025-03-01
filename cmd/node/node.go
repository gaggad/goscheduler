// Command goscheduler-node
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gaggad/goscheduler/internal/modules/rpc/auth"
	"github.com/gaggad/goscheduler/internal/modules/rpc/server"
	"github.com/gaggad/goscheduler/internal/modules/utils"
	"github.com/gaggad/goscheduler/internal/util"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	AppVersion, BuildDate, GitCommit string
)

// getLocalIP 获取本机的IP地址
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logrus.Warnf("Failed to get interface addresses: %v", err)
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return ""
}

func main() {
	app := &cli.App{
		Name:    "goscheduler-node",
		Usage:   "goscheduler node service",
		Version: AppVersion,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "allow-root",
				Usage: "allow running as root user",
			},
			&cli.StringFlag{
				Name:    "server-addr",
				Aliases: []string{"s"},
				Value:   "0.0.0.0:5921",
				Usage:   "server address to listen on",
			},
			&cli.BoolFlag{
				Name:  "enable-tls",
				Usage: "enable TLS",
			},
			&cli.StringFlag{
				Name:  "ca-file",
				Usage: "path to CA file",
			},
			&cli.StringFlag{
				Name:  "cert-file",
				Usage: "path to cert file",
			},
			&cli.StringFlag{
				Name:  "key-file",
				Usage: "path to key file",
			},
			&cli.StringFlag{
				Name:  "log-level",
				Value: "info",
				Usage: "log level (debug|info|warn|error|fatal)",
			},
			&cli.StringFlag{
				Name:    "master-addr",
				Aliases: []string{"m"},
				Value:   "localhost:5920",
				Usage:   "master node address for auto registration",
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func run(c *cli.Context) error {
	level, err := logrus.ParseLevel(c.String("log-level"))
	if err != nil {
		return err
	}
	logrus.SetLevel(level)

	if c.Bool("version") {
		util.PrintAppVersion(AppVersion, GitCommit, BuildDate)
		return nil
	}

	enableTLS := c.Bool("enable-tls")
	if enableTLS {
		caFile := c.String("ca-file")
		certFile := c.String("cert-file")
		keyFile := c.String("key-file")

		if !utils.FileExist(caFile) {
			return cli.Exit("failed to read ca cert file: "+caFile, 1)
		}
		if !utils.FileExist(certFile) {
			return cli.Exit("failed to read server cert file: "+certFile, 1)
		}
		if !utils.FileExist(keyFile) {
			return cli.Exit("failed to read server key file: "+keyFile, 1)
		}
	}

	certificate := auth.Certificate{
		CAFile:   strings.TrimSpace(c.String("ca-file")),
		CertFile: strings.TrimSpace(c.String("cert-file")),
		KeyFile:  strings.TrimSpace(c.String("key-file")),
	}

	if runtime.GOOS != "windows" && os.Getuid() == 0 && !c.Bool("allow-root") {
		return cli.Exit("Do not run goscheduler-node as root user", 1)
	}

	// 获取主机名和IP地址
	hostname, err := os.Hostname()
	if err != nil {
		return cli.Exit("failed to get hostname: "+err.Error(), 1)
	}

	ipAddr := getLocalIP()
	if ipAddr == "" {
		logrus.Warn("Failed to get local IP address, using 127.0.0.1 instead")
		ipAddr = "127.0.0.1"
	}

	// 解析服务器地址
	serverAddr := c.String("server-addr")
	_, port, err := net.SplitHostPort(serverAddr)
	if err != nil {
		return cli.Exit("invalid server address format: "+err.Error(), 1)
	}

	// 自动注册到master节点
	masterAddr := c.String("master-addr")
	// 处理masterAddr，确保包含正确的协议前缀
	if !strings.HasPrefix(masterAddr, "http://") && !strings.HasPrefix(masterAddr, "https://") {
		masterAddr = "http://" + masterAddr
	}
	registerURL := fmt.Sprintf("%s/api/host/register", masterAddr)

	// 从环境变量获取注册密钥
	registerKey := os.Getenv("NODE_REGISTER_KEY")
	if registerKey == "" {
		logrus.Warn("未设置节点注册密钥(NODE_REGISTER_KEY)，节点注册可能会失败")
	}

	portInt, _ := strconv.Atoi(port)
	registerData := map[string]interface{}{
		"name":   ipAddr,
		"alias":  hostname,
		"port":   portInt,
		"remark": "Auto registered node",
		"key":    registerKey, // 添加注册密钥
	}

	jsonData, err := json.Marshal(registerData)
	if err != nil {
		logrus.Warnf("Failed to marshal register data: %v", err)
	} else {
		// 使用指数退避算法进行重试
		registerWithRetry(registerURL, jsonData)
	}

	server.Start(serverAddr, enableTLS, certificate)
	return nil
}

// registerWithRetry 使用指数退避算法进行节点注册重试
func registerWithRetry(registerURL string, jsonData []byte) {
	baseDelay := 1 * time.Second
	maxDelay := 60 * time.Second
	maxRetries := 0 // 0表示无限重试

	var resp *http.Response
	var err error
	retryCount := 0

	for {
		resp, err = http.Post(registerURL, "application/json", bytes.NewBuffer(jsonData))
		if err == nil {
			defer resp.Body.Close()
			// 读取响应内容以便记录错误信息
			body, readErr := io.ReadAll(resp.Body)
			if readErr == nil {
				logrus.Warnf("Registration failed with status code %d : %s", resp.StatusCode, string(body))
			} else {
				logrus.Warnf("Registration failed with status code %d, could not read response: %v", resp.StatusCode, readErr)
			}
			// 检查响应状态码
			if resp.StatusCode >= 200 && resp.StatusCode < 300 && strings.Contains(string(body), `"code":0,`) {
				logrus.Info("Node registration completed successfully ", resp.StatusCode, " ", string(body))
				return
			}
		} else {
			logrus.Warnf("Failed to register node: %v", err)
		}

		retryCount++
		if maxRetries > 0 && retryCount >= maxRetries {
			logrus.Error("Maximum retry attempts reached, giving up registration")
			return
		}

		// 计算下一次重试的延迟时间（指数退避）
		delay := baseDelay * time.Duration(math.Pow(2, float64(retryCount-1)))
		if delay > maxDelay {
			delay = maxDelay
		}

		// 添加一些随机抖动，避免多个节点同时重试
		jitter := time.Duration(rand.Int63n(int64(delay) / 2))
		delay = delay + jitter

		logrus.Infof("Retrying registration in %v (attempt %d)...", delay, retryCount)
		time.Sleep(delay)
	}
}
