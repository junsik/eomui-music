package main

import (
        "bytes"
        "context"
        "encoding/json"
        "log"
        "os"
        "os/exec"
        "path/filepath"
        "strings"
        "time"
)

// yt-dlp 는 YouTube 가 바뀔 때마다 깨지고 그때마다 새 버전이 나온다.
// 설치할 때 넣어 준 것을 그대로 두면 몇 주 뒤 버튼이 안 먹는 날이 오는데,
// 어무이는 고치실 수 없다. 그래서 주기적으로 최신인지 확인한다.
//
// 내려받기를 직접 구현하지 않고 yt-dlp 의 자가 업데이트(-U)를 쓴다.
// 공식 경로라 배포 형식이 바뀌어도 따라간다.
const (
        updateCheckInterval = 7 * 24 * time.Hour
        updateTimeout       = 5 * time.Minute
        stateFileName       = "state.json"
)

type appState struct {
        LastUpdateCheck time.Time `json:"lastUpdateCheck"`
        YtDlpVersion    string    `json:"ytDlpVersion"`
}

func loadState() appState {
        var s appState
        data, err := os.ReadFile(filepath.Join(BaseDir, stateFileName))
        if err != nil {
                return s
        }
        if err := json.Unmarshal(data, &s); err != nil {
                log.Printf("[WARN] state.json 을 읽을 수 없습니다: %v", err)
        }
        return s
}

func saveState(s appState) {
        out, err := json.MarshalIndent(s, "", "  ")
        if err != nil {
                return
        }
        if err := os.WriteFile(filepath.Join(BaseDir, stateFileName), out, 0644); err != nil {
                log.Printf("[WARN] state.json 저장 실패: %v", err)
        }
}

// ytDlpVersion은 지금 설치된 버전 문자열을 돌려준다 (예: 2026.07.04).
func ytDlpVersion() string {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        var out bytes.Buffer
        cmd := exec.CommandContext(ctx, YtDlp, "--version")
        hideConsole(cmd)
        cmd.Stdout = &out
        if err := cmd.Run(); err != nil {
                return ""
        }
        return strings.TrimSpace(out.String())
}

// maybeUpdateYtDlp는 마지막 확인이 오래되었으면 yt-dlp 를 최신으로 올린다.
//
// ensureTools 안에서 toolsReady 를 켜기 전에 부른다.
// 그래야 다운로드 요청이 아직 막혀 있는 동안 실행 파일이 교체되고,
// 사용 중인 yt-dlp 를 덮어쓰는 일이 생기지 않는다.
func maybeUpdateYtDlp(force bool) {
        if !toolExists(YtDlp) {
                return
        }

        state := loadState()
        if !force && time.Since(state.LastUpdateCheck) < updateCheckInterval {
                log.Printf("[SETUP] yt-dlp 업데이트 확인 생략 (마지막 확인: %s)",
                        state.LastUpdateCheck.Format("2006-01-02"))
                return
        }

        before := ytDlpVersion()
        setBootstrapMsg("음악 받기 기능 최신 확인 중...")

        ctx, cancel := context.WithTimeout(context.Background(), updateTimeout)
        defer cancel()

        var out bytes.Buffer
        cmd := exec.CommandContext(ctx, YtDlp, "-U")
        hideConsole(cmd)
        cmd.Stdout = &out
        cmd.Stderr = &out

        if err := cmd.Run(); err != nil {
                // 실패해도 치명적이지 않다. 기존 버전으로 계속 동작한다.
                log.Printf("[WARN] yt-dlp 업데이트 실패(기존 버전으로 계속): %v — %s",
                        err, strings.TrimSpace(decodeOutput(out.Bytes())))
                // 매번 재시도해서 시작이 느려지지 않도록 확인 시각은 남긴다.
                state.LastUpdateCheck = time.Now()
                saveState(state)
                return
        }

        after := ytDlpVersion()
        if after != "" && after != before {
                log.Printf("[SETUP] yt-dlp 업데이트: %s → %s", before, after)
        } else {
                log.Printf("[SETUP] yt-dlp 최신 상태 (%s)", after)
        }

        state.LastUpdateCheck = time.Now()
        state.YtDlpVersion = after
        saveState(state)
}
