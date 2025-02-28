package auth

import (
	"strings"

	"github.com/gaggad/goscheduler/internal/modules/app"
)

func GetNodeRegisterKey() string {
	return strings.TrimSpace(app.Setting.NodeRegisterKey)
}
