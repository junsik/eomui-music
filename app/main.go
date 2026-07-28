package main

import (
        "bytes"
        "context"
        _ "embed"
        "encoding/json"
        "fmt"
        "io"
        "log"
        "net"
        "net/http"
        "os"
        "os/exec"
        "path/filepath"
        "regexp"
        "runtime"
        "strconv"
        "strings"
        "sync"
        "time"
        "unicode/utf8"

        "github.com/getlantern/systray"
        "github.com/google/uuid"
        "golang.org/x/text/encoding/korean"
        "golang.org/x/text/transform"
)

//go:embed icon.ico
var trayIconData []byte

// 큰 글씨·큰 버튼으로 된 시니어용 화면.
// 네이티브 GUI 툴킷은 cgo 가 붙어 "exe 하나" 구조가 깨지는데,
// 이미 로컬 서버가 있으므로 HTML 이 더 간단하고 화면 배율도 그대로 따라간다.
//
//go:embed ui.html
var uiHTML []byte

// ============================================================
// 어무이 음악 다운로더 - Standalone Go Server with System Tray
// Build: GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -H windowsgui" -o eomui-music.exe
// ============================================================

const PORT = "8080"

// Global paths
var (
        BaseDir       string
        TmpDir        string
        DesktopDir    string
        YtDlp         string
        FFmpeg        string
        JsRuntime     string // JS 런타임 실행 파일 경로
        JsRuntimeName string // yt-dlp가 아는 런타임 이름 (deno / node / bun / quickjs)
        LogFile       *os.File
)

// trayReady는 systray 초기화가 끝나면 닫힌다.
// 준비 작업 goroutine이 트레이보다 먼저 시작될 수 있어 그 전의 툴팁 갱신을 막아 준다.
var trayReady = make(chan struct{})

// JS 런타임은 준비 작업 goroutine이 나중에 바꿀 수 있어 잠금으로 감싼다.
var jsMu sync.RWMutex

func setJsRuntime(name, path string) {
        jsMu.Lock()
        defer jsMu.Unlock()
        JsRuntimeName, JsRuntime = name, path
}

func currentJsRuntime() (string, string) {
        jsMu.RLock()
        defer jsMu.RUnlock()
        return JsRuntimeName, JsRuntime
}

func setTrayTooltip(msg string) {
        select {
        case <-trayReady:
                systray.SetTooltip("어무이 음악 다운로더 — " + msg)
        default:
        }
}

// 콘솔이 확보되었는지. 없으면 stdout 쓰기가 실패하므로 로그 대상에서 뺀다.
var consoleAttached bool

// -console 로 실행하면 콘솔 창을 새로 띄운다.
// 터미널에서 실행한 경우엔 이 옵션 없이도 그 콘솔에 붙는다.
func wantsConsole() bool { return hasArg("console") }

// -open 이면 준비가 끝난 뒤 음악 목록 화면을 브라우저로 연다.
// 바탕화면·시작 메뉴 바로가기가 이 옵션을 쓴다.
// 자동 실행(Run 키)에는 붙이지 않는다 — 로그인할 때마다 브라우저가 뜨면 안 된다.
func wantsOpen() bool { return hasArg("open") }

func hasArg(name string) bool {
        for _, a := range os.Args[1:] {
                switch strings.ToLower(a) {
                case "-" + name, "--" + name, "/" + name:
                        return true
                }
        }
        return false
}

// setupLogging은 콘솔과 로그 파일 양쪽으로 출력한다.
// 터미널에서 실행했거나 -console 을 주면 콘솔에도 그대로 찍힌다.
func setupLogging() {
        var writers []io.Writer

        logPath := filepath.Join(BaseDir, "eomui-music.log")
        f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
        if err == nil {
                LogFile = f
                writers = append(writers, f)
        }

        // io.MultiWriter는 첫 쓰기 실패에서 멈춘다.
        // 콘솔이 없을 때 os.Stdout을 넣으면 파일 기록까지 같이 죽는다.
        if consoleAttached {
                writers = append(writers, os.Stdout)
        }

        switch len(writers) {
        case 0:
                log.SetOutput(io.Discard)
        case 1:
                log.SetOutput(writers[0])
        default:
                log.SetOutput(io.MultiWriter(writers...))
        }

        log.Println("=== 어무이 음악 다운로더 시작 ===")
        if err != nil {
                log.Printf("[WARN] 로그 파일을 열 수 없습니다: %v", err)
        } else {
                log.Printf("로그 파일: %s", logPath)
        }
}

// Detect base directory (where exe lives)
func getBaseDir() string {
        exe, err := os.Executable()
        if err == nil && exe != "" {
                dir := filepath.Dir(exe)
                if dir != "" && dir != "." && dir != "/" && dir != "\\" {
                        return dir
                }
        }
        cwd, _ := os.Getwd()
        return cwd
}

