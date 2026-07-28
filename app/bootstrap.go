package main

import (
        "archive/zip"
        "context"
        "fmt"
        "io"
        "log"
        "net/http"
        "os"
        "path"
        "path/filepath"
        "runtime"
        "strings"
        "sync/atomic"
        "time"
)

// 필수 실행 파일 배포처 (Windows x64)
const (
        ytDlpURL  = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"
        ffmpegURL = "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"
        denoURL   = "https://github.com/denoland/deno/releases/latest/download/deno-x86_64-pc-windows-msvc.zip"
)

// toolsReady: yt-dlp / ffmpeg 준비 완료 여부. 준비 전에는 다운로드 요청을 거절한다.
var toolsReady atomic.Bool

// bootstrapMsg: 현재 준비 상태 문구 (트레이 툴팁 / /api/status 공용)
var bootstrapMsg atomic.Value // string

// 한 번 치명적 오류가 걸리면 이후 진행 문구로 덮이지 않는다.
// 포트를 못 잡았는데 "준비 완료"로 보이면 사용자가 원인을 찾을 수 없다.
var fatalFailure atomic.Bool

func setBootstrapMsg(format string, a ...interface{}) {
        if fatalFailure.Load() {
                return
        }
        msg := fmt.Sprintf(format, a...)
        bootstrapMsg.Store(msg)
        log.Printf("[SETUP] %s", msg)
        setTrayTooltip(msg)
}

// setFatalMsg는 더 이상 정상 동작할 수 없는 상태를 표시한다.
func setFatalMsg(format string, a ...interface{}) {
        msg := fmt.Sprintf(format, a...)
        bootstrapMsg.Store(msg)
        fatalFailure.Store(true)
        toolsReady.Store(false)
        log.Printf("[ERROR] %s", msg)
        setTrayTooltip(msg)
}

func getBootstrapMsg() string {
        if v, ok := bootstrapMsg.Load().(string); ok {
                return v
        }
        return ""
}

// renameWithRetry는 .part 파일을 최종 이름으로 옮긴다.
// Windows에서는 방금 쓴 exe를 백신이 실시간 검사로 잠깐 잡고 있어
// rename이 "다른 프로세스가 사용 중" 오류로 실패하는 일이 있다. 잠시 뒤 다시 시도한다.
func renameWithRetry(src, dest string) error {
        var err error
        for i := 0; i < 10; i++ {
                if err = os.Rename(src, dest); err == nil {
                        return nil
                }
                time.Sleep(time.Duration(i+1) * 300 * time.Millisecond)
        }
        return err
}

// 0바이트 잔재 파일은 없는 것으로 취급한다.
func toolExists(p string) bool {
        if p == "" {
                return false
        }
        info, err := os.Stat(p)
        return err == nil && !info.IsDir() && info.Size() > 0
}

// ensureTools는 exe와 같은 폴더에 yt-dlp.exe / ffmpeg.exe가 있는지 확인하고
// 없으면 내려받는다. 트레이 아이콘이 즉시 뜨도록 goroutine에서 호출한다.
func ensureTools() {
        defer func() {
                if r := recover(); r != nil {
                        log.Printf("[ERROR] 준비 작업 중 예외: %v", r)
                        setBootstrapMsg("필수 파일 준비 실패 — 로그를 확인해 주세요.")
                }
        }()

        if runtime.GOOS != "windows" {
                // 자동 다운로드 URL은 Windows 바이너리 전용이다. 그 외 OS는 확인만 한다.
                if toolExists(YtDlp) {
                        toolsReady.Store(true)
                        setBootstrapMsg("준비 완료")
                } else {
                        setBootstrapMsg("yt-dlp를 찾을 수 없습니다. 직접 설치해 주세요.")
                }
                return
        }

        if toolExists(YtDlp) {
                log.Printf("[SETUP] yt-dlp 확인됨: %s", YtDlp)
        } else if err := downloadTo(ytDlpURL, YtDlp, "yt-dlp"); err != nil {
                setBootstrapMsg("yt-dlp 내려받기 실패: %v", err)
                return
        }

        if toolExists(FFmpeg) {
                log.Printf("[SETUP] ffmpeg 확인됨: %s", FFmpeg)
        } else if err := installFfmpeg(FFmpeg); err != nil {
                setBootstrapMsg("ffmpeg 내려받기 실패: %v", err)
                return
        }

        ensureJsRuntime()

        // 다운로드가 아직 막혀 있는 지금 교체해야 안전하다.
        // toolsReady 이후면 사용 중인 yt-dlp.exe 를 덮어쓸 수 있다.
        maybeUpdateYtDlp(false)

        // 준비하는 동안 포트 바인딩 같은 치명적 오류가 났으면 완료로 덮지 않는다.
        if fatalFailure.Load() {
                return
        }
        toolsReady.Store(true)
        setBootstrapMsg("준비 완료 — 실행 중")
}

// ensureJsRuntime은 node/deno 중 아무것도 없을 때만 deno를 내려받는다.
// 실패해도 치명적이지 않다 — yt-dlp는 동작하되 일부 포맷이 빠질 수 있다.
func ensureJsRuntime() {
        if name, path := currentJsRuntime(); name != "" {
                log.Printf("[SETUP] JS 런타임 확인됨: %s (%s)", name, path)
                return
        }
        if runtime.GOOS != "windows" {
                log.Println("[WARN] JS 런타임 없음 — deno 자동 설치는 Windows만 지원합니다.")
                return
        }

        denoPath := filepath.Join(BaseDir, "deno.exe")
        if !toolExists(denoPath) {
                if err := installDeno(denoPath); err != nil {
                        log.Printf("[WARN] deno 내려받기 실패: %v", err)
                        return
                }
        }
        setJsRuntime("deno", denoPath)
        log.Printf("[SETUP] JS 런타임 설정: deno (%s)", denoPath)
}

