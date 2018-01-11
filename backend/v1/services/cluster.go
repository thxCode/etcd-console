package services

import (
	"github.com/thxcode/etcd-console/backend/v1/datamodels"
	"github.com/thxcode/etcd-console/backend"
	"github.com/kataras/iris"
	sv2 "github.com/Masterminds/semver"
	v3 "github.com/coreos/etcd/clientv3"
	"context"
	"time"
	"fmt"
	"sync"
	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	"errors"
	"path/filepath"
	"io/ioutil"
	"archive/zip"
	"os"
	"io"
	"github.com/coreos/etcd/pkg/fileutil"
	"crypto/sha1"
)

type ClusterService interface {
	GetVersion(ctx context.Context, irisCtx iris.Context) *sv2.Version
	GetStatuses(ctx context.Context, irisCtx iris.Context) ([]datamodels.MemberStatus, error)
	GetBackups(ctx context.Context, irisCtx iris.Context) ([]datamodels.Backup, error)
	NewBackup(ctx context.Context, irisCtx iris.Context) (datamodels.Backup, error)
	DelBackup(ctx context.Context, irisCtx iris.Context) error
	DownloadBackup(ctx context.Context, irisCtx iris.Context) error
}

type clusterService struct {
}

func NewClusterService() ClusterService {
	return &clusterService{
	}
}

func (c *clusterService) GetVersion(ctx context.Context, irisCtx iris.Context) *sv2.Version {
	etcdClient := irisCtx.Values().Get("etcd-console.client").(*backend.EtcdClient)

	return etcdClient.Version()
}

func (c *clusterService) GetStatuses(ctx context.Context, irisCtx iris.Context) ([]datamodels.MemberStatus, error) {
	etcdClient := irisCtx.Values().Get("etcd-console.client").(*backend.EtcdClient)

	timeout, err := irisCtx.URLParamInt64Default("timeout", 5)
	if err != nil {
		timeout = 5
	}
	timeoutCtx, timeoutCancelFn := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer timeoutCancelFn()

	var retMemberStatuses []datamodels.MemberStatus

	version := etcdClient.Version()
	if version.Major() == 2 {
		return nil, errors.New("cannot support v2 now")
	} else {
		client, err := etcdClient.V3()
		if err != nil {
			return nil, err
		}

		memberListRep, err := client.MemberList(timeoutCtx)
		if err != nil {
			return nil, err
		}

		if memberSize := len(memberListRep.Members); memberSize > 0 {
			var (
				wg               = &sync.WaitGroup{}
				memberStatusChan = make(chan datamodels.MemberStatus, memberSize)
			)

			for _, member := range memberListRep.Members {
				wg.Add(1)

				go func(memberEndpoint string, memberName string, memberId uint64) {
					defer wg.Done()

					memberStatus := datamodels.MemberStatus{
						Endpoint: memberEndpoint,
						ID:       fmt.Sprintf("%x", memberId),
						Name:     memberName,
					}

					statusRep, err := client.Status(timeoutCtx, memberEndpoint)
					if err == nil {
						memberStatus.IsLeader = memberId == statusRep.Leader
						memberStatus.IsConnected = true
						memberStatus.Version = statusRep.Version
						memberStatus.DBSize = statusRep.DbSize

						epClient, err := v3.New(v3.Config{
							Endpoints:   []string{memberEndpoint},
							DialTimeout: 5 * time.Second,
						})
						if err == nil {
							_, err = epClient.Get(timeoutCtx, "health")
							if err == nil || err == rpctypes.ErrPermissionDenied {
								memberStatus.IsHealth = true
							}
						}
					}

					memberStatusChan <- memberStatus
				}(member.ClientURLs[0], member.Name, member.ID)
			}

			wg.Wait()
			close(memberStatusChan)

			for memberStatus := range memberStatusChan {
				retMemberStatuses = append(retMemberStatuses, memberStatus)
			}
		}

	}

	return retMemberStatuses, nil
}

