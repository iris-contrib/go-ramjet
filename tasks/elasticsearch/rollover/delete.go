package rollover

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Laisky/go-utils"
	"github.com/Laisky/zap"
	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"
)

// RunDeleteTask start to delete indices
func RunDeleteTask(ctx context.Context, sem *semaphore.Weighted, st *IdxSetting) {
	sem.Acquire(ctx, 1)
	defer sem.Release(1)
	utils.Logger.Debug("start to running delete expired index for alias", zap.String("alias", st.IdxAlias))

	var (
		allIdx        []string
		tobeDeleteIdx []string
		err           error
	)

	allIdx, err = LoadAllIndicesNames(st.API)
	if err != nil {
		utils.Logger.Error("load indices got error", zap.Error(err))
	}

	// Delete expired indices
	tobeDeleteIdx, err = FilterToBeDeleteIndicies(allIdx, st)
	if err != nil {
		utils.Logger.Error("try to filter delete indices got error", zap.Error(err))
	}

	// Do not delete write-alias
	tobeDeleteIdx, err = FilterReadyToBeDeleteIndices(GetAliasURL(st), tobeDeleteIdx)
	if err != nil {
		utils.Logger.Error("try to filter indices aliases got error", zap.Error(err))
	}

	utils.Logger.Info("try to delete indices", zap.Strings("index", tobeDeleteIdx))
	for _, idx := range tobeDeleteIdx {
		err = RemoveIndexByName(st.API, idx)
		if err != nil {
			utils.Logger.Error("try to delete index %v got error", zap.String("index", idx), zap.Error(err))
		}
	}
}

// RemoveIndexByName delete index by elasticsearch RESTful API
func RemoveIndexByName(api, index string) (err error) {
	utils.Logger.Info("remove es index", zap.String("index", index))
	url := api + index
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return errors.Wrap(err, "make request error")
	}

	utils.Logger.Debug("remove index", zap.String("index", index))
	if utils.Settings.GetBool("dry") {
		return nil
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "do request got error")
	}
	err = utils.CheckResp(resp)
	if err != nil {
		return errors.Wrap(err, "remove index got error")
	}

	utils.Logger.Info("success delete index", zap.String("index", index))
	return nil
}

// IsIdxShouldDelete check whether a index is should tobe deleted
// dateStr like `2016.10.31`, treated as +0800
func IsIdxShouldDelete(now time.Time, dateStr string, expires time.Duration) (bool, error) {
	layout := "2006.01.02 -0700"
	t, err := time.Parse(layout, strings.Replace(dateStr, "-", ".", -1)+" +0800")
	//t, err := time.Parse(layout, dateStr+" +0800")
	if err != nil {
		return false, errors.Wrapf(err, "parse date %v with layout %v error", dateStr, layout)
	}
	t = t.Add(24 * time.Hour) // elasticsearch dateStr has 1 day delay
	return now.Sub(t) > expires, nil
}

// FilterToBeDeleteIndicies return the indices that need be delete
func FilterToBeDeleteIndicies(allInd []string, idxSt *IdxSetting) (indices []string, err error) {
	utils.Logger.Debug("start to filter tobe delete indices", zap.Strings("indices", allInd), zap.String("regex", idxSt.Regexp.String()))
	var (
		idx      string
		subS     []string
		toDelete bool
	)
	for _, idx = range allInd {
		subS = idxSt.Regexp.FindStringSubmatch(idx)
		if len(subS) < 2 {
			continue
		}

		utils.Logger.Debug("check is index should be deleted with expires", zap.String("idx", idx), zap.Duration("expires", idxSt.Expires))
		toDelete, err = IsIdxShouldDelete(time.Now(), subS[1], idxSt.Expires)
		if err != nil {
			err = errors.Wrapf(err, "check whether index %v(%v) should delete got error", idx, idxSt.Expires)
			return nil, err
		}
		if toDelete {
			indices = append(indices, subS[0])
		}
	}

	utils.Logger.Debug("tobe delete indices", zap.Strings("indices", indices))
	return indices, nil
}
