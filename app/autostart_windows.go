//go:build windows

package main

import (
        "log"
        "os"
        "strings"

        "golang.org/x/sys/windows/registry"
)

// HKCU 의 Run 키에 등록하면 로그인할 때마다 자동으로 실행된다.
//
// 다른 방법을 쓰지 않은 이유:
//   - Windows 서비스: Vista 이후 서비스는 세션 0 에 격리되어 있어
//     트레이 아이콘을 띄울 수도, 어무이 화면에서 음악 앱을 열 수도 없다.
//   - 작업 스케줄러: 관리자 권한·지연 실행 같은 장점이 이 앱엔 필요 없는데
//     schtasks 의존과 제거할 작업 항목만 늘어난다.
//
// HKCU 라서 관리자 권한이 필요 없고, 어무이 세션에서 실행되므로 트레이가 정상 동작한다.
const (
        runKeyPath   = `Software\Microsoft\Windows\CurrentVersion\Run`
        autostartKey = "EomuiMusic"
)

// autostartCommand는 Run 키에 넣을 명령줄이다.
// 경로에 공백이 있을 수 있어 따옴표로 감싼다.
func autostartCommand() (string, error) {
        exe, err := os.Executable()
        if err != nil {
                return "", err
        }
        return `"` + exe + `"`, nil
}

// autostartEnabled는 지금 등록되어 있는지, 그리고 그 경로가 현재 exe 인지 확인한다.
// 프로그램을 옮겼으면 등록이 남아 있어도 엉뚱한 곳을 가리키므로 꺼진 것으로 본다.
func autostartEnabled() bool {
        k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
        if err != nil {
                return false
        }
        defer k.Close()

        got, _, err := k.GetStringValue(autostartKey)
        if err != nil {
                return false
        }
        want, err := autostartCommand()
        if err != nil {
                return false
        }
        return strings.EqualFold(strings.TrimSpace(got), want)
}

func setAutostart(enable bool) error {
        k, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
        if err != nil {
                return err
        }
        defer k.Close()

        if !enable {
                err := k.DeleteValue(autostartKey)
                if err != nil && !os.IsNotExist(err) {
                        return err
                }
                log.Println("[INFO] 자동 실행 해제")
                return nil
        }

        cmd, err := autostartCommand()
        if err != nil {
                return err
        }
        if err := k.SetStringValue(autostartKey, cmd); err != nil {
                return err
        }
        log.Printf("[INFO] 자동 실행 등록: %s", cmd)
        return nil
}
