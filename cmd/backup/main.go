package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/snapshot"
	"github.com/gqq/etcd-operator/api/v1alpha1"
	"github.com/gqq/etcd-operator/cmd/backup/pkg/file"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"

	"os"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"time"
)

func loggedError(log logr.Logger, err error, message string) error {
	log.Error(err, message)
	return fmt.Errorf("%s : %s\n", message, err)
}

func main() {
	var (
		backupTempDir          string
		etcdURL                string
		etcdDialTimeoutSeconds int64
		timeoutSeconds         int64
		//bucketname             string
		//objectname             string
		backupUrl string
	)

	flag.StringVar(&backupTempDir, "backup-temp-dir", os.TempDir(), "the directory to temp place backups before they are uploaded to their destination")
	flag.StringVar(&etcdURL, "etcd-url", "http://localhost2379", "url for etcd")
	flag.Int64Var(&etcdDialTimeoutSeconds, "etcd-dial-timeout-seconds", 5, "Timeout,in seconds for dialing the Etcd API")
	flag.Int64Var(&timeoutSeconds, "timeout-seconds", 60, "Timeout for Backup the Etcd.")
	flag.StringVar(&backupUrl, "backup-url", "", "url backup stroage")
	//flag.StringVar(&objectname, "objectname", "snapshot.db", "Remote S3 bucket object name")
	flag.Parse()
	newlogger := zap.NewRaw(zap.UseDevMode(true))
	ctrl.SetLogger(zapr.NewLogger(newlogger))
	logger := ctrl.Log.WithName("backup-agent")
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*time.Duration(timeoutSeconds))
	defer cancelFunc()

	logger.Info("Connecting to Etcd and getting snapshot")
	localpath := filepath.Join(backupTempDir, "snapshot.db")
	storageType, bucketname, objectname, err := file.ParseBackupUrl(backupUrl)
	if err != nil {
		panic(loggedError(logger, err, "failed to parse etcd backup url"))
	}
	etcdClientv3 := snapshot.NewV3(newlogger.Named("etcd-client"))
	err = etcdClientv3.Save(
		ctx,
		clientv3.Config{
			Endpoints:   []string{etcdURL},
			DialTimeout: time.Second * time.Duration(etcdDialTimeoutSeconds),
		},
		localpath,
	)
	if err != nil {
		panic(loggedError(logger, err, "failed to get etcd snapshot"))
	}
	logger.Info("Uploading snapshot...")

	switch storageType {
	case string(v1alpha1.BackupStorageTypeS3): //s3
		s3, err := handleS3(ctx, localpath, bucketname, objectname)
		if err != nil {
			panic(loggedError(logger, err, "failed to upload backup"))
		}
		logger.WithValues("upload-size", s3).Info("Backup complete")
	case string(v1alpha1.BackupStorageTypeOSS): // oss
	default:
		panic(loggedError(logger, fmt.Errorf("storage type error"), fmt.Sprintf("unknow StorageType: %v\n", storageType)))
	}

}

func handleS3(ctx context.Context, localPath, buket, objectname string) (int64, error) {
	endpoint := os.Getenv("ENDPOINT")
	assessKeyID := os.Getenv("MINIO_ACCESS_KEY")
	secretAccessKeyID := os.Getenv("MINIO_SECRET_KEY")
	uploader := file.News3Uploader(endpoint, assessKeyID, secretAccessKeyID)
	uploadsize, err := uploader.Upload(ctx, localPath, buket, objectname)
	return uploadsize, err
}

//package main
//
//import (
//	"fmt"
//	"net/url"
//)
//
//func ParseBackupUrl(backupurl string) (string, string, string, error) {
//	parse, err := url.Parse(backupurl)
//	if err != nil {
//		return "", "", "", err
//	}
//	return parse.Scheme, parse.Host, parse.Path[:], err
//}
//
//func main() {
//	backupUrl := "s3://gqq/edoc2/snapshot.db"
//
//	fmt.Println(ParseBackupUrl(backupUrl))
//}
