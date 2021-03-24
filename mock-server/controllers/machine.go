package controllers

import (
	"mayfly-go/base"
	"mayfly-go/base/biz"
	"mayfly-go/base/ctx"
	"mayfly-go/base/rediscli"
	"mayfly-go/base/utils"
	"mayfly-go/mock-server/machine"
	"mayfly-go/mock-server/models"
	"net/http"

	"github.com/gorilla/websocket"
)

type MachineController struct {
	base.Controller
}

const machineKey = "ccbscf:machines"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024 * 1024 * 10,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (c *MachineController) Machines() {
	c.ReturnData(ctx.NewNoLogReqCtx(true), func(account *ctx.LoginAccount) interface{} {
		return rediscli.HGetAll(machineKey)
	})
}

// 创建机器信息
func (c *MachineController) CreateMachine() {
	c.Operation(ctx.NewNoLogReqCtx(true), func(account *ctx.LoginAccount) {
		machine := &models.Machine{}
		c.UnmarshalBodyAndValid(machine)
		machine.CreateMachine()
	})
}

// @router /api/mock-datas/:method [delete]
func (c *MockController) DeleteMachine() {
	c.Operation(ctx.NewReqCtx(false, "删除mock数据"), func(account *ctx.LoginAccount) {
		models.DeleteMachine(c.Ctx.Input.Param(":ip"))
	})
}

func (c *MachineController) Run() {
	rc := ctx.NewReqCtx(true, "执行机器命令")
	c.ReturnData(rc, func(account *ctx.LoginAccount) interface{} {
		cmd := c.GetString("cmd")
		biz.NotEmpty(cmd, "cmd不能为空")

		rc.ReqParam = cmd

		res, err := c.getCli().Run(cmd)
		biz.BizErrIsNil(err, "执行命令失败")
		return res
	})
}

// 系统基本信息
func (c *MachineController) SysInfo() {
	c.ReturnData(ctx.NewNoLogReqCtx(true), func(account *ctx.LoginAccount) interface{} {
		res, err := c.getCli().GetSystemInfo()
		biz.BizErrIsNil(err, "获取系统基本信息失败")
		return res
	})
}

// top命令信息
func (c *MachineController) Top() {
	c.ReturnData(ctx.NewNoLogReqCtx(true), func(account *ctx.LoginAccount) interface{} {
		return c.getCli().GetTop()
	})
}

func (c *MachineController) GetProcessByName() {
	c.ReturnData(ctx.NewNoLogReqCtx(true), func(account *ctx.LoginAccount) interface{} {
		name := c.GetString("name")
		biz.NotEmpty(name, "name不能为空")
		res, err := c.getCli().GetProcessByName(name)
		biz.BizErrIsNil(err, "获取失败")
		return res
	})
}

func (c *MachineController) WsSSH() {
	wsConn, err := upgrader.Upgrade(c.Ctx.ResponseWriter, c.Ctx.Request, nil)
	if err != nil {
		panic(biz.NewBizErr("获取requst responsewirte错误"))
	}

	cols, _ := c.GetInt("cols", 80)
	rows, _ := c.GetInt("rows", 40)

	sws, err := machine.NewLogicSshWsSession(cols, rows, c.getCli(), wsConn)
	if sws == nil {
		panic(biz.NewBizErr("连接失败"))
	}
	//if wshandleError(wsConn, err) {
	//	return
	//}
	defer sws.Close()

	quitChan := make(chan bool, 3)
	sws.Start(quitChan)
	go sws.Wait(quitChan)

	<-quitChan
}

func (c *MachineController) GetMachineIp() string {
	machineIp := c.Ctx.Input.Param(":ip")
	biz.IsTrue(utils.StrLen(machineIp) > 0, "ip错误")
	return machineIp
}

func (c *MachineController) getCli() *machine.Cli {
	cli, err := machine.GetCli(c.GetMachineIp())
	biz.BizErrIsNil(err, "获取客户端错误")
	return cli
}