package models

import (
	"xorm.io/xorm"
)

// 主机
type Host struct {
	Id        int16  `json:"id" xorm:"smallint pk autoincr"`
	Name      string `json:"name" xorm:"varchar(64) notnull"`                // 主机名称
	Alias     string `json:"alias" xorm:"varchar(32) notnull default '' "`   // 主机别名
	Port      int    `json:"port" xorm:"notnull default 5921"`               // 主机端口
	Remark    string `json:"remark" xorm:"varchar(100) notnull default '' "` // 备注
	Status    Status `json:"status" xorm:"tinyint notnull default 1"`        // 节点状态 0:离线 1:在线
	BaseModel `json:"-" xorm:"-"`
	Selected  bool `json:"-" xorm:"-"`
}

// 新增
func (host *Host) Create() (insertId int16, err error) {
	_, err = Db.Insert(host)
	if err == nil {
		insertId = host.Id
	}

	return
}

func (host *Host) UpdateBean(id int16) (int64, error) {
	return Db.ID(id).Cols("name,alias,port,remark,ip,status").Update(host)
}

// 更新
func (host *Host) Update(id int, data CommonMap) (int64, error) {
	return Db.Table(host).ID(id).Update(data)
}

// 删除
func (host *Host) Delete(id int) (int64, error) {
	return Db.ID(id).Delete(new(Host))
}

func (host *Host) Find(id int) error {
	_, err := Db.ID(id).Get(host)

	return err
}

func (host *Host) NameExists(name string, id int16) (bool, error) {
	if id == 0 {
		count, err := Db.Where("name = ?", name).Count(host)
		return count > 0, err
	}

	count, err := Db.Where("name = ? AND id != ?", name, id).Count(host)
	return count > 0, err
}

// 通过IP查找主机ID
func (host *Host) FindByName(name string) (int, error) {
	has, err := Db.Where("name = ?", name).Get(host)
	if err != nil {
		return 0, err
	}
	if !has {
		return 0, nil
	}
	return int(host.Id), nil
}

// 更新节点状态
func (host *Host) UpdateStatus(id int, status Status) (int64, error) {
	return Db.Table(host).ID(id).Update(CommonMap{"status": status})
}

// 设置节点在线
func (host *Host) SetOnline(id int) (int64, error) {
	return host.UpdateStatus(id, Enabled)
}

// 设置节点离线
func (host *Host) SetOffline(id int) (int64, error) {
	return host.UpdateStatus(id, Disabled)
}

func (host *Host) List(params CommonMap) ([]Host, error) {
	host.parsePageAndPageSize(params)
	list := make([]Host, 0)
	session := Db.Desc("id")
	host.parseWhere(session, params)
	err := session.Limit(host.PageSize, host.pageLimitOffset()).Find(&list)

	return list, err
}

func (host *Host) AllList() ([]Host, error) {
	list := make([]Host, 0)
	err := Db.Cols("name,port").Desc("id").Find(&list)

	return list, err
}

func (host *Host) Total(params CommonMap) (int64, error) {
	session := Db.NewSession()
	host.parseWhere(session, params)
	return session.Count(host)
}

// 解析where
func (host *Host) parseWhere(session *xorm.Session, params CommonMap) {
	if len(params) == 0 {
		return
	}
	id, ok := params["Id"]
	if ok && id.(int) > 0 {
		session.And("id = ?", id)
	}
	name, ok := params["Name"]
	if ok && name.(string) != "" {
		session.And("name = ?", name)
	}
	status, ok := params["Status"]
	if ok && status.(int) > -1 {
		session.And("status = ?", status)
	}
}
