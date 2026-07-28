package main

import (
        "encoding/json"
        "net/http"
        "net/http/httptest"
        "os"
        "path/filepath"
        "strings"
        "testing"
        "time"
)

// 화면에서 부르는 API 는 전역 library 를 쓰므로 테스트용으로 갈아 끼운다.
func withTestLibrary(t *testing.T, keepCount int) *Library {
        t.Helper()

        oldBase, oldDesktop, oldLib := BaseDir, DesktopDir, library
        BaseDir = t.TempDir()
        DesktopDir = t.TempDir()
        library = &Library{settings: Settings{
                DesktopKeepCount: keepCount,
                DesktopKeepDays:  30,
        }}
        t.Cleanup(func() {
                BaseDir, DesktopDir, library = oldBase, oldDesktop, oldLib
        })
        return library
}

func uiRequest(method, target string) *http.Request {
        req := httptest.NewRequest(method, target, nil)
        req.Header.Set(clientHeader, "1")
        return req
}

// 목록에 없는 파일은 절대 열어 주면 안 된다.
// 열어 준다면 로컬 서버가 임의 파일 실행 통로가 된다.
func TestPlayRejectsFilesOutsideLibrary(t *testing.T) {
        l := withTestLibrary(t, 12)

        // 관리 대상 한 곡
        managed := filepath.Join(DesktopDir, "내노래.mp3")
        if err := os.WriteFile(managed, []byte("mp3"), 0644); err != nil {
                t.Fatal(err)
        }
        l.entries = append(l.entries, LibraryEntry{
                Title: "내노래", Filename: "내노래.mp3", Path: managed, Downloaded: time.Now(),
        })

        // 실제로 존재하지만 목록에는 없는 파일
        outsider := filepath.Join(t.TempDir(), "남의파일.mp3")
        if err := os.WriteFile(outsider, []byte("mp3"), 0644); err != nil {
                t.Fatal(err)
        }

        denied := []struct{ name, path string }{
                {"목록에 없는 파일", outsider},
                {"시스템 파일", `C:\Windows\System32\calc.exe`},
                {"상위 경로 탈출", filepath.Join(DesktopDir, "..", "..", "secret.txt")},
                {"빈 경로", ""},
        }
        for _, c := range denied {
                t.Run(c.name, func(t *testing.T) {
                        rec := httptest.NewRecorder()
                        handlePlay(rec, uiRequest(http.MethodGet, "/api/play?path="+c.path))

                        if rec.Code != http.StatusForbidden {
                                t.Fatalf("status = %d, 기대값 403 (경로: %q)", rec.Code, c.path)
                        }
                })
        }

        // 관리 대상은 통과해야 한다. 여기서는 실제로 앱이 열리므로
        // 경로 검사만 확인하고 실행 여부는 보지 않는다.
        if !l.hasPath(managed) {
                t.Fatal("관리 중인 곡을 hasPath 가 못 찾음")
        }
}

// 우리 화면이 아닌 곳에서 온 요청은 막아야 한다.
func TestUIEndpointsRejectForgedRequests(t *testing.T) {
        withTestLibrary(t, 12)

        handlers := map[string]http.HandlerFunc{
                "/api/songs":       handleSongs,
                "/api/play":        handlePlay,
                "/api/open-folder": handleOpenFolder,
                "/api/tidy":        handleTidy,
        }
        for path, handler := range handlers {
                t.Run(path, func(t *testing.T) {
                        // 헤더도 Origin 도 없는 요청 (<img> 등)
                        rec := httptest.NewRecorder()
                        handler(rec, httptest.NewRequest(http.MethodGet, path, nil))
                        if rec.Code != http.StatusForbidden {
                                t.Fatalf("헤더 없음: status = %d, 기대값 403", rec.Code)
                        }

                        // 악성 사이트 Origin
                        req := httptest.NewRequest(http.MethodGet, path, nil)
                        req.Header.Set("Origin", "https://evil.com")
                        rec2 := httptest.NewRecorder()
                        handler(rec2, req)
                        if rec2.Code != http.StatusForbidden {
                                t.Fatalf("악성 origin: status = %d, 기대값 403", rec2.Code)
                        }
                })
        }
}

