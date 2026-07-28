package main

import (
        "encoding/json"
        "net/http"
        "net/http/httptest"
        "path/filepath"
        "strings"
        "testing"
        "unicode/utf8"

        "golang.org/x/text/encoding/korean"
        "golang.org/x/text/transform"
)

func TestSanitizeFilenameKoreanTruncation(t *testing.T) {
        // 한글은 UTF-8에서 3바이트다. 바이트로 자르면 글자 중간에서 깨진다.
        long := strings.Repeat("가", 150)
        got := sanitizeFilename(long)

        if !utf8.ValidString(got) {
                t.Fatalf("깨진 UTF-8: %q", got)
        }
        if n := utf8.RuneCountInString(got); n != maxFilenameRunes {
                t.Fatalf("글자 수 = %d, 기대값 %d", n, maxFilenameRunes)
        }
        if strings.ContainsRune(got, '�') {
                t.Fatal("치환 문자(U+FFFD)가 들어 있음")
        }
}

func TestSanitizeFilenameRules(t *testing.T) {
        cases := []struct{ in, want string }{
                {`a<b>c:d"e/f\g|h?i*j`, "abcdefghij"},
                {"  여러   공백  ", "여러 공백"},
                {"", "music"},
                {"...", "music"},         // 불법 문자 제거 후 남은 마침표까지 정리
                {"이름.", "이름"},           // Windows는 끝의 마침표를 떼어 낸다
                {"CON", "_CON"},          // 예약된 장치 이름
                {"nul", "_nul"},          // 대소문자 구분 없음
                {"COM1", "_COM1"},
                {"정상 제목", "정상 제목"},
        }
        for _, c := range cases {
                if got := sanitizeFilename(c.in); got != c.want {
                        t.Errorf("sanitizeFilename(%q) = %q, 기대값 %q", c.in, got, c.want)
                }
        }
}

func TestDecodeOutputCP949(t *testing.T) {
        const want = "한국어 제목"

        // UTF-8 입력은 그대로 통과해야 한다.
        if got := decodeOutput([]byte(want + "\n")); got != want {
                t.Errorf("UTF-8: %q, 기대값 %q", got, want)
        }

        // CP949로 인코딩한 입력은 복원되어야 한다.
        cp949, _, err := transform.Bytes(korean.EUCKR.NewEncoder(), []byte(want))
        if err != nil {
                t.Fatalf("테스트 입력 인코딩 실패: %v", err)
        }
        if utf8.Valid(cp949) {
                t.Skip("이 문자열은 CP949에서도 유효한 UTF-8이라 분기를 못 탐")
        }
        if got := decodeOutput(cp949); got != want {
                t.Errorf("CP949: %q, 기대값 %q", got, want)
        }
}

func TestOriginAllowed(t *testing.T) {
        allowed := []string{
                "https://www.youtube.com",
                "https://m.youtube.com",
                "https://music.youtube.com",
                "chrome-extension://abcdefghijklmnopabcdefghijklmnop",
        }
        for _, o := range allowed {
                if !originAllowed(o) {
                        t.Errorf("%q 는 허용되어야 함", o)
                }
        }
        denied := []string{
                "",
                "https://evil.com",
                "http://localhost:8080",
                "https://youtube.com.evil.com",
                "https://notyoutube.com",
        }
        for _, o := range denied {
                if originAllowed(o) {
                        t.Errorf("%q 는 거부되어야 함", o)
                }
        }
}

// 다른 웹사이트가 <img src="http://localhost:8080/api/download?url=...">로
// 몰래 다운로드를 거는 것을 막아야 한다.
func TestDownloadRejectsForgedRequests(t *testing.T) {
        toolsReady.Store(true)
        defer toolsReady.Store(false)

        const target = "/api/download?url=https://www.youtube.com/watch?v=aqz-KE-bpKQ"

        forged := []struct {
                name    string
                headers map[string]string
        }{
                // <img>/<script>/최상위 이동은 Origin 헤더가 아예 없다.
                {"헤더 없음", nil},
                {"악성 사이트 origin", map[string]string{"Origin": "https://evil.com"}},
                {"유사 도메인", map[string]string{"Origin": "https://youtube.com.evil.com"}},
        }
        for _, f := range forged {
                t.Run(f.name, func(t *testing.T) {
                        req := httptest.NewRequest(http.MethodGet, target, nil)
                        for k, v := range f.headers {
                                req.Header.Set(k, v)
                        }
                        rec := httptest.NewRecorder()
                        handleDownload(rec, req)

                        if rec.Code != http.StatusForbidden {
                                t.Fatalf("status = %d, 기대값 403", rec.Code)
                        }
                        // 허용되지 않은 출처에는 CORS 헤더도 주지 않는다.
                        if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
                                t.Fatalf("Access-Control-Allow-Origin = %q, 비어 있어야 함", got)
                        }
                })
        }
}

// 확장에서 온 요청은 403에서 막히지 않고 다음 단계로 넘어가야 한다.
func TestDownloadAcceptsExtensionRequests(t *testing.T) {
        toolsReady.Store(false) // 준비 전이면 503 — 403만 아니면 게이트를 통과한 것
        setBootstrapMsg("준비 중")

        accepted := []map[string]string{
                {clientHeader: "1"},
                {"Origin": "https://www.youtube.com"},
                {"Origin": "chrome-extension://abcdefghijklmnopabcdefghijklmnop"},
        }
        for _, h := range accepted {
                req := httptest.NewRequest(http.MethodGet,
                        "/api/download?url=https://www.youtube.com/watch?v=aqz-KE-bpKQ", nil)
                for k, v := range h {
                        req.Header.Set(k, v)
                }
                rec := httptest.NewRecorder()
                handleDownload(rec, req)

                if rec.Code == http.StatusForbidden {
                        t.Errorf("%v: 403으로 막힘 (통과해야 함)", h)
                }
        }
}

