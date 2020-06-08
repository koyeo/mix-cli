package mysql

import (
	"fmt"
	"github.com/urfave/cli"
	mysql2 "github.com/koyeo/mix-cli/database/mysql"
	"github.com/koyeo/snippet/logger"
	"strconv"
	"strings"
)

type varRow struct {
	Key   string `xorm:"Variable_name"`
	Value string `xorm:"Value"`
}

type varRowsMap map[string]string

type blockPidRow struct {
	WaitingPid           int    `xorm:"waiting_pid"`
	WaitingQuery         string `xorm:"waiting_query"`
	BlockingPid          int    `xorm:"blocking_pid"`
	BlockingQuery        string `xorm:"blocking_query"`
	WaitAge              int    `xorm:"wait_age"`
	SqlKillBlockingQuery string `xorm:"sql_kill_blocking_query"`
}

type slowQueryRow struct {
	Id      int64  `xorm:"id"`
	User    string `xorm:"user"`
	Host    string `xorm:"host"`
	DB      string `xorm:"db"`
	Command string `xorm:"command"`
	Time    int64  `xorm:"time"`
	State   string `xorm:"state"`
	Info    string `xorm:"info"`
}

type monitor struct {
	handler     *Handler
	monitorData *monitorData
	db          *mysql2.MySQL
}

func (p *Handler) MonitorCommand(ctx *cli.Context) (err error) {

	p.loadConfig()

	daemonMode := ctx.Bool(DAEMON)
	if daemonMode {
		return
	}

	m := monitor{
		handler:     p,
		monitorData: newMonitorData(),
	}

	return m.run(ctx)
}

func (p *monitor) run(ctx *cli.Context) (err error) {

	connectionName := ctx.String("connection")

	if _, ok := p.handler.config.Connections[connectionName ]; !ok {
		logger.Error(fmt.Sprintf("Connection %s not defined", connectionName), nil)
		return
	}
	dsn := strings.TrimSuffix(p.handler.config.Connections[connectionName], "/") + "/mysql"
	p.db, err = mysql2.NewMySQL(dsn)
	if err != nil {
		logger.Error("", err)
		return
	}
	defer p.db.Close()

	p.setMonitorData()
	monitorDashboard := newMonitorDashboard(p)
	monitorDashboard.run()

	return
}

func (p *monitor) validateMonitorArgs(ctx *cli.Context) (name string, err error) {
	name = ctx.Args().First()
	return
}

func (p *monitor) setMonitorData() {

	globalStatusRows, err := p.getDbGlobalStatus()
	if err != nil {
		logger.Error("", err)
		return
	}

	// 1. 设置连接数
	p.monitorData.mu.Lock()
	defer p.monitorData.mu.Unlock()

	p.monitorData.Connections = p.getVarThreadsConnected(globalStatusRows)
	p.monitorData.MaxConnections = 151
	p.monitorData.Concurrences = p.getVarThreadsRunning(globalStatusRows)
	p.monitorData.CacheHitRate = p.getCacheHitRate(globalStatusRows)

	uptime := p.getVarUptime(globalStatusRows)
	queries := p.getVarQueries(globalStatusRows)
	tc := p.getTC(globalStatusRows)
	bytesReceived := p.getVarBytesReceived(globalStatusRows)
	bytesSent := p.getVarBytesSent(globalStatusRows)

	p.monitorData.QPS = p.getQPS(queries, p.monitorData.Queries, uptime, p.monitorData.Uptime)
	p.monitorData.TPS = p.getTPS(tc, p.monitorData.TC, uptime, p.monitorData.Uptime)
	p.monitorData.Input = p.getIO(bytesReceived, p.monitorData.BytesReceived, uptime, p.monitorData.Uptime)
	p.monitorData.Output = p.getIO(bytesSent, p.monitorData.BytesSent, uptime, p.monitorData.Uptime)
	p.monitorData.Queries = queries
	p.monitorData.TC = tc
	p.monitorData.Uptime = uptime

	return
}

func (p *monitor) getMonitorData() *monitorData {

	p.monitorData.mu.RLock()
	defer p.monitorData.mu.RUnlock()

	return p.monitorData
}

func (p *monitor) getDbStatus() (r varRowsMap, err error) {

	var rows []*varRow
	err = p.db.Engine.SQL(fmt.Sprintf("show status")).Find(&rows)
	if err != nil {
		logger.Error("", err)
		return
	}

	r = p.getVarsMap(rows)

	return
}

