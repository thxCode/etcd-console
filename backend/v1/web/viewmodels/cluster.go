package viewmodels

import "github.com/thxcode/etcd-console/backend/v1/datamodels"

type ClusterVersionResponse struct {
	Version string `json:"version"`
	Major   int64  `json:"major"`
	Minor   int64  `json:"minor"`
	Patch   int64  `json:"patch"`
}

type ClusterMemberStatusResponse struct {
	Members []datamodels.MemberStatus `json:"members"`
}

type ClusterBackupResponse struct {
	Backups []datamodels.Backup `json:"backups"`
}
