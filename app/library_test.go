package main

import (
        "os"
        "path/filepath"
        "strings"
        "testing"
        "time"
)

// 테스트용으로 BaseDir/DesktopDir 를 임시 폴더로 바꾸고 빈 라이브러리를 만든다.
func newTestLibrary(t *testing.T, keepCount, keepDays int) *Library {
        t.Helper()

        oldBase, oldDesktop := BaseDir, DesktopDir
        BaseDir = t.TempDir()
        DesktopDir = t.TempDir()
        t.Cleanup(func() { BaseDir, DesktopDir = oldBase, oldDesktop })

        return &Library{settings: Settings{
                DesktopKeepCount: keepCount,
                DesktopKeepDays:  keepDays,
        }}
}

// 바탕화면에 곡 파일을 만들고 라이브러리에 등록한다.
func addSong(t *testing.T, l *Library, name string, age time.Duration) string {
        t.Helper()

        path := filepath.Join(DesktopDir, name)
        if err := os.WriteFile(path, []byte("mp3"), 0644); err != nil {
                t.Fatalf("파일 생성 실패: %v", err)
        }
        l.entries = append(l.entries, LibraryEntry{
                VideoID:    strings.TrimSuffix(name, ".mp3"),
                Title:      strings.TrimSuffix(name, ".mp3"),
                Filename:   name,
                Path:       path,
                Downloaded: time.Now().Add(-age),
        })
        return path
}

func desktopMP3Count(t *testing.T) int {
        t.Helper()

        entries, err := os.ReadDir(DesktopDir)
        if err != nil {
                t.Fatalf("바탕화면 읽기 실패: %v", err)
        }
        n := 0
        for _, e := range entries {
                if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".mp3") {
                        n++
                }
        }
        return n
}

func TestExtractVideoID(t *testing.T) {
        cases := map[string]string{
                "https://www.youtube.com/watch?v=aqz-KE-bpKQ":          "aqz-KE-bpKQ",
                "https://youtu.be/aqz-KE-bpKQ":                         "aqz-KE-bpKQ",
                "https://www.youtube.com/shorts/aqz-KE-bpKQ":           "aqz-KE-bpKQ",
                "https://www.youtube.com/embed/aqz-KE-bpKQ":            "aqz-KE-bpKQ",
                "https://www.youtube.com/watch?v=aqz-KE-bpKQ&t=42s":    "aqz-KE-bpKQ",
                "https://m.youtube.com/watch?app=desktop&v=aqz-KE-bpKQ": "aqz-KE-bpKQ",
                "https://example.com/nope":                             "",
        }
        for url, want := range cases {
                if got := extractVideoID(url); got != want {
                        t.Errorf("extractVideoID(%q) = %q, 기대값 %q", url, got, want)
                }
        }
}

// 화면 배율이 커서 보이는 아이콘이 적다. 날짜가 최근이어도 개수 상한을 넘으면 내려가야 한다.
func TestOrganizeCapsDesktopCount(t *testing.T) {
        l := newTestLibrary(t, 12, 30)

        // 전부 오늘 받은 곡 20개 — 날짜 조건만으로는 하나도 안 내려간다.
        for i := 0; i < 20; i++ {
                addSong(t, l, "노래"+string(rune('A'+i))+".mp3", time.Hour)
        }

        l.organize()

        if n := desktopMP3Count(t); n != 12 {
                t.Fatalf("바탕화면에 %d곡, 기대값 12곡", n)
        }
        if len(l.entries) != 20 {
                t.Fatalf("목록에서 곡이 사라짐: %d개", len(l.entries))
        }

        // 내려간 8곡은 보관함 안에 있어야 한다.
        archived := 0
        for _, e := range l.entries {
                if strings.Contains(e.Path, archiveFolderName) {
                        archived++
                        if _, err := os.Stat(e.Path); err != nil {
                                t.Errorf("보관함 파일이 없음: %s", e.Path)
                        }
                }
        }
        if archived != 8 {
                t.Fatalf("보관함에 %d곡, 기대값 8곡", archived)
        }
}