func (c *clusterService) GetBackups(ctx context.Context, irisCtx iris.Context) ([]datamodels.Backup, error) {
	etcdClient := irisCtx.Values().Get("etcd-console.client").(*backend.EtcdClient)
	configuration := irisCtx.Values().Get("etcd-console.config").(backend.Configuration)

	timeout, err := irisCtx.URLParamInt64Default("timeout", 15)
	if err != nil {
		timeout = 15
	}
	timeoutCtx, timeoutCancelFn := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer timeoutCancelFn()

	var retBackups []datamodels.Backup

	backupDir := configuration.BackupDir
	backupZips, err := ioutil.ReadDir(backupDir)
	if err != nil {
		return nil, err
	}

	version := etcdClient.Version()
	if version.Major() == 2 {
		return nil, errors.New("cannot support v2 now")
	} else {
		var (
			wg                = &sync.WaitGroup{}
			backupZipsSyncMap = &sync.Map{}
		)

		for _, backupZip := range backupZips {
			if !backupZip.IsDir() {
				wg.Add(1)

				go func(ctx context.Context, backupZip os.FileInfo) {
					defer wg.Done()

					backupZipName := backupZip.Name()
					backupPath := filepath.Join(backupDir, backupZipName)

					backup, err := zip.OpenReader(backupPath)
					if err == nil {
						modTime := backupZip.ModTime()

						backupZipsSyncMap.Store(modTime.Unix(), datamodels.Backup{
							Name:       backupZipName,
							Size:       backupZip.Size(),
							CreateTime: backend.JSONTime(modTime),
						})
					}
					defer backup.Close()
				}(timeoutCtx, backupZip)
			}
		}

		wg.Wait()

		backupZipsSyncMap.Range(func(key, value interface{}) bool {
			retBackups = append(retBackups, value.(datamodels.Backup))
			return true
		})

	}

	return retBackups, nil
}

func (c *clusterService) NewBackup(ctx context.Context, irisCtx iris.Context) (datamodels.Backup, error) {
	etcdClient := irisCtx.Values().Get("etcd-console.client").(*backend.EtcdClient)
	configuration := irisCtx.Values().Get("etcd-console.config").(backend.Configuration)

	timeout, err := irisCtx.URLParamInt64Default("timeout", 30)
	if err != nil {
		timeout = 30
	}
	timeoutCtx, timeoutCancelFn := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer timeoutCancelFn()

	var retBackup datamodels.Backup

	backupDir := configuration.BackupDir
	if stat, err := os.Stat(backupDir); err != nil {
		return retBackup, err
	} else if !stat.IsDir() {
		return retBackup, errors.New("bakcup dir is lost")
	}

	version := etcdClient.Version()
	if version.Major() == 2 {
		return retBackup, errors.New("cannot support v2 now")
	} else {
		client, err := etcdClient.V3()
		if err != nil {
			return retBackup, err
		}

		maintenance := v3.NewMaintenance(client)

		// snapshot stores
		snapshotReader, err := maintenance.Snapshot(timeoutCtx)
		if err != nil {
			return retBackup, err
		}
		snapshotName := fmt.Sprintf("%x", sha1.Sum([]byte(fmt.Sprintf("%d-snapshot", time.Now().UnixNano()))))
		snapshotTmpPath := filepath.Join(os.TempDir(), snapshotName)
		snapshotTmpFile, err := os.Create(snapshotTmpPath)
		if err != nil {
			irisCtx.Application().Logger().Error(err)
			return retBackup, errors.New("cannot create backup")
		}
		if _, err := io.Copy(snapshotTmpFile, snapshotReader); err != nil {
			irisCtx.Application().Logger().Error(err)
			return retBackup, errors.New("cannot create backup")
		}
		fileutil.Fsync(snapshotTmpFile)
		snapshotReader.Close()
		//snapshotTmpFile.Close()

		// snapshot packages
		retBackupPath := filepath.Join(backupDir, fmt.Sprintf("%s.zip", snapshotName))
		snapshotZipFile, err := os.Create(retBackupPath)
		if err != nil {
			irisCtx.Application().Logger().Error(err)
			return retBackup, errors.New("cannot create backup")
		}
		snapshotArchiveWriter := zip.NewWriter(snapshotZipFile)
		//snapshotTmpFileReOpen, err := os.Open(snapshotTmpPath)
		//if err != nil {
		//	irisCtx.Application().Logger().Error(err)
		//	return retBackupName, errors.New("cannot create backup")
		//}
		//snapshotTmpFileInfo, err := snapshotTmpFileReOpen.Stat()
		snapshotTmpFileInfo, err := snapshotTmpFile.Stat()
		if err != nil {
			irisCtx.Application().Logger().Error(err)
			return retBackup, errors.New("cannot create backup")
		}
		snapshotZipFileHeader, err := zip.FileInfoHeader(snapshotTmpFileInfo)
		if err != nil {
			irisCtx.Application().Logger().Error(err)
			return retBackup, errors.New("cannot create backup")
		}
		snapshotZipFileWriter, err := snapshotArchiveWriter.CreateHeader(snapshotZipFileHeader)
		if err != nil {
			irisCtx.Application().Logger().Error(err)
			return retBackup, errors.New("cannot create backup")
		}
		if _, err := io.Copy(snapshotZipFileWriter, snapshotTmpFile); err != nil {
			irisCtx.Application().Logger().Error(err)
			return retBackup, errors.New("cannot create backup")
		}
		snapshotArchiveWriter.Close()
		snapshotZipFile.Close()
		snapshotTmpFile.Close()

		os.Remove(snapshotTmpPath)

		snapshotZipFileStat, err := os.Stat(retBackupPath)
		if err != nil {
			return retBackup, err
		} else {
			retBackup = datamodels.Backup{
				Name:       snapshotZipFileStat.Name(),
				Size:       snapshotZipFileStat.Size(),
				CreateTime: backend.JSONTime(snapshotZipFileStat.ModTime()),
			}
		}
	}

	return retBackup, nil
}