// Detect desktop directory
func getDesktopDir() string {
        if runtime.GOOS == "windows" {
                profile := os.Getenv("USERPROFILE")
                if profile == "" {
                        profile = os.Getenv("HOME")
                }
                if profile != "" {
                        desktop := filepath.Join(profile, "Desktop")
                        if _, err := os.Stat(desktop); err == nil {
                                return desktop
                        }
                }
        }
        home := os.Getenv("HOME")
        if home != "" {
                desktop := filepath.Join(home, "Desktop")
                if _, err := os.Stat(desktop); err == nil {
                        return desktop
                }
        }
        cwd, _ := os.Getwd()
        return cwd
}

// Detect JS runtime for yt-dlp.
// yt-dlp의 --js-runtimes는 RUNTIME[:PATH] 형식이므로 이름과 경로를 함께 돌려준다.
// (지원 이름: deno, node, bun, quickjs)
func getJsRuntime() (string, string) {
        // 1) deno.exe next to exe
        denoName := "deno"
        if runtime.GOOS == "windows" {
                denoName = "deno.exe"
        }
        denoPath := filepath.Join(BaseDir, denoName)
        if _, err := os.Stat(denoPath); err == nil {
                return "deno", denoPath
        }
        // 2) node in PATH
        if node, err := exec.LookPath("node"); err == nil {
                return "node", node
        }
        // 3) Common Windows Node.js install paths
        if runtime.GOOS == "windows" {
                candidates := []string{
                        filepath.Join(os.Getenv("ProgramFiles"), "nodejs", "node.exe"),
                        filepath.Join(os.Getenv("ProgramFiles(x86)"), "nodejs", "node.exe"),
                        filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "nodejs", "node.exe"),
                }
                for _, p := range candidates {
                        if p != "" {
                                if _, err := os.Stat(p); err == nil {
                                        return "node", p
                                }
                        }
                }
        }
        return "", ""
}

// Decode yt-dlp output. \uc720\ud6a8\ud55c UTF-8\uc774\uba74 \uadf8\ub300\ub85c \uc4f0\uace0,
// \uc544\ub2c8\uba74 CP949(Windows \ud655\uc7a5 EUC-KR)\ub85c \ud574\uc11d\ud55c\ub2e4.
// \uc608\uc804 \uad6c\ud604\uc740 iconv\ub97c shell\ub85c \ubd88\ub800\ub294\ub370 Windows\uc5d0\ub294 iconv\uac00 \uc5c6\uc5b4\uc11c \uc0ac\uc2e4\uc0c1 \ub3d9\uc791\ud558\uc9c0 \uc54a\uc558\ub2e4.
func decodeOutput(buf []byte) string {
        if utf8.Valid(buf) {
                return strings.TrimSpace(string(buf))
        }
        // korean.EUCKR\ub294 CP949(Windows-949)\ub97c \uad6c\ud604\ud55c\ub2e4.
        if decoded, _, err := transform.Bytes(korean.EUCKR.NewDecoder(), buf); err == nil {
                return strings.TrimSpace(string(decoded))
        }
        return strings.TrimSpace(string(buf))
}

var (
        illegalCharRe = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
        multiSpaceRe  = regexp.MustCompile(`\s+`)
        // Windows\uac00 \uc7a5\uce58 \uc774\ub984\uc73c\ub85c \uc608\uc57d\ud574 \ub454 \uac83\ub4e4. \ud655\uc7a5\uc790\ub97c \ubd99\uc5ec\ub3c4 \ud30c\uc77c\ub85c \ub9cc\ub4e4 \uc218 \uc5c6\ub2e4.
        reservedNameRe = regexp.MustCompile(`(?i)^(CON|PRN|AUX|NUL|COM[1-9]|LPT[1-9])$`)
)

// \ud30c\uc77c\uba85 \ucd5c\ub300 \uae38\uc774(\uae00\uc790 \uc218). \ubc14\uc774\ud2b8\uac00 \uc544\ub2c8\ub77c rune \uae30\uc900\uc774\ub2e4.
const maxFilenameRunes = 100

// Sanitize filename
func sanitizeFilename(name string) string {
        name = illegalCharRe.ReplaceAllString(name, "")
        name = multiSpaceRe.ReplaceAllString(name, " ")
        name = strings.TrimSpace(name)

        // \ubc14\uc774\ud2b8\ub85c \uc790\ub974\uba74 \ud55c\uae00(UTF-8 3\ubc14\uc774\ud2b8)\uc774 \uae00\uc790 \uc911\uac04\uc5d0\uc11c \uc798\ub824 \uae68\uc9c4\ub2e4.
        if r := []rune(name); len(r) > maxFilenameRunes {
                name = string(r[:maxFilenameRunes])
        }

        // Windows\ub294 \uc774\ub984 \ub05d\uc758 \ub9c8\uce68\ud45c\u00b7\uacf5\ubc31\uc744 \uc870\uc6a9\ud788 \ub5bc\uc5b4 \ub0b8\ub2e4. \ubbf8\ub9ac \uc815\ub9ac\ud55c\ub2e4.
        name = strings.TrimRight(name, ". ")

        if name == "" {
                name = "music"
        }
        if reservedNameRe.MatchString(name) {
                name = "_" + name
        }
        return name
}