// installDeno는 deno 배포 ZIP에서 deno.exe만 꺼내 놓는다.
func installDeno(dest string) error {
        zipPath := filepath.Join(BaseDir, "deno-download.zip")
        os.Remove(zipPath)
        defer os.Remove(zipPath)

        if err := downloadTo(denoURL, zipPath, "deno (약 43MB)"); err != nil {
                return err
        }
        setBootstrapMsg("deno 압축 푸는 중...")
        return extractFromZip(zipPath, "deno.exe", dest)
}

// downloadTo는 url을 dest로 내려받는다. .part 파일에 먼저 쓰고 완료 후 rename하므로
// 중간에 끊겨도 반쪽짜리 exe가 남지 않는다.
func downloadTo(url, dest, label string) error {
        tmp := dest + ".part"
        os.Remove(tmp)

        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
        defer cancel()

        req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
        if err != nil {
                return err
        }
        req.Header.Set("User-Agent", "eomui-music")

        setBootstrapMsg("%s 내려받는 중...", label)
        resp, err := http.DefaultClient.Do(req)
        if err != nil {
                return err
        }
        defer resp.Body.Close()
        if resp.StatusCode != http.StatusOK {
                return fmt.Errorf("HTTP %d", resp.StatusCode)
        }

        f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
        if err != nil {
                return err
        }
        pw := &progressWriter{total: resp.ContentLength, label: label}
        if _, err := io.Copy(io.MultiWriter(f, pw), resp.Body); err != nil {
                f.Close()
                os.Remove(tmp)
                return err
        }
        if err := f.Close(); err != nil {
                os.Remove(tmp)
                return err
        }
        if err := renameWithRetry(tmp, dest); err != nil {
                os.Remove(tmp)
                return err
        }
        log.Printf("[SETUP] %s 내려받기 완료: %s", label, dest)
        return nil
}

// installFfmpeg는 ffmpeg 배포 ZIP을 받아 bin/ffmpeg.exe 하나만 꺼내 놓고 ZIP은 지운다.
func installFfmpeg(dest string) error {
        zipPath := filepath.Join(BaseDir, "ffmpeg-download.zip")
        // 이전 실행이 압축 풀기 직전에 죽었으면 완성된 ZIP이 남아 있을 수 있다.
        os.Remove(zipPath)
        defer os.Remove(zipPath)

        if err := downloadTo(ffmpegURL, zipPath, "ffmpeg (약 110MB, 몇 분 걸립니다)"); err != nil {
                return err
        }

        setBootstrapMsg("ffmpeg 압축 푸는 중...")
        return extractFromZip(zipPath, "ffmpeg.exe", dest)
}

// extractFromZip은 ZIP 안에서 entryName과 이름이 같은 첫 엔트리를 dest로 꺼낸다.
// 배포본마다 상위 폴더 이름이 달라서(ffmpeg-<버전>-essentials_build/bin/…)
// 전체 경로가 아닌 파일 이름으로 찾는다.
func extractFromZip(zipPath, entryName, dest string) error {
        zr, err := zip.OpenReader(zipPath)
        if err != nil {
                return err
        }
        defer zr.Close()

        for _, f := range zr.File {
                // ZIP 엔트리 이름은 항상 '/' 구분자라 filepath가 아닌 path를 쓴다.
                if f.FileInfo().IsDir() || !strings.EqualFold(path.Base(f.Name), entryName) {
                        continue
                }
                rc, err := f.Open()
                if err != nil {
                        return err
                }
                defer rc.Close()

                tmp := dest + ".part"
                out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
                if err != nil {
                        return err
                }
                if _, err := io.Copy(out, rc); err != nil {
                        out.Close()
                        os.Remove(tmp)
                        return err
                }
                if err := out.Close(); err != nil {
                        os.Remove(tmp)
                        return err
                }
                if err := os.Rename(tmp, dest); err != nil {
                        os.Remove(tmp)
                        return err
                }
                log.Printf("[SETUP] %s 설치 완료: %s", entryName, dest)
                return nil
        }
        return fmt.Errorf("ZIP 안에서 %s를 찾지 못했습니다", entryName)
}

// progressWriter는 io.Copy를 지나가는 바이트 수를 세어 2초마다 진행률을 갱신한다.
type progressWriter struct {
        total   int64
        done    int64
        label   string
        lastLog time.Time
}

func (p *progressWriter) Write(b []byte) (int, error) {
        n := len(b)
        p.done += int64(n)
        if time.Since(p.lastLog) >= 2*time.Second {
                p.lastLog = time.Now()
                if p.total > 0 {
                        setBootstrapMsg("%s 내려받는 중... %.0f%% (%.1f/%.1f MB)",
                                p.label, float64(p.done)*100/float64(p.total), toMB(p.done), toMB(p.total))
                } else {
                        setBootstrapMsg("%s 내려받는 중... %.1f MB", p.label, toMB(p.done))
                }
        }
        return n, nil
}

func toMB(n int64) float64 {
        return float64(n) / (1024 * 1024)
}