func (c *clusterService) DelBackup(ctx context.Context, irisCtx iris.Context) error {
	etcdClient := irisCtx.Values().Get("etcd-console.client").(*backend.EtcdClient)
	configuration := irisCtx.Values().Get("etcd-console.config").(backend.Configuration)

	backupDir := configuration.BackupDir
	if stat, err := os.Stat(backupDir); err != nil {
		irisCtx.Application().Logger().Error(err)
		return errors.New("bakcup dir is lost")
	} else if !stat.IsDir() {
		return errors.New("bakcup dir is lost")
	}

	version := etcdClient.Version()
	if version.Major() == 2 {
		return errors.New("cannot support v2 now")
	} else {
		name := irisCtx.URLParamEscape("name")
		if name == "" {
			return errors.New("name is required")
		}

		backupZipPath := filepath.Join(backupDir, name)
		backupZip, err := os.Stat(backupZipPath)
		if err != nil {
			irisCtx.Application().Logger().Error(err)
			return errors.New("cannot find backup")
		} else if backupZip.IsDir() {
			return errors.New("cannot find backup")
		}

		if err := os.Remove(backupZipPath); err != nil {
			irisCtx.Application().Logger().Error(err)
			return errors.New("cannot remove backup")
		}
	}

	return nil
}

func (c *clusterService) DownloadBackup(ctx context.Context, irisCtx iris.Context) error {
	etcdClient := irisCtx.Values().Get("etcd-console.client").(*backend.EtcdClient)
	configuration := irisCtx.Values().Get("etcd-console.config").(backend.Configuration)

	backupDir := configuration.BackupDir
	if stat, err := os.Stat(backupDir); err != nil {
		irisCtx.Application().Logger().Error(err)
		return errors.New("bakcup dir is lost")
	} else if !stat.IsDir() {
		return errors.New("bakcup dir is lost")
	}

	version := etcdClient.Version()
	if version.Major() == 2 {
		return errors.New("cannot support v2 now")
	} else {
		name := irisCtx.URLParamEscape("name")
		if name == "" {
			return errors.New("name is required")
		}

		backupZipPath := filepath.Join(backupDir, name)
		backupZip, err := os.Open(backupZipPath)
		if err != nil {
			irisCtx.Application().Logger().Error(err)
			return errors.New("cannot find backup")
		} else if stat, err := backupZip.Stat(); err != nil {
			irisCtx.Application().Logger().Error(err)
			return errors.New("cannot find backup")
		} else if stat.IsDir() {
			return errors.New("cannot find backup")
		}
		defer backupZip.Close()

		if _, err := io.Copy(irisCtx.ResponseWriter(), backupZip); err != nil {
			irisCtx.Application().Logger().Error(err)
			return errors.New("cannot find backup")
		}

	}

	return nil
}
