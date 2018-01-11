package routes

import (
	"github.com/kataras/iris"
	"context"
	"github.com/thxcode/etcd-console/backend/v1/services"
	"github.com/thxcode/etcd-console/backend/v1/web/viewmodels"
	"github.com/kataras/iris/hero"
	"github.com/thxcode/etcd-console/backend/v1/datamodels"
	"sort"
	"strings"
)

type backupSlice []datamodels.Backup

func (bs backupSlice) Len() int {
	return len(bs)
}

func (bs backupSlice) Less(i, j int) bool {
	return bs[i].CreateTime.Unix() > bs[j].CreateTime.Unix()
}

func (bs backupSlice) Swap(i, j int) {
	bs[i], bs[j] = bs[j], bs[i]
}


type memberStatusSlice []datamodels.MemberStatus

func (ms memberStatusSlice) Len() int {
	return len(ms)
}

func (ms memberStatusSlice) Less(i, j int) bool {
	return strings.Compare(ms[i].Name, ms[j].Name) < 0
}

func (ms memberStatusSlice) Swap(i, j int) {
	ms[i], ms[j] = ms[j], ms[i]
}


func Cluster(irisCtx iris.Context, service services.ClusterService, op string) hero.Result {
	var (
		response      = hero.Response{}
		rootCtx       = irisCtx.Values().Get("etcd-console.ctx").(context.Context)
		requestMethod = irisCtx.Method()
	)

	switch op {
	case "version":
		version := service.GetVersion(rootCtx, irisCtx)
		response.Object = viewmodels.ClusterVersionResponse{
			Version: version.String(),
			Major:   version.Major(),
			Minor:   version.Minor(),
			Patch:   version.Patch(),
		}
	case "status":
		members, err := service.GetStatuses(rootCtx, irisCtx)
		if err != nil {
			irisCtx.Application().Logger().Error(err)

			response.Code = iris.StatusInternalServerError
			response.Err = err
		} else {
			memberStatusSlices := memberStatusSlice(members)
			sort.Sort(memberStatusSlices)

			response.Object = viewmodels.ClusterMemberStatusResponse{
				Members: []datamodels.MemberStatus(memberStatusSlices),
			}
		}
	case "backup":
		switch requestMethod {
		case iris.MethodGet:
			if irisCtx.URLParamExists("name") {
				err := service.DownloadBackup(rootCtx, irisCtx)
				if err != nil {
					irisCtx.Application().Logger().Error(err)

					response.Code = iris.StatusInternalServerError
					response.Err = err
				}
			} else {
				backups, err := service.GetBackups(rootCtx, irisCtx)
				if err != nil {
					irisCtx.Application().Logger().Error(err)

					response.Code = iris.StatusInternalServerError
					response.Err = err
				} else {
					backupSlices := backupSlice(backups)
					sort.Sort(backupSlices)

					response.Object = viewmodels.ClusterBackupResponse{
						Backups: []datamodels.Backup(backupSlices),
					}
				}
			}
		case iris.MethodDelete:
			err := service.DelBackup(rootCtx, irisCtx)
			if err != nil {
				irisCtx.Application().Logger().Error(err)

				response.Code = iris.StatusInternalServerError
				response.Err = err
			}
		case iris.MethodPost:
			backup, err := service.NewBackup(rootCtx, irisCtx)
			if err != nil {
				irisCtx.Application().Logger().Error(err)

				response.Code = iris.StatusInternalServerError
				response.Err = err
			} else {
				response.Object = viewmodels.ClusterBackupResponse{
					Backups: []datamodels.Backup{backup},
				}
			}
		}
	}

	return response
}