// yt-dlp 한 번 실행에 허용하는 최대 시간.
// 긴 영상도 받아야 해서 넉넉하게 두되, 멈춘 프로세스가 핸들러를 영원히 잡는 것은 막는다.
const ytDlpTimeout = 30 * time.Minute

// 동시에 돌릴 수 있는 다운로드 수. 버튼 연타로 yt-dlp가 무한정 뜨는 것을 막는다.
var downloadSlots = make(chan struct{}, 2)

// 진행 중인 임시 파일 목록. 오래된 파일 청소가 살아 있는 다운로드를 건드리지 않도록 한다.
var activeTmpFiles sync.Map

// Run yt-dlp and return stdout
func runYtDlp(args []string) ([]byte, error) {
        ctx, cancel := context.WithTimeout(context.Background(), ytDlpTimeout)
        defer cancel()

        cmd := exec.CommandContext(ctx, YtDlp, args...)
        hideConsole(cmd) // 다운로드할 때마다 검은 창이 깜빡이지 않도록
        var stdout, stderr bytes.Buffer
        cmd.Stdout = &stdout
        cmd.Stderr = &stderr
        err := cmd.Run()
        if err != nil {
                if ctx.Err() == context.DeadlineExceeded {
                        return nil, fmt.Errorf("yt-dlp timeout: %v 안에 끝나지 않음", ytDlpTimeout)
                }
                // stderr는 rune 기준으로 자른다. 바이트로 자르면 한글이 글자 중간에서 깨진다.
                stderrStr := decodeOutput(stderr.Bytes())
                if r := []rune(stderrStr); len(r) > 200 {
                        stderrStr = string(r[len(r)-200:])
                }
                // exe 자체가 없거나 깨진 경우 stderr가 비어 있다.
                // 이때 err를 버리면 로그에 아무 단서도 남지 않는다.
                if stderrStr == "" {
                        return nil, fmt.Errorf("yt-dlp 실행 실패: %w", err)
                }
                return nil, fmt.Errorf("yt-dlp error: %s (%v)", stderrStr, err)
        }
        return stdout.Bytes(), nil
}

// Download audio and save to desktop
func downloadAudio(videoUrl string) (string, string, error) {
        // Ensure tmp dir
        if err := os.MkdirAll(TmpDir, 0755); err != nil {
                return "", "", fmt.Errorf("cannot create tmp dir: %v", err)
        }

        // Cleanup old files (>10 min)
        cleanupOldFiles()

        // Base args
        baseArgs := []string{"--no-playlist"}
        if jsName, jsPath := currentJsRuntime(); jsName != "" && jsPath != "" {
                // RUNTIME:PATH 형식. 경로만 넘기면 yt-dlp가 드라이브 문자(C:)를
                // 런타임 이름으로 오인해 무시해 버린다.
                baseArgs = append(baseArgs, "--js-runtimes", jsName+":"+jsPath)
        }
        // 내려받은 ffmpeg는 PATH에 없으므로 위치를 직접 알려 준다.
        if toolExists(FFmpeg) {
                baseArgs = append(baseArgs, "--ffmpeg-location", FFmpeg)
        }

        // 1) Get title
        title := "music"
        titleArgs := append([]string{}, baseArgs...)
        titleArgs = append(titleArgs, "--print", "title", "--no-download", videoUrl)
        titleBuf, err := runYtDlp(titleArgs)
        if err == nil {
                title = decodeOutput(titleBuf)
                log.Printf("[DEBUG] title = %s", title)
        } else {
                log.Printf("[WARN] Title fetch failed: %v", err)
        }

        safeName := sanitizeFilename(title)
        tmpId := uuid.New().String()
        tmpFile := filepath.Join(TmpDir, tmpId+".mp3")

        // 진행 중인 임시 파일은 cleanupOldFiles가 지우지 못하게 등록해 둔다.
        activeTmpFiles.Store(tmpFile, struct{}{})
        defer activeTmpFiles.Delete(tmpFile)

        // 2) Download and convert
        dlArgs := append([]string{}, baseArgs...)
        dlArgs = append(dlArgs,
                "-f", "bestaudio",
                "--extract-audio",
                "--audio-format", "mp3",
                "--audio-quality", "0",
                "--no-warnings",
                "--no-progress",
                "-o", tmpFile,
                videoUrl,
        )
        log.Printf("[DEBUG] downloading: %s", videoUrl)
        if _, err := runYtDlp(dlArgs); err != nil {
                os.Remove(tmpFile) // 실패하면 반쪽 파일을 남기지 않는다
                return "", "", fmt.Errorf("Download failed: %v", err)
        }

        // 3) Move to desktop. 방금 받은 곡은 항상 바탕화면에 보이게 둔다.
        savedPath := uniquePath(DesktopDir, safeName+".mp3")

        if err := os.Rename(tmpFile, savedPath); err != nil {
                // tmp와 바탕화면이 다른 드라이브면 rename이 실패한다. 이때는 복사한다.
                log.Printf("[WARN] rename failed: %v, doing manual copy", err)
                if err := copyFile(tmpFile, savedPath); err != nil {
                        return "", "", err
                }
                os.Remove(tmpFile)
        }

        finalFilename := filepath.Base(savedPath)
        log.Printf("[INFO] Saved to: %s", savedPath)

        library.add(LibraryEntry{
                VideoID:    extractVideoID(videoUrl),
                Title:      title,
                Filename:   finalFilename,
                Path:       savedPath,
                Downloaded: time.Now(),
        })
        // 방금 받은 곡은 바탕화면에 남고, 넘치는 오래된 곡이 보관함으로 내려간다.
        library.organize()

        return finalFilename, savedPath, nil
}