// 수백 곡이어도 첫 화면이 짧도록 나눠서 와야 한다.
func TestSongsPagination(t *testing.T) {
        l := withTestLibrary(t, 12)

        const count = 55
        for i := 0; i < count; i++ {
                name := "곡" + string(rune('A'+i%26)) + string(rune('a'+i/26)) + ".mp3"
                path := filepath.Join(DesktopDir, name)
                if err := os.WriteFile(path, []byte("mp3"), 0644); err != nil {
                        t.Fatal(err)
                }
                l.entries = append(l.entries, LibraryEntry{
                        Title:      strings.TrimSuffix(name, ".mp3"),
                        Filename:   name,
                        Path:       path,
                        Downloaded: time.Now().Add(-time.Duration(i) * time.Hour),
                })
        }

        type resp struct {
                Songs []SongView `json:"songs"`
                Total int        `json:"total"`
        }

        get := func(target string) resp {
                t.Helper()
                rec := httptest.NewRecorder()
                handleSongs(rec, uiRequest(http.MethodGet, target))
                if rec.Code != http.StatusOK {
                        t.Fatalf("status = %d", rec.Code)
                }
                var r resp
                if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
                        t.Fatalf("JSON 아님: %v", err)
                }
                return r
        }

        first := get("/api/songs?offset=0&limit=20")
        if first.Total != count {
                t.Fatalf("total = %d, 기대값 %d", first.Total, count)
        }
        if len(first.Songs) != 20 {
                t.Fatalf("첫 페이지 %d곡, 기대값 20곡", len(first.Songs))
        }

        // 최신순이어야 한다.
        if first.Songs[0].Downloaded < first.Songs[19].Downloaded {
                t.Error("최신순 정렬이 아님")
        }

        // 마지막 페이지는 남은 만큼만.
        last := get("/api/songs?offset=40&limit=20")
        if len(last.Songs) != 15 {
                t.Fatalf("마지막 페이지 %d곡, 기대값 15곡", len(last.Songs))
        }

        // 범위를 넘으면 빈 목록. 오류가 아니어야 한다.
        beyond := get("/api/songs?offset=999&limit=20")
        if len(beyond.Songs) != 0 {
                t.Fatalf("범위 밖에서 %d곡", len(beyond.Songs))
        }

        // 페이지를 이어 붙이면 중복 없이 전체가 나와야 한다.
        seen := map[string]bool{}
        for off := 0; off < count; off += 20 {
                for _, s := range get("/api/songs?offset="+itoa(off)+"&limit=20").Songs {
                        if seen[s.Path] {
                                t.Fatalf("중복된 곡: %s", s.Path)
                        }
                        seen[s.Path] = true
                }
        }
        if len(seen) != count {
                t.Fatalf("전체 %d곡, 기대값 %d곡", len(seen), count)
        }
}

// 곡이 어디 있는지(바탕화면 / 월별 폴더)를 화면에 표시할 수 있어야 한다.
func TestSongViewShowsLocation(t *testing.T) {
        l := withTestLibrary(t, 1)

        for i, name := range []string{"남을곡.mp3", "내려갈곡.mp3"} {
                path := filepath.Join(DesktopDir, name)
                if err := os.WriteFile(path, []byte("mp3"), 0644); err != nil {
                        t.Fatal(err)
                }
                l.entries = append(l.entries, LibraryEntry{
                        Title:      strings.TrimSuffix(name, ".mp3"),
                        Filename:   name,
                        Path:       path,
                        Downloaded: time.Now().Add(-time.Duration(i) * time.Hour),
                })
        }
        l.organize()

        songs, _ := l.page(0, 10)
        if len(songs) != 2 {
                t.Fatalf("%d곡", len(songs))
        }
        if songs[0].Where != "바탕화면" {
                t.Errorf("최근 곡 위치 = %q, 기대값 바탕화면", songs[0].Where)
        }
        if !strings.Contains(songs[1].Where, "월") {
                t.Errorf("내려간 곡 위치 = %q, 월별 폴더명이어야 함", songs[1].Where)
        }
}

func itoa(n int) string {
        if n == 0 {
                return "0"
        }
        var b []byte
        for n > 0 {
                b = append([]byte{byte('0' + n%10)}, b...)
                n /= 10
        }
        return string(b)
}
