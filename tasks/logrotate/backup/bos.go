package backup

// Backup log files via Baidu Object Storage
// Configs:
//     mode: "bos"
//     remote: bos endpoint
// 	   bucket: bos bucket name
// 	   access_key: bos accessKey
//     access_secret: bos accessSecret

import (
	"fmt"
	"path/filepath"
	"runtime"
	"runtime/debug"

	"github.com/Laisky/go-utils"
	"github.com/Laisky/zap"
	"github.com/baidubce/bce-sdk-go/bce"
	"github.com/baidubce/bce-sdk-go/services/bos"
	"github.com/pkg/errors"
)

var (
	uploadChunkSize  int64 = 1000 * 1024 * 1024
	uploadConcurrent int64 = 2
)

type BosArgs struct {
	Remote       string // "https://gz.bcebos.com"
	Bucket       string
	AccessKey    string
	AccessSecret string
}

type bosUploader struct {
	*baseUploader
	args *BosArgs
	cli  *bos.Client
}

func (u *bosUploader) New(st *backupSetting) error {
	u.baseUploader = createBaseUploader(st)
	u.args = &BosArgs{
		Remote:       st.Args["remote"].(string),
		AccessKey:    st.Args["access_key"].(string),
		AccessSecret: st.Args["access_secret"].(string),
		Bucket:       st.Args["bucket"].(string),
	}

	// connect
	var err error
	u.cli, err = Connect2bos(u.args.Remote, u.args.AccessKey, u.args.AccessSecret)
	if err != nil {
		return errors.Wrap(err, "try to connect bos error")
	}

	return nil
}

// loadRemoteFileLength load file length from bos
// return 0 if file not exists
func (u *bosUploader) loadRemoteFileLength(objName string) int64 {
	meta, err := u.cli.GetObjectMeta(u.args.Bucket, objName)
	if realErr, ok := err.(*bce.BceServiceError); ok {
		if realErr.StatusCode == 404 {
			return 0
		}
	}
	return meta.ContentLength
}

func (u *bosUploader) Upload(fpath string) {
	utils.Logger.Debug("uploading file...", zap.String("fpath", fpath))
	defer u.Done()

	if utils.Settings.GetBool("dry") {
		utils.Logger.Debug("upload %v", zap.String("fpath", fpath))
		return
	}

	var (
		objName       string
		localFileSize int64
		err           error
		r             = ""
	)

	if localFileSize, err = u.CheckIsFileReady(fpath); err != nil {
		utils.Logger.Error("try to get file info error", zap.Error(err))
		u.AddFaiFile(fpath)
		return
	}

	objName = u.getObjFname(fpath)
	if remoteFileLen := u.loadRemoteFileLength(objName); remoteFileLen != 0 {
		if localFileSize < remoteFileLen {
			utils.Logger.Warn("will discard local file since of remote already exists", zap.String("file", objName))
			// remove local file if local file is smaller than remote
			// TODO: download remote file and merge into local size, then upload the new file
			u.AddSucFile(fpath)
			return
		}
		utils.Logger.Info("will replace remote by local file", zap.String("file", objName))
	}

	if localFileSize < 1024*1024*1024 {
		r, err = u.cli.PutObjectFromFile(u.args.Bucket, objName, fpath, nil) // upload single file
	} else { // file size must greater than 5 MB
		err = u.cli.UploadSuperFile(u.args.Bucket, objName, fpath, "") // upload by multipart
	}
	if err != nil {
		utils.Logger.Error("upload file got error", zap.Error(err))
		u.AddFaiFile(fpath)
		return
	}

	if u.loadRemoteFileLength(objName) == 0 { // double check after uploading
		u.AddFaiFile(fpath)
		utils.Logger.Error("file not exists after upload")
		return
	}

	u.AddSucFile(fpath)
	utils.Logger.Info("success uploaded file", zap.String("fpath", fpath), zap.String("result", r))
}

func (u *bosUploader) Clean() {
	u.CleanFiles()
	go func() {
		runtime.GC() // bos taken too much memory
		debug.FreeOSMemory()
	}()
}

func (u *bosUploader) getObjFname(fpath string) string {
	_, fname := filepath.Split(fpath)
	return fmt.Sprintf("%v/%v", u.GetName(), fname)
}

func Connect2bos(remote, accessKey, accessSecret string) (c *bos.Client, err error) {
	utils.Logger.Debug("connect to bos for remote", zap.String("remote", remote))

	c, err = bos.NewClient(accessKey, accessSecret, remote)
	if err != nil {
		return nil, errors.Wrapf(err, "try to connect to bos %v error", remote)
	}
	c.MultipartSize = uploadChunkSize
	c.MaxParallel = uploadConcurrent

	return c, nil
}
