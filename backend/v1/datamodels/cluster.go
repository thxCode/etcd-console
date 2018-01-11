package datamodels

import (
	"github.com/thxcode/etcd-console/backend"
)

// Member Status
type MemberStatus struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Endpoint    string `json:"endpoint"`
	IsLeader    bool   `json:"leader"`
	IsHealth    bool   `json:"health"`
	IsConnected bool   `json:"connected"`
	DBSize      int64  `json:"dbSize"`
	Version     string `json:"version"`
}

// Backup
type Backup struct {
	Name       string           `json:"name"`
	Size       int64            `json:"size"`
	CreateTime backend.JSONTime `json:"createTime"`
}
