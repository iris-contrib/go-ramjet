package backup_test

import (
	"fmt"
	"testing"

	"github.com/baidubce/bce-sdk-go/services/bos"

	"github.com/Laisky/go-ramjet/tasks/logrotate/backup"
	"github.com/Laisky/go-utils"
)

var (
	bosargs                                 *backup.BosArgs
	remote, accessKey, accessSecret, bucket string
	client                                  *bos.Client
)

func init() {
	setUp()
}

func setUp() {
	utils.SetupSettings()
	st := backup.LoadSettings()[0]
	fmt.Printf("%+v\n", st)

	remote = st.Args["remote"].(string)
	accessKey = st.Args["access_key"].(string)
	accessSecret = st.Args["access_secret"].(string)
	bucket = st.Args["bucket"].(string)
}
func Test0Connect2BOS(t *testing.T) {
	var err error
	client, err = backup.Connect2bos(remote, accessKey, accessSecret)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	t.Logf("got client: %+v", client)
}

func TestUploadFile(t *testing.T) {
	got, err := client.PutObjectFromFile(bucket, "test", "/Users/laisky/repo/pateo/fluentd-conf/README.md", nil)
	if err != nil {
		t.Errorf("%+v", err)
	}
	t.Logf("%+v", got)
}
