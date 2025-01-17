package alias

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Laisky/go-ramjet/tasks/store"
	utils "github.com/Laisky/go-utils"
	"github.com/Laisky/zap"
)

var (
	httpClient = http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			MaxIdleConns:        3,
			MaxIdleConnsPerHost: 3,
		},
	}
)

func BindAliasesTask() {
	step := utils.Settings.GetDuration("tasks.elasticsearch-v2.aliases.interval")
	if utils.Settings.GetBool("debug") {
		step = 5
	}

	go store.TickerAfterRun(step*time.Second, runTask)
}

func runTask() {
	utils.Logger.Info("run elasticsearch.alias")
	var (
		err   error
		alias string
	)
	aliases := utils.Settings.Get("tasks.elasticsearch-v2.aliases.aliases").(map[string]interface{})
	api := utils.Settings.GetString("tasks.elasticsearch-v2.aliases.api")
	for index, aliasI := range aliases {
		alias = aliasI.(string)
		if err = createAlias(api, index, alias); err != nil {
			utils.Logger.Error("failed to refresh aliases",
				zap.String("api", maskAPI(api)),
				zap.String("index", index),
				zap.String("alias", alias),
				zap.Error(err))
		} else {
			utils.Logger.Info("success refresh alias",
				zap.String("index", index),
				zap.String("alias", alias))
		}
	}
}

func createAlias(api, index, alias string) error {
	data := map[string]interface{}{
		"actions": []interface{}{
			map[string]interface{}{
				"add": map[string]interface{}{
					"index": index,
					"alias": alias,
				},
			},
		},
	}
	reqJB, err := json.Marshal(data)
	utils.Logger.Debug("post", zap.ByteString("body", reqJB))
	if err != nil {
		utils.Logger.Error("try to marshal json got error", zap.Error(err))
	}

	if utils.Settings.GetBool("dry") {
		utils.Logger.Info("refresh aliases via post",
			zap.String("api", maskAPI(api)),
			zap.String("index", index),
			zap.String("alias", alias))
		return nil
	}

	resp, err := httpClient.Post(api, utils.HTTPJSONHeaderVal, bytes.NewReader(reqJB))
	if err != nil {
		utils.Logger.Error("try to request api got error",
			zap.String("api", maskAPI(api)),
			zap.String("index", index),
			zap.String("alias", alias),
			zap.Error(err))
		return err
	}
	defer resp.Body.Close()
	utils.Logger.Debug("got response code", zap.Int("code", resp.StatusCode))
	if err = utils.CheckResp(resp); err != nil {
		utils.Logger.Error("request api got error",
			zap.String("api", maskAPI(api)),
			zap.String("index", index),
			zap.String("alias", alias),
			zap.Error(err))
		return err
	}

	return nil
}

func maskAPI(api string) string {
	return strings.Join(strings.Split(api, "@")[1:], "")
}
