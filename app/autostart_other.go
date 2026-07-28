//go:build !windows

package main

import "errors"

// 자동 실행 등록은 Windows 전용이다.
func autostartEnabled() bool { return false }

func setAutostart(bool) error {
        return errors.New("자동 실행 등록은 Windows에서만 지원합니다")
}