// copyFile은 파일을 통째로 메모리에 올리지 않고 옮긴다.
// 긴 영상의 MP3는 수백 MB가 될 수 있다.
func copyFile(src, dst string) error {
        in, err := os.Open(src)
        if err != nil {
                return err
        }
        defer in.Close()

        out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
        if err != nil {
                return err
        }
        if _, err := io.Copy(out, in); err != nil {
                out.Close()
                os.Remove(dst)
                return err
        }
        return out.Close()
}

// Cleanup old tmp files (>10 min)
func cleanupOldFiles() {
        entries, err := os.ReadDir(TmpDir)
        if err != nil {
                return
        }
        now := time.Now()
        for _, e := range entries {
                info, err := e.Info()
                if err != nil {
                        continue
                }
                p := filepath.Join(TmpDir, e.Name())
                // 10분이 넘었어도 아직 받고 있는 중이면 건드리지 않는다.
                if _, busy := activeTmpFiles.Load(p); busy {
                        continue
                }
                if now.Sub(info.ModTime()) > 10*time.Minute {
                        os.Remove(p)
                }
        }
}

// HTTP handlers

// 확장 프로그램이 붙이는 표시 헤더.
// <img>·<script>·폼 같은 단순 요청은 커스텀 헤더를 붙일 수 없어서
// 이 헤더의 존재만으로 다른 웹사이트가 몰래 부르는 요청을 걸러낼 수 있다.
const clientHeader = "X-Eomui-Client"

// 확장이 호출해 오는 정상적인 출처들.
var allowedOrigins = map[string]bool{
        "https://www.youtube.com":   true,
        "https://m.youtube.com":     true,
        "https://music.youtube.com": true,
        "https://youtube.com":       true,
}

func originAllowed(origin string) bool {
        if origin == "" {
                return false
        }
        // 확장 ID는 설치할 때마다 달라져서 스킴으로만 판단한다.
        return allowedOrigins[origin] || strings.HasPrefix(origin, "chrome-extension://")
}

// addCORS는 허용된 출처에만 CORS 헤더를 돌려준다.
// 예전에는 Access-Control-Allow-Origin을 * 로 열어 두어
// 사용자가 방문한 아무 웹사이트나 이 서버를 부를 수 있었다.
func addCORS(w http.ResponseWriter, r *http.Request) {
        if origin := r.Header.Get("Origin"); originAllowed(origin) {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Add("Vary", "Origin")
        }
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, "+clientHeader)
}

