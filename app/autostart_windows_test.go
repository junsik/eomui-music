//go:build windows

package main

import (
        "os"
        "strings"
        "testing"

        "golang.org/x/sys/windows/registry"
)

// 이 파일의 테스트는 진짜 HKCU Run 키를 건드린다.
// 끝나면 복원하지만 중간에 죽으면 남의 레지스트리에 값이 남는다.
// installer/build.ps1 이 매번 go test 를 돌리므로 기본으로는 건너뛰고,
// 확인하고 싶을 때만 EOMUI_TEST_REGISTRY=1 로 켠다.
func skipUnlessRegistryTestsEnabled(t *testing.T) {
        t.Helper()

        if os.Getenv("EOMUI_TEST_REGISTRY") == "" {
                t.Skip("EOMUI_TEST_REGISTRY 미설정 — 실제 레지스트리를 건드리는 테스트는 건너뜀")
        }
}

// 실제 HKCU Run 키를 건드리므로 테스트 전후 상태를 반드시 복원한다.
func TestAutostartRegisterAndRemove(t *testing.T) {
        skipUnlessRegistryTestsEnabled(t)

        original, hadOriginal := readRunValue(t)
        t.Cleanup(func() { restoreRunValue(t, original, hadOriginal) })

        // 테스트를 깨끗한 상태에서 시작한다.
        if hadOriginal {
                restoreRunValue(t, "", false)
        }
        if autostartEnabled() {
                t.Fatal("등록 전인데 켜져 있다고 나옴")
        }

        if err := setAutostart(true); err != nil {
                t.Fatalf("등록 실패: %v", err)
        }
        if !autostartEnabled() {
                t.Fatal("등록했는데 꺼져 있다고 나옴")
        }

        // 값이 실제 exe 경로를 따옴표로 감싼 형태여야 한다.
        got, ok := readRunValue(t)
        if !ok {
                t.Fatal("레지스트리에 값이 없음")
        }
        exe, err := os.Executable()
        if err != nil {
                t.Fatal(err)
        }
        if got != `"`+exe+`"` {
                t.Fatalf("값 = %q, 기대값 %q", got, `"`+exe+`"`)
        }
        if !strings.HasPrefix(got, `"`) || !strings.HasSuffix(got, `"`) {
                t.Fatal("경로에 공백이 있으면 깨진다 — 따옴표가 없음")
        }

        // 두 번 등록해도 문제가 없어야 한다.
        if err := setAutostart(true); err != nil {
                t.Fatalf("중복 등록 실패: %v", err)
        }

        if err := setAutostart(false); err != nil {
                t.Fatalf("해제 실패: %v", err)
        }
        if autostartEnabled() {
                t.Fatal("해제했는데 켜져 있다고 나옴")
        }
        // 없는 것을 또 지워도 오류가 나면 안 된다.
        if err := setAutostart(false); err != nil {
                t.Fatalf("중복 해제에서 오류: %v", err)
        }
}

// 등록된 경로가 지금 exe 와 다르면(프로그램을 옮긴 경우) 꺼진 것으로 봐야 한다.
func TestAutostartDetectsStalePath(t *testing.T) {
        skipUnlessRegistryTestsEnabled(t)

        original, hadOriginal := readRunValue(t)
        t.Cleanup(func() { restoreRunValue(t, original, hadOriginal) })

        k, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
        if err != nil {
                t.Fatal(err)
        }
        err = k.SetStringValue(autostartKey, `"C:\옛날경로\eomui-music.exe"`)
        k.Close()
        if err != nil {
                t.Fatal(err)
        }

        if autostartEnabled() {
                t.Fatal("옛 경로가 등록되어 있는데 켜진 것으로 판단함")
        }
}

func readRunValue(t *testing.T) (string, bool) {
        t.Helper()

        k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
        if err != nil {
                return "", false
        }
        defer k.Close()

        v, _, err := k.GetStringValue(autostartKey)
        return v, err == nil
}

func restoreRunValue(t *testing.T, value string, had bool) {
        t.Helper()

        k, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
        if err != nil {
                t.Fatalf("레지스트리 복원 실패: %v", err)
        }
        defer k.Close()

        if had {
                if err := k.SetStringValue(autostartKey, value); err != nil {
                        t.Fatalf("레지스트리 복원 실패: %v", err)
                }
                return
        }
        if err := k.DeleteValue(autostartKey); err != nil && !os.IsNotExist(err) {
                t.Logf("정리 중 삭제 실패(무시): %v", err)
        }
}
