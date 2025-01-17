package rollover

import (
	"bytes"
	"context"
	"html/template"
	"net/http"

	"github.com/Laisky/go-utils"
	"github.com/Laisky/zap"
	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"
)

var (
	idxRolloverReqBodyTpl *template.Template
)

func RunRolloverTask(ctx context.Context, sem *semaphore.Weighted, st *IdxSetting) {
	sem.Acquire(ctx, 1)
	defer sem.Release(1)

	if err := RolloverNewIndex(st.API, st); err != nil {
		utils.Logger.Error("rollover index got error", zap.String("index", st.IdxAlias), zap.Error(err))
	}
}

func init() {
	var err error
	idxRolloverReqBodyTpl, err = template.New("idxRolloverReqBodyTpl").Parse(`
	{
		"conditions": {
			"max_age": "{{.Rollover}}"
		},
		"aliases": {
			"{{.IdxAlias}}": {}
		},
		"settings": {
			"index": {
				"query.default_field": "message",
				"number_of_shards": {{.NShards}},
				"number_of_replicas": {{.NRepls}},
				"store.type": "niofs"
			}
		},
		{{.Mapping}}
	}`)
	if err != nil {
		err = errors.Wrap(err, "load index rollover template error")
		panic(err)
	}
}

func GetIdxRolloverReqBodyByIdxAlias(idxSt *IdxSetting) (jb *bytes.Buffer, err error) {
	utils.Logger.Debug("get rollover json body for index", zap.String("index", idxSt.IdxAlias))
	jb = new(bytes.Buffer)
	if err = idxRolloverReqBodyTpl.Execute(jb, idxSt); err != nil {
		return nil, errors.Wrapf(err, "parse index rollover for %v got error", idxSt.IdxAlias)
	}

	return jb, nil
}

func RolloverNewIndex(api string, st *IdxSetting) (err error) {
	utils.Logger.Info("rollover index", zap.String("index", st.IdxAlias))
	var (
		url  = api + st.IdxWriteAlias + "/_rollover"
		jb   *bytes.Buffer
		req  *http.Request
		resp *http.Response
	)
	jb, err = GetIdxRolloverReqBodyByIdxAlias(st)
	if err != nil {
		return errors.Wrap(err, "get index rollover body got error")
	}

	req, err = http.NewRequest("POST", url, jb)
	if err != nil {
		return errors.Wrap(err, "try to make rollover index http request got error")
	}
	req.Header.Set("Content-Type", "application/json")

	utils.Logger.Debug("request to rollover index", zap.String("index", jb.String()))
	if utils.Settings.GetBool("dry") {
		return nil
	}

	resp, err = httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "try to request rollover api got error")
	}

	err = utils.CheckResp(resp)
	if err != nil {
		return errors.Wrap(err, "rollover api return incorrect")
	}

	utils.Logger.Info("suceess rollover index", zap.String("index", st.IdxAlias))
	return nil
}