// 커스텀 헤더를 쓰려면 브라우저가 preflight를 보낼 수 있다. 이를 받아 줘야 한다.
func TestPreflightAllowsClientHeader(t *testing.T) {
        req := httptest.NewRequest(http.MethodOptions, "/api/download", nil)
        req.Header.Set("Origin", "https://www.youtube.com")
        req.Header.Set("Access-Control-Request-Headers", clientHeader)
        rec := httptest.NewRecorder()
        handleDownload(rec, req)

        if rec.Code != http.StatusNoContent {
                t.Fatalf("preflight status = %d, 기대값 204", rec.Code)
        }
        if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://www.youtube.com" {
                t.Fatalf("Allow-Origin = %q", got)
        }
        if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, clientHeader) {
                t.Fatalf("Allow-Headers = %q, %q 를 포함해야 함", got, clientHeader)
        }
}

// 준비 완료 전에는 다운로드를 시작하지 않고 진행 상황을 알려 준다.
func TestDownloadGatedUntilToolsReady(t *testing.T) {
        toolsReady.Store(false)
        setBootstrapMsg("ffmpeg 내려받는 중... 42%%")

        req := httptest.NewRequest(http.MethodGet,
                "/api/download?url=https://www.youtube.com/watch?v=aqz-KE-bpKQ", nil)
        req.Header.Set(clientHeader, "1")
        rec := httptest.NewRecorder()
        handleDownload(rec, req)

        if rec.Code != http.StatusServiceUnavailable {
                t.Fatalf("status = %d, 기대값 503", rec.Code)
        }
        var body map[string]string
        if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
                t.Fatalf("JSON 아님: %v", err)
        }
        if !strings.Contains(body["error"], "42%") {
                t.Fatalf("진행 상황이 안내에 없음: %q", body["error"])
        }
}

// 포트 바인딩 실패 같은 치명적 오류는 이후 진행 문구로 덮이면 안 된다.
// 서버가 없는데 "준비 완료 — 실행 중"으로 보이면 원인을 찾을 수 없다.
func TestFatalMsgIsNotOverwritten(t *testing.T) {
        defer func() {
                fatalFailure.Store(false)
                toolsReady.Store(false)
        }()

        toolsReady.Store(true)
        setFatalMsg("포트 8080를 쓸 수 없습니다 — 이미 실행 중인지 확인하세요")

        if toolsReady.Load() {
                t.Fatal("치명적 오류 후에도 준비 완료 상태로 남아 있음")
        }

        // 준비 goroutine이 뒤늦게 끝나며 덮어쓰려는 상황
        setBootstrapMsg("준비 완료 — 실행 중")

        if got := getBootstrapMsg(); !strings.Contains(got, "포트") {
                t.Fatalf("치명적 오류 문구가 덮였음: %q", got)
        }
}

// yt-dlp.exe 가 없으면 stderr가 비어 있다. 이때도 로그에 남길 단서는 있어야 한다.
// (응답은 정해 둔 문구만 나가므로 오류 자체가 유일한 진단 수단이다.)
func TestRunYtDlpReportsCauseWhenStderrEmpty(t *testing.T) {
        old := YtDlp
        YtDlp = filepath.Join(t.TempDir(), "없는-yt-dlp.exe")
        defer func() { YtDlp = old }()

        _, err := runYtDlp([]string{"--version"})
        if err == nil {
                t.Fatal("오류가 나야 함")
        }
        msg := err.Error()
        if strings.TrimSpace(strings.TrimPrefix(msg, "yt-dlp error:")) == "" {
                t.Fatalf("단서 없는 오류: %q", msg)
        }
        if !strings.Contains(msg, "없는-yt-dlp") {
                t.Fatalf("어떤 파일이 문제인지 알 수 없음: %q", msg)
        }
        t.Logf("로그에 남는 오류: %s", msg)
}

// 실패 응답에 yt-dlp의 stderr(로컬 경로 포함)가 그대로 새어 나가면 안 된다.
func TestDownloadErrorDoesNotLeakPaths(t *testing.T) {
        toolsReady.Store(true)
        defer toolsReady.Store(false)

        // YtDlp 가 없는 경로면 downloadAudio 가 실패한다.
        oldYtDlp, oldTmp, oldDesktop := YtDlp, TmpDir, DesktopDir
        YtDlp = filepath.Join(t.TempDir(), "없는-yt-dlp.exe")
        TmpDir = t.TempDir()
        DesktopDir = t.TempDir()
        defer func() { YtDlp, TmpDir, DesktopDir = oldYtDlp, oldTmp, oldDesktop }()

        req := httptest.NewRequest(http.MethodGet,
                "/api/download?url=https://www.youtube.com/watch?v=aqz-KE-bpKQ", nil)
        req.Header.Set(clientHeader, "1")
        rec := httptest.NewRecorder()
        handleDownload(rec, req)

        if rec.Code != http.StatusInternalServerError {
                t.Fatalf("status = %d, 기대값 500", rec.Code)
        }
        var body map[string]string
        if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
                t.Fatalf("JSON 아님: %v", err)
        }
        for _, leak := range []string{"yt-dlp error", "없는-yt-dlp", ":\\", "exec:"} {
                if strings.Contains(body["error"], leak) {
                        t.Fatalf("응답에 내부 정보가 노출됨(%q): %q", leak, body["error"])
                }
        }
        t.Logf("사용자에게 보이는 문구: %s", body["error"])
}
