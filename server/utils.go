package server

import (
	"errors"
	"github.com/sjqzhang/googleAuthenticator"
	slog "github.com/sjqzhang/seelog"
	"runtime/debug"
	"strings"
)

//
func (server *Service) getRequestURI(action string) string {
	var uri string
	if supportGroupManage {
		uri = "/" + group + "/" + action
	} else {
		uri = "/" + action
	}
	return uri
}

//
func (server *Service) SaveStat() {
	SaveStatFunc := func() {
		defer func() {
			if re := recover(); re != nil {
				buffer := debug.Stack()
				slog.Error("SaveStatFunc")
				slog.Error(re)
				slog.Error(string(buffer))
			}
		}()
		stat := server.statMap.Get()
		if v, ok := stat[CONST_STAT_FILE_COUNT_KEY]; ok {
			switch v.(type) {
			case int64, int32, int, float64, float32:
				if v.(int64) >= 0 {
					if data, err := json.Marshal(stat); err != nil {
						slog.Error(err)
					} else {
						util.WriteBinFile(CONST_STAT_FILE_NAME, data)
					}
				}
			}
		}
	}
	SaveStatFunc()
}

//
func (server *Service) VerifyGoogleCode(secret string, code string, discrepancy int64) bool {
	var (
		goauth *googleAuthenticator.GAuth
	)
	goauth = googleAuthenticator.NewGAuth()
	if ok, err := goauth.VerifyCode(secret, code, discrepancy); ok {
		return ok
	} else {
		slog.Error(err)
		return ok
	}
}

//
func (server *Service) checkScene(scene string) (bool, error) {
	var (
		scenes []string
	)
	if len(scenes) == 0 {
		return true, nil
	}
	for _, s := range scenes {
		scenes = append(scenes, strings.Split(s, ":")[0])
	}
	if !util.Contains(scene, scenes) {
		return false, errors.New("not valid scene")
	}
	return true, nil
}
