//go:build !windows

package main

import "log"

// Windows가 아니면 대화상자를 띄울 수 없다. 로그로만 남기고 진행한다.
func confirmDialog(title, text string) bool {
        log.Printf("[INFO] %s: %s (Windows가 아니라 확인 없이 진행)", title, text)
        return true
}

func infoDialog(title, text string) {
        log.Printf("[INFO] %s: %s", title, text)
}
