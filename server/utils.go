package server

import (
	"errors"
	"github.com/sjqzhang/googleAuthenticator"
	slog "github.com/sjqzhang/seelog"
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
