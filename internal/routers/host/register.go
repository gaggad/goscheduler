package host

import (
	"strings"

	"github.com/gaggad/goscheduler/internal/models"
	"github.com/gaggad/goscheduler/internal/modules/logger"
	"github.com/gaggad/goscheduler/internal/modules/rpc/auth"
	"github.com/gaggad/goscheduler/internal/modules/utils"
	"github.com/go-macaron/binding"
	macaron "gopkg.in/macaron.v1"
)

type RegisterForm struct {
	Name   string `binding:"Required;MaxSize(64)"`
	Alias  string `binding:"Required;MaxSize(32)"`
	Port   int    `binding:"Required;Range(1-65535)"`
	Remark string
	Key    string // 节点注册密钥
}

// Error 表单验证错误处理
func (f RegisterForm) Error(ctx *macaron.Context, errs binding.Errors) {
	if len(errs) == 0 {
		return
	}
	json := utils.JsonResponse{}
	content := json.CommonFailure("表单验证失败, 请检测输入")
	ctx.Write([]byte(content))
}

// Register 节点自动注册接口
func Register(ctx *macaron.Context, form RegisterForm) string {
	// 验证注册密钥
	expectedKey := auth.GetNodeRegisterKey()
	// if expectedKey == "" {
	// 	json := utils.JsonResponse{}
	// 	return json.CommonFailure("节点注册功能未配置密钥，请先配置环境变量 NODE_REGISTER_KEY")
	// }

	if form.Key != expectedKey {
		json := utils.JsonResponse{}
		return json.CommonFailure("节点注册密钥验证失败")
	}
	json := utils.JsonResponse{}

	hostModel := new(models.Host)
	// 检查IP是否已存在
	ipExist, err := hostModel.NameExists(form.Name, 0)
	if err != nil {
		return json.CommonFailure("操作失败", err)
	}
	if ipExist {
		// 如果IP已存在，更新该节点信息
		hostId, err := hostModel.FindByName(form.Name)
		if err != nil {
			return json.CommonFailure("操作失败", err)
		}

		hostModel.Name = strings.TrimSpace(form.Name)
		hostModel.Alias = strings.TrimSpace(form.Alias)
		hostModel.Port = form.Port
		hostModel.Remark = strings.TrimSpace(form.Remark)
		hostModel.Status = models.Enabled // 设置节点状态为在线

		_, err = hostModel.UpdateBean(int16(hostId))
		if err != nil {
			return json.CommonFailure("更新节点信息失败", err)
		}

		logger.Infof("节点信息更新成功 [id: %d name: %s]", hostId, form.Name)
		return json.Success("更新成功", map[string]interface{}{
			"id": hostId,
		})
	}

	// 如果IP不存在，检查主机名是否存在
	nameExist, err := hostModel.NameExists(form.Name, 0)
	if err != nil {
		return json.CommonFailure("操作失败", err)
	}
	if nameExist {
		return json.CommonFailure("主机名已存在")
	}

	hostModel.Name = strings.TrimSpace(form.Name)
	hostModel.Alias = strings.TrimSpace(form.Alias)
	hostModel.Port = form.Port
	hostModel.Remark = strings.TrimSpace(form.Remark)
	hostModel.Status = models.Enabled // 设置节点状态为在线

	id, err := hostModel.Create()
	if err != nil {
		return json.CommonFailure("注册失败", err)
	}

	logger.Infof("新节点注册成功 [id: %d name: %s]", id, form.Name)
	return json.Success("注册成功", map[string]interface{}{
		"id": id,
	})
}
