package auth

import (
	"os"
	"strings"
)

func GetNodeRegisterKey() string {
	return strings.TrimSpace(os.Getenv("NODE_REGISTER_KEY"))
}
