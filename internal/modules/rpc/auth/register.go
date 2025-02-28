package auth

import (
	"os"
	"strings"
)

const (
	// NodeRegisterKeyEnv 节点注册密钥的环境变量名
	NodeRegisterKeyEnv = "NODE_REGISTER_KEY"
)

// GetNodeRegisterKey 获取节点注册密钥
func GetNodeRegisterKey() string {
	return strings.TrimSpace(os.Getenv(NodeRegisterKeyEnv))
}
