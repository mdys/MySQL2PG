// Package mpp 提供 MPP 分布式数据库支持
// 支持 Greenplum 和 YugabyteDB 等分布式 PostgreSQL 数据库
package mpp

// Config MPP 配置
type Config struct {
	Enabled  bool   // 是否启用 MPP 模式（启用后才会创建 UNIQUE INDEX）
	Database string // MPP 数据库类型: greenplum/yugabyte/auto
}