func (p *monitor) getDbGlobalStatus() (r varRowsMap, err error) {

	var rows []*varRow
	err = p.db.Engine.SQL("show global status").Find(&rows)
	if err != nil {
		logger.Error("", err)
		return
	}

	r = p.getVarsMap(rows)

	return
}

func (p *monitor) getVarsMap(rows []*varRow) varRowsMap {

	rowsMap := make(map[string]string)

	for _, v := range rows {
		rowsMap[v.Key] = v.Value
	}

	return rowsMap
}

func (p *monitor) getVarQueries(varsMap varRowsMap) (r int64) {

	r, _ = strconv.ParseInt(varsMap["Queries"], 10, 64)
	return
}

func (p *monitor) getVarUptime(varsMap varRowsMap) (r int64) {
	r, _ = strconv.ParseInt(varsMap["Uptime"], 10, 64)
	return
}

func (p *monitor) getTC(varsMap varRowsMap) (r int64) {
	return p.getVarComInsert(varsMap) + p.getVarComUpdate(varsMap) + p.getVarComDelete(varsMap)
}

func (p *monitor) getVarComInsert(varsMap varRowsMap) (r int64) {
	r, _ = strconv.ParseInt(varsMap["Com_insert"], 10, 64)
	return
}

func (p *monitor) getVarComDelete(varsMap varRowsMap) (r int64) {
	r, _ = strconv.ParseInt(varsMap["Com_delete"], 10, 64)
	return
}

func (p *monitor) getVarComUpdate(varsMap varRowsMap) (r int64) {
	r, _ = strconv.ParseInt(varsMap["Com_delete"], 10, 64)
	return
}

func (p *monitor) getVarBytesReceived(varsMap varRowsMap) (r int64) {
	r, _ = strconv.ParseInt(varsMap["Bytes_received"], 10, 64)
	return
}

func (p *monitor) getVarBytesSent(varsMap varRowsMap) (r int64) {
	r, _ = strconv.ParseInt(varsMap["Bytes_sent"], 10, 64)
	return
}

func (p *monitor) getIO(bytes2, bytes1, uptime2, uptime1 int64) (r float64) {
	if uptime2-uptime1 == 0 {
		return 0
	}
	return float64(bytes2-bytes1) / float64(uptime2-uptime1) / 1000
}

func (p *monitor) getQPS(queries2, queries1, uptime2, uptime1 int64) (r float64) {
	if uptime2-uptime1 == 0 {
		return 0
	}
	return float64(queries2-queries1) / float64(uptime2-uptime1)
}

func (p *monitor) getTPS(tc2, tc1, uptime2, uptime1 int64) (r float64) {
	if uptime2-uptime1 == 0 {
		return 0
	}
	return float64(tc2-tc1) / float64(uptime2-uptime1)
}

func (p *monitor) getVarThreadsRunning(varsMap varRowsMap) (r int64) {
	r, _ = strconv.ParseInt(varsMap["Threads_running"], 10, 64)
	return
}

func (p *monitor) getVarThreadsConnected(varsMap varRowsMap) (r int64) {
	r, _ = strconv.ParseInt(varsMap["Threads_connected"], 10, 64)
	return
}

func (p *monitor) getCacheHitRate(varsMap varRowsMap) float64 {
	r1, _ := strconv.ParseInt(varsMap["Innodb_buffer_pool_read_requests"], 10, 64)
	r2, _ := strconv.ParseInt(varsMap["Innodb_buffer_pool_reads"], 10, 64)
	return (float64(r1) - float64(r2)) / float64(r1) * 100
}

func (p *monitor) getBlockQueries() (rows []*blockPidRow) {

	err := p.db.Engine.SQL(
		fmt.Sprintf(`select waiting_pid,waiting_query, blocking_pid, blocking_query, wait_age, sql_kill_blocking_query from sys.innodb_lock_waits where( unix_timestamp() - unix_timestamp(wait_started)) > 30`)).
		Find(&rows)
	if err != nil {
		logger.Error("Exec sql error", err)
		return
	}

	return
}

func (p *monitor) getSlowQueries() (rows []*slowQueryRow) {

	err := p.db.Engine.SQL(
		fmt.Sprintf("select * from information_schema.processlist where time>60 and command <> 'sleep';")).
		Find(&rows)
	if err != nil {
		logger.Error("Exec sql error", err)
		return
	}

	return
}