// 이 요청이 우리 확장에서 온 것인지 확인한다.
// 표시 헤더가 있거나 출처가 YouTube/확장이면 통과.
// 다른 사이트가 <img src="http://localhost:8080/api/download?url=...">로
// 몰래 다운로드를 걸어도 둘 다 만족시킬 수 없다.
func fromOurExtension(r *http.Request) bool {
        return r.Header.Get(clientHeader) != "" || originAllowed(r.Header.Get("Origin"))
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
        addCORS(w, r)
        // Preflight
        if r.Method == http.MethodOptions {
                w.WriteHeader(http.StatusNoContent)
                return
        }
        w.Header().Set("Content-Type", "application/json; charset=utf-8")

        // 우리 확장에서 온 요청만 받는다.
        if !fromOurExtension(r) {
                log.Printf("[WARN] 거부된 요청: origin=%q ua=%q", r.Header.Get("Origin"), r.Header.Get("User-Agent"))
                w.WriteHeader(http.StatusForbidden)
                json.NewEncoder(w).Encode(map[string]string{
                        "error": "이 프로그램은 어무이 음악 다운로더 확장에서만 사용할 수 있습니다.",
                })
                return
        }

        // yt-dlp / ffmpeg 준비가 끝나기 전이면 다운로드를 시작하지 않는다.
        if !toolsReady.Load() {
                msg := "필수 파일을 준비하는 중입니다. 잠시 후 다시 눌러 주세요."
                if s := getBootstrapMsg(); s != "" {
                        msg = s + " — 잠시 후 다시 눌러 주세요."
                }
                w.WriteHeader(http.StatusServiceUnavailable)
                json.NewEncoder(w).Encode(map[string]string{"error": msg})
                return
        }

        videoUrl := r.URL.Query().Get("url")
        if videoUrl == "" {
                w.WriteHeader(http.StatusBadRequest)
                json.NewEncoder(w).Encode(map[string]string{"error": "YouTube 주소를 입력해 주세요."})
                return
        }

        matched, _ := regexp.MatchString(`^(https?:\/\/)?(www\.)?(youtube\.com\/(watch\?v=|shorts\/|embed\/)|youtu\.be\/)`, videoUrl)
        if !matched {
                w.WriteHeader(http.StatusBadRequest)
                json.NewEncoder(w).Encode(map[string]string{"error": "올바른 YouTube 주소가 아닙니다."})
                return
        }

        // 동시 다운로드 수 제한. 자리가 없으면 기다리게 하지 않고 바로 안내한다.
        select {
        case downloadSlots <- struct{}{}:
                defer func() { <-downloadSlots }()
        default:
                w.WriteHeader(http.StatusTooManyRequests)
                json.NewEncoder(w).Encode(map[string]string{
                        "error": "지금 다른 음악을 받고 있어요. 끝난 뒤에 다시 눌러 주세요.",
                })
                return
        }

        // 이미 받은 영상이면 다시 받지 않는다.
        // 버튼을 두 번 누르는 일이 잦은데, 예전에는 "제목 (1).mp3"로 쌓였다.
        if existing, ok := library.find(extractVideoID(videoUrl)); ok {
                log.Printf("[INFO] 이미 받은 곡: %s", existing.Path)
                json.NewEncoder(w).Encode(map[string]interface{}{
                        "success":   true,
                        "duplicate": true,
                        "filename":  existing.Filename,
                        "path":      existing.Path,
                        "message":   "이미 받으신 노래예요.",
                })
                return
        }

        filename, path, err := downloadAudio(videoUrl)
        if err != nil {
                // 원본 오류는 로그에만 남긴다. yt-dlp의 stderr에는 로컬 경로가 섞여 있다.
                log.Printf("[ERROR] 다운로드 실패: %v", err)

                // 응답에는 정해 둔 문구만 내보낸다.
                raw := err.Error()
                msg := "다운로드에 실패했습니다. 다른 영상을 시도해 주세요."
                status := 500
                switch {
                case strings.Contains(raw, "Sign in"):
                        msg = "로그인이 필요한 영상입니다. 다른 영상을 시도해 주세요."
                        status = 400
                case strings.Contains(raw, "Video unavailable"), strings.Contains(raw, "Private"):
                        msg = "사용할 수 없는 영상입니다."
                        status = 400
                case strings.Contains(raw, "429"), strings.Contains(raw, "Too Many"):
                        msg = "너무 많은 요청입니다. 잠시 후 다시 시도해 주세요."
                        status = 429
                case strings.Contains(raw, "timeout"):
                        msg = "시간이 너무 오래 걸려서 중단했습니다. 더 짧은 영상을 시도해 주세요."
                        status = 504
                }
                w.WriteHeader(status)
                json.NewEncoder(w).Encode(map[string]string{"error": msg})
                return
        }

        json.NewEncoder(w).Encode(map[string]interface{}{
                "success":  true,
                "filename": filename,
                "path":     path,
                "message":  "바탕화면에 저장되었습니다.",
        })
}

// 시니어용 음악 목록 화면. 다운로드는 확장이 하고, 이 화면은
// 받은 곡을 큰 글씨로 보여 주고 눌러서 듣게 해 준다.
func handleRoot(w http.ResponseWriter, r *http.Request) {
        addCORS(w, r)
        // If user navigates to any unknown path, return 404 JSON (don't serve HTML)
        if r.URL.Path != "/" && r.URL.Path != "/index.html" {
                w.Header().Set("Content-Type", "application/json; charset=utf-8")
                w.WriteHeader(http.StatusNotFound)
                json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
                return
        }
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        w.Write(uiHTML)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
        addCORS(w, r)
        w.Header().Set("Content-Type", "application/json")
        jsName, jsPath := currentJsRuntime()
        json.NewEncoder(w).Encode(map[string]interface{}{
                "ok":        true,
                "ready":     toolsReady.Load(),
                "setup":     getBootstrapMsg(),
                "desktop":   DesktopDir,
                "ytDlp":     YtDlp,
                "ffmpeg":    FFmpeg,
                "jsRuntime": jsName,
                "jsPath":    jsPath,
        })
}