// 개수가 남아도 30일이 지났으면 내려가야 한다.
func TestOrganizeAppliesAgeLimit(t *testing.T) {
        l := newTestLibrary(t, 12, 30)

        addSong(t, l, "최근곡.mp3", 24*time.Hour)
        addSong(t, l, "오래된곡.mp3", 60*24*time.Hour)

        l.organize()

        if n := desktopMP3Count(t); n != 1 {
                t.Fatalf("바탕화면에 %d곡, 기대값 1곡", n)
        }
        for _, e := range l.entries {
                inArchive := strings.Contains(e.Path, archiveFolderName)
                if e.Title == "오래된곡" && !inArchive {
                        t.Error("오래된 곡이 바탕화면에 남아 있음")
                }
                if e.Title == "최근곡" && inArchive {
                        t.Error("최근 곡이 보관함으로 내려감")
                }
        }
}

// 보관 폴더는 받은 달 기준이고, 이름순 정렬이 곧 시간순이어야 한다.
func TestArchiveDirNaming(t *testing.T) {
        oldDesktop := DesktopDir
        DesktopDir = t.TempDir()
        defer func() { DesktopDir = oldDesktop }()

        got := filepath.Base(archiveDir(time.Date(2026, 7, 28, 0, 0, 0, 0, time.Local)))
        if got != "2026년 07월" {
                t.Fatalf("폴더명 = %q, 기대값 %q", got, "2026년 07월")
        }
        // 0을 채우지 않으면 이름순에서 10월이 7월보다 앞선다.
        oct := filepath.Base(archiveDir(time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local)))
        if !(got < oct) {
                t.Fatalf("이름순 정렬이 시간순과 다름: %q >= %q", got, oct)
        }
}

// 이 프로그램이 만들지 않은 파일은 자동 정리에서 절대 건드리면 안 된다.
func TestOrganizeNeverTouchesUnknownFiles(t *testing.T) {
        l := newTestLibrary(t, 1, 30)

        addSong(t, l, "내려갈곡.mp3", 2*time.Hour)
        addSong(t, l, "남을곡.mp3", time.Hour)

        // 어무이가 직접 둔 파일들
        others := []string{"가족사진.jpg", "메모.txt", "직접넣은음악.mp3"}
        for _, name := range others {
                if err := os.WriteFile(filepath.Join(DesktopDir, name), []byte("x"), 0644); err != nil {
                        t.Fatal(err)
                }
        }

        l.organize()

        for _, name := range others {
                if _, err := os.Stat(filepath.Join(DesktopDir, name)); err != nil {
                        t.Errorf("목록에 없는 파일이 사라짐: %s", name)
                }
        }
}

// 어무이가 파일을 지우셨으면 목록에서 빠지고 다시 받을 수 있어야 한다.
func TestFindDropsMissingFile(t *testing.T) {
        l := newTestLibrary(t, 12, 30)
        path := addSong(t, l, "지운곡.mp3", time.Hour)

        if _, ok := l.find("지운곡"); !ok {
                t.Fatal("있는 곡을 못 찾음")
        }
        if err := os.Remove(path); err != nil {
                t.Fatal(err)
        }
        if _, ok := l.find("지운곡"); ok {
                t.Fatal("파일이 없는데 중복으로 판단함 — 다시 받을 수 없게 됨")
        }
        if len(l.entries) != 0 {
                t.Fatalf("목록에서 안 빠짐: %d개", len(l.entries))
        }
}

// 기존에 흩어진 MP3만 관리 대상에 넣고, 다른 파일은 놔둬야 한다.
func TestAdoptLooseDesktopFiles(t *testing.T) {
        l := newTestLibrary(t, 12, 30)
        managed := addSong(t, l, "관리중인곡.mp3", time.Hour)

        loose := []string{"옛날노래1.mp3", "옛날노래2.MP3"}
        for _, name := range loose {
                if err := os.WriteFile(filepath.Join(DesktopDir, name), []byte("x"), 0644); err != nil {
                        t.Fatal(err)
                }
        }
        if err := os.WriteFile(filepath.Join(DesktopDir, "문서.txt"), []byte("x"), 0644); err != nil {
                t.Fatal(err)
        }

        if n := l.countLooseDesktopMP3s(); n != 2 {
                t.Fatalf("정리 대상 %d개, 기대값 2개", n)
        }

        added, failed := l.adoptLooseDesktopFiles()
        if added != 2 || failed != 0 {
                t.Fatalf("등록 %d개 실패 %d개, 기대값 2/0", added, failed)
        }
        if len(l.entries) != 3 {
                t.Fatalf("목록 %d개, 기대값 3개", len(l.entries))
        }
        // 이미 관리 중인 곡을 두 번 등록하면 안 된다.
        if again, _ := l.adoptLooseDesktopFiles(); again != 0 {
                t.Fatalf("같은 파일을 %d개 다시 등록함", again)
        }
        if _, err := os.Stat(managed); err != nil {
                t.Error("관리 중인 곡을 건드림")
        }
        if _, err := os.Stat(filepath.Join(DesktopDir, "문서.txt")); err != nil {
                t.Error("MP3가 아닌 파일을 건드림")
        }
}

// 일괄 정리 후에도 평소 규칙이 그대로 적용되어야 한다 —
// 전부 보관함으로 들어가 바탕화면이 텅 비면 안 된다.
func TestAdoptThenOrganizeKeepsRecentOnDesktop(t *testing.T) {
        l := newTestLibrary(t, 12, 30)

        // 어무이가 이미 갖고 계신 100곡: 절반은 최근, 절반은 오래된 것
        for i := 0; i < 100; i++ {
                age := time.Duration(i) * 6 * time.Hour
                path := filepath.Join(DesktopDir, "옛곡_"+string(rune('A'+i%26))+string(rune('a'+i/26))+".mp3")
                if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
                        t.Fatal(err)
                }
                if err := os.Chtimes(path, time.Now().Add(-age), time.Now().Add(-age)); err != nil {
                        t.Fatal(err)
                }
        }

        l.adoptLooseDesktopFiles()
        l.organize()

        if n := desktopMP3Count(t); n != 12 {
                t.Fatalf("정리 후 바탕화면에 %d곡, 기대값 12곡 (0곡이면 다 사라진 것)", n)
        }
        onDesktop, archived := l.counts()
        if onDesktop != 12 || archived != 88 {
                t.Fatalf("바탕화면 %d곡 / 보관함 %d곡, 기대값 12 / 88", onDesktop, archived)
        }
}

// 같은 이름이 있으면 덮어쓰지 않고 번호를 붙여야 한다.
func TestUniquePathAvoidsOverwrite(t *testing.T) {
        dir := t.TempDir()
        if err := os.WriteFile(filepath.Join(dir, "노래.mp3"), []byte("원본"), 0644); err != nil {
                t.Fatal(err)
        }
        got := uniquePath(dir, "노래.mp3")
        if filepath.Base(got) != "노래 (2).mp3" {
                t.Fatalf("경로 = %q, 기대값 %q", filepath.Base(got), "노래 (2).mp3")
        }
        data, err := os.ReadFile(filepath.Join(dir, "노래.mp3"))
        if err != nil || string(data) != "원본" {
                t.Fatal("원본이 손상됨")
        }
}

// 정리를 여러 번 돌려도 결과가 달라지면 안 된다 (매 다운로드마다 실행된다).
func TestOrganizeIsIdempotent(t *testing.T) {
        l := newTestLibrary(t, 3, 30)
        for i := 0; i < 6; i++ {
                addSong(t, l, "곡"+string(rune('A'+i))+".mp3", time.Duration(i)*time.Hour)
        }

        l.organize()
        first := desktopMP3Count(t)
        paths := make([]string, len(l.entries))
        for i, e := range l.entries {
                paths[i] = e.Path
        }

        l.organize()
        l.organize()

        if n := desktopMP3Count(t); n != first {
                t.Fatalf("반복 실행으로 바탕화면 곡 수가 %d → %d 로 변함", first, n)
        }
        for i, e := range l.entries {
                if e.Path != paths[i] {
                        t.Fatalf("반복 실행으로 경로가 바뀜: %q → %q", paths[i], e.Path)
                }
        }
}