// handleSongs는 받은 곡 목록을 최신순으로 나눠서 돌려준다.
// 수백 곡이어도 첫 화면이 짧도록 offset/limit 으로 잘라 보낸다.
func handleSongs(w http.ResponseWriter, r *http.Request) {
        if !guardAPI(w, r) {
                return
        }
        offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
        limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
        if err != nil || limit <= 0 || limit > 200 {
                limit = 20
        }

        songs, total := library.page(offset, limit)
        json.NewEncoder(w).Encode(map[string]interface{}{
                "songs":  songs,
                "total":  total,
                "offset": offset,
        })
}

// handlePlay는 곡을 기본 음악 앱으로 연다.
// 브라우저는 로컬 파일을 직접 실행할 수 없으므로 서버가 대신 연다.
func handlePlay(w http.ResponseWriter, r *http.Request) {
        if !guardAPI(w, r) {
                return
        }
        path := r.URL.Query().Get("path")

        // 아무 경로나 열어 주면 안 된다. 우리가 관리하는 곡인지 확인한다.
        if !library.hasPath(path) {
                log.Printf("[WARN] 목록에 없는 파일 재생 요청: %q", path)
                w.WriteHeader(http.StatusForbidden)
                json.NewEncoder(w).Encode(map[string]string{"error": "목록에 없는 파일입니다."})
                return
        }
        if _, err := os.Stat(path); err != nil {
                w.WriteHeader(http.StatusNotFound)
                json.NewEncoder(w).Encode(map[string]string{"error": "파일을 찾을 수 없습니다."})
                return
        }

        log.Printf("[INFO] 재생: %s", path)
        openPath(path)
        json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func handleOpenFolder(w http.ResponseWriter, r *http.Request) {
        if !guardAPI(w, r) {
                return
        }
        openPath(musicFolder())
        json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleTidy는 바탕화면에 흩어진 MP3 를 정리한다.
// 확인은 화면에서 이미 받았으므로 여기서는 대화상자를 띄우지 않는다.
func handleTidy(w http.ResponseWriter, r *http.Request) {
        if !guardAPI(w, r) {
                return
        }
        added, failed := library.adoptLooseDesktopFiles()
        library.organize()
        onDesktop, archived := library.counts()

        log.Printf("[INFO] 화면에서 정리 실행: %d곡 등록, 바탕화면 %d곡 / 보관함 %d곡",
                added, onDesktop, archived)

        json.NewEncoder(w).Encode(map[string]interface{}{
                "success":   true,
                "added":     added,
                "failed":    failed,
                "onDesktop": onDesktop,
                "archived":  archived,
        })
}

// guardAPI는 모든 API 핸들러가 공통으로 하는 앞단 처리를 맡는다.
// 계속 진행해도 되면 true 를 돌려준다.
func guardAPI(w http.ResponseWriter, r *http.Request) bool {
        addCORS(w, r)
        if r.Method == http.MethodOptions {
                w.WriteHeader(http.StatusNoContent)
                return false
        }
        w.Header().Set("Content-Type", "application/json; charset=utf-8")

        if !fromOurExtension(r) {
                w.WriteHeader(http.StatusForbidden)
                json.NewEncoder(w).Encode(map[string]string{
                        "error": "이 프로그램의 화면에서만 사용할 수 있습니다.",
                })
                return false
        }
        return true
}

const youtubeURL = "https://www.youtube.com"

// openBrowserTo는 기본 브라우저로 주소를 연다.
func openBrowserTo(url string) {
        startHidden(shellOpenCommand(url))
}

// openBrowser는 음악 목록 화면을 연다.
func openBrowser() {
        openBrowserTo(fmt.Sprintf("http://localhost:%s", PORT))
}

// shellOpenCommand는 OS 의 기본 프로그램으로 여는 명령을 만든다.
func shellOpenCommand(target string) *exec.Cmd {
        switch runtime.GOOS {
        case "windows":
                return exec.Command("cmd", "/c", "start", "", target)
        case "darwin":
                return exec.Command("open", target)
        default:
                return exec.Command("xdg-open", target)
        }
}

// startHidden은 명령을 콘솔 창 없이 띄운다.
// cmd /c start 는 그냥 실행하면 검은 창이 깜빡인다.
func startHidden(cmd *exec.Cmd) {
        hideConsole(cmd)
        if err := cmd.Start(); err != nil {
                log.Printf("[WARN] 실행 실패(%v): %v", cmd.Args, err)
                return
        }
        // 좀비 프로세스가 남지 않도록 종료를 거둬 준다.
        go cmd.Wait()
}

// openPath는 폴더면 탐색기로 열고, 파일이면 기본 연결 프로그램으로 연다.
func openPath(path string) {
        if runtime.GOOS == "windows" {
                // explorer.exe 는 파일이면 연결된 기본 프로그램을, 폴더면 탐색기를 연다.
                startHidden(exec.Command("explorer.exe", path))
                return
        }
        startHidden(shellOpenCommand(path))
}

// toggleAutostart는 자동 실행 등록을 켜고 끄며 메뉴 체크 표시를 맞춘다.
func toggleAutostart(item *systray.MenuItem) {
        enable := !item.Checked()
        if err := setAutostart(enable); err != nil {
                log.Printf("[ERROR] 자동 실행 설정 실패: %v", err)
                infoDialog("자동 실행", "설정을 바꾸지 못했습니다.\n로그 파일을 확인해 주세요.")
                return
        }
        if enable {
                item.Check()
        } else {
                item.Uncheck()
        }
}

// tidyDesktopWithConfirm은 바탕화면에 흩어진 MP3를 보관함으로 옮긴다.
// 파일을 여러 개 옮기는 되돌리기 어려운 작업이라 먼저 물어본다.
func tidyDesktopWithConfirm() {
        n := library.countLooseDesktopMP3s()
        if n == 0 {
                infoDialog("바탕화면 음악 정리", "정리할 MP3 파일이 없습니다.")
                return
        }

        keep := library.settings.DesktopKeepCount
        if !confirmDialog("바탕화면 음악 정리", fmt.Sprintf(
                "바탕화면에 있는 MP3 %d개를 정리합니다.\n\n"+
                        "· 최근 %d곡은 바탕화면에 그대로 둡니다\n"+
                        "· 나머지는 바탕화면\\%s\\(연도)년 (월)월\\ 폴더로 옮깁니다\n"+
                        "· MP3가 아닌 파일은 건드리지 않습니다\n\n"+
                        "계속할까요?", n, keep, archiveFolderName)) {
                log.Println("[INFO] 사용자가 바탕화면 정리를 취소했습니다")
                return
        }

        _, failed := library.adoptLooseDesktopFiles()
        library.organize()
        onDesktop, archived := library.counts()

        msg := fmt.Sprintf("정리했습니다.\n\n바탕화면에 %d곡, 보관함에 %d곡.", onDesktop, archived)
        if failed > 0 {
                msg += fmt.Sprintf("\n\n%d곡은 처리하지 못했습니다 (재생 중이거나 잠겨 있을 수 있습니다).", failed)
        }
        infoDialog("바탕화면 음악 정리", msg)
}

// ===== System Tray =====

func onTrayReady() {
        systray.SetIcon(trayIconData)
        systray.SetTitle("")
        systray.SetTooltip("어무이 음악 다운로더 (실행 중)")

        // 이제부터 준비 상태를 툴팁에 반영할 수 있다. 이미 진행된 상태가 있으면 즉시 표시.
        close(trayReady)
        if msg := getBootstrapMsg(); msg != "" {
                setTrayTooltip(msg)
        }

        mYouTube := systray.AddMenuItem("유튜브 열기", "음악을 받으러 YouTube 로 이동")
        mList := systray.AddMenuItem("음악 목록 보기", "받은 곡을 보고 들을 수 있는 화면")
        systray.AddSeparator()
        mMusic := systray.AddMenuItem("음악 폴더 열기", "받은 음악이 있는 폴더")
        mTidy := systray.AddMenuItem("바탕화면 음악 정리", "바탕화면에 흩어진 MP3를 보관함으로 옮깁니다")
        systray.AddSeparator()
        mLog := systray.AddMenuItem("로그 폴더 열기", "eomui-music.log")
        systray.AddSeparator()
        mAuto := systray.AddMenuItemCheckbox("윈도우 시작할 때 자동 실행",
                "컴퓨터를 켜면 이 프로그램이 자동으로 실행됩니다", autostartEnabled())
        systray.AddSeparator()
        mQuit := systray.AddMenuItem("종료", "프로그램 종료")

        go func() {
                for {
                        select {
                        case <-mYouTube.ClickedCh:
                                openBrowserTo(youtubeURL)
                        case <-mList.ClickedCh:
                                openBrowser()
                        case <-mMusic.ClickedCh:
                                openPath(musicFolder())
                        case <-mTidy.ClickedCh:
                                go tidyDesktopWithConfirm()
                        case <-mAuto.ClickedCh:
                                toggleAutostart(mAuto)
                        case <-mLog.ClickedCh:
                                openPath(BaseDir)
                        case <-mQuit.ClickedCh:
                                log.Println("[INFO] 사용자가 트레이에서 종료 클릭")
                                systray.Quit()
                                return
                        }
                }
        }()
}

func onTrayExit() {
        log.Println("[INFO] 트레이 종료 — 프로세스 종료")
        if LogFile != nil {
                LogFile.Close()
        }
        os.Exit(0)
}

func main() {
        // 로그를 찍기 전에 콘솔부터 확보한다.
        // 터미널에서 실행했으면 그 콘솔에 붙고, -console 이면 새로 띄운다.
        consoleAttached = attachConsole(wantsConsole())

        // 두 벌이 뜨면 두 번째는 포트를 못 잡아 트레이만 떠 있는 상태가 된다.
        // 이미 돌고 있으면 음악 목록 화면만 열어 주고 조용히 빠진다 —
        // 어머니가 바로가기를 다시 누른 경우 "이미 실행 중" 창보다 이쪽이 자연스럽다.
        if !claimSingleInstance() {
                // 로그 파일은 아직 열기 전이다. 콘솔로 실행했다면 여기 찍힌다.
                log.Println("[INFO] 이미 실행 중 — 음악 목록 화면만 열고 종료합니다")
                openBrowser()
                return
        }

        BaseDir = getBaseDir()
        TmpDir = filepath.Join(BaseDir, "tmp")
        DesktopDir = getDesktopDir()

        setupLogging()

        if runtime.GOOS == "windows" {
                YtDlp = filepath.Join(BaseDir, "yt-dlp.exe")
                FFmpeg = filepath.Join(BaseDir, "ffmpeg.exe")
        } else {
                YtDlp = filepath.Join(BaseDir, "yt-dlp")
                if _, err := os.Stat(YtDlp); err != nil {
                        if p, err := exec.LookPath("yt-dlp"); err == nil {
                                YtDlp = p
                        }
                }
                if p, err := exec.LookPath("ffmpeg"); err == nil {
                        FFmpeg = p
                }
        }

        setJsRuntime(getJsRuntime())

        log.Printf("[INFO] BASE_DIR = %s", BaseDir)
        log.Printf("[INFO] Desktop = %s", DesktopDir)
        if jsName, jsPath := currentJsRuntime(); jsName != "" {
                log.Printf("[INFO] JS runtime: %s (%s)", jsName, jsPath)
        } else {
                log.Println("[WARN] No JS runtime found. YouTube downloads may fail.")
        }

        http.HandleFunc("/api/download", handleDownload)
        http.HandleFunc("/api/status", handleStatus)
        http.HandleFunc("/api/songs", handleSongs)
        http.HandleFunc("/api/play", handlePlay)
        http.HandleFunc("/api/open-folder", handleOpenFolder)
        http.HandleFunc("/api/tidy", handleTidy)
        http.HandleFunc("/", handleRoot)

        // 받은 음악 목록을 불러오고, 바탕화면에 넘치는 곡을 보관함으로 내린다.
        library.load()
        go library.organize()

        // yt-dlp / ffmpeg가 없으면 내려받는다. 트레이 아이콘이 바로 뜨도록 백그라운드로 돌린다.
        setBootstrapMsg("필수 파일 확인 중...")
        go ensureTools()

        // Start HTTP server in background goroutine
        go func() {
                // 127.0.0.1에만 묶는다. ":8080"이면 같은 공유기에 붙은 다른 기기도
                // 이 서버를 불러 바탕화면에 파일을 쓸 수 있다.
                ln, err := net.Listen("tcp", "127.0.0.1:"+PORT)
                if err != nil {
                        // 포트를 못 잡으면 서버가 없는 채로 트레이만 떠 있게 된다.
                        // 준비 상태 문구를 덮어써서 정상처럼 보이지 않게 한다.
                        log.Printf("[ERROR] 포트 바인딩 실패: %v", err)
                        setFatalMsg("포트 %s를 쓸 수 없습니다 — 이미 실행 중인지 확인하세요", PORT)
                        return
                }
                log.Printf("🎵 어무이 음악 다운로드 서버 시작: http://localhost:%s", PORT)
                log.Printf("📋 트레이 아이콘 우클릭으로 종료하세요.")

                // 바로가기로 실행한 경우에만 화면을 띄운다.
                // 서버가 뜬 뒤에 열어야 빈 페이지가 보이지 않는다.
                if wantsOpen() {
                        openBrowser()
                }

                if err := http.Serve(ln, nil); err != nil {
                        log.Printf("ERROR: server failed: %v", err)
                }
        }()

        // Run systray on main goroutine — blocks until user clicks "종료"
        systray.Run(onTrayReady, onTrayExit)
}
