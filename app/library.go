package main

import (
        "encoding/json"
        "fmt"
        "log"
        "os"
        "path/filepath"
        "regexp"
        "runtime"
        "sort"
        "strings"
        "sync"
        "time"
)

const (
        archiveFolderName = "음악 보관함"
        indexFileName     = "music-index.json"
        settingsFileName  = "settings.json"
)

// 바탕화면에 남길 기본값.
// 화면 배율이 150%면 바탕화면에 한눈에 보이는 아이콘이 10~20개 남짓이다.
// 날짜 조건만 걸면 많이 받은 달에 화면이 넘치므로 곡 수 상한을 함께 둔다.
const (
        defaultKeepCount = 12
        defaultKeepDays  = 30
)

// Settings는 exe 옆 settings.json 으로 노출된다. 없으면 기본값으로 만들어 준다.
type Settings struct {
        DesktopKeepCount int `json:"desktopKeepCount"`
        DesktopKeepDays  int `json:"desktopKeepDays"`
}

// LibraryEntry는 이 프로그램이 만든 파일 하나를 가리킨다.
// 정리할 때 이 목록에 있는 파일만 옮긴다 — 어무이가 직접 둔 파일은 건드리지 않는다.
type LibraryEntry struct {
        VideoID    string    `json:"videoId"`
        Title      string    `json:"title"`
        Filename   string    `json:"filename"`
        Path       string    `json:"path"`
        Downloaded time.Time `json:"downloaded"`
}

type Library struct {
        mu       sync.Mutex
        entries  []LibraryEntry
        settings Settings
}

var library = &Library{settings: Settings{
        DesktopKeepCount: defaultKeepCount,
        DesktopKeepDays:  defaultKeepDays,
}}

// YouTube 영상 ID. 같은 영상을 또 받지 않기 위한 열쇠다.
var videoIDRe = regexp.MustCompile(`(?:v=|youtu\.be/|/shorts/|/embed/)([A-Za-z0-9_-]{11})`)

func extractVideoID(rawURL string) string {
        if m := videoIDRe.FindStringSubmatch(rawURL); len(m) == 2 {
                return m[1]
        }
        return ""
}

// samePath는 두 경로가 같은 곳을 가리키는지 본다.
//
// Windows 파일 시스템은 대소문자를 구분하지 않는다. 문자열로만 비교하면
// "...\desktop" 과 "...\Desktop" 을 다른 곳으로 보고, 제자리에 있는 파일을
// 옮기려 하다가 이름이 겹쳐 "노래 (2).mp3" 로 바꿔 버린다.
func samePath(a, b string) bool {
        a, b = filepath.Clean(a), filepath.Clean(b)
        if runtime.GOOS == "windows" {
                return strings.EqualFold(a, b)
        }
        return a == b
}

// ===== 설정 / 목록 파일 =====

func (l *Library) load() {
        l.mu.Lock()
        defer l.mu.Unlock()

        l.settings = loadSettings()

        data, err := os.ReadFile(filepath.Join(BaseDir, indexFileName))
        if err != nil {
                return // 첫 실행
        }
        if err := json.Unmarshal(data, &l.entries); err != nil {
                log.Printf("[WARN] 음악 목록을 읽을 수 없습니다(새로 시작): %v", err)
                l.entries = nil
        }
        log.Printf("[INFO] 음악 목록 %d곡, 바탕화면 유지 기준 = 최근 %d일 · 최대 %d곡",
                len(l.entries), l.settings.DesktopKeepDays, l.settings.DesktopKeepCount)
}

func loadSettings() Settings {
        s := Settings{DesktopKeepCount: defaultKeepCount, DesktopKeepDays: defaultKeepDays}
        path := filepath.Join(BaseDir, settingsFileName)

        data, err := os.ReadFile(path)
        if err != nil {
                // 아들이 열어서 고칠 수 있도록 기본값 파일을 만들어 둔다.
                if out, err := json.MarshalIndent(s, "", "  "); err == nil {
                        os.WriteFile(path, out, 0644)
                }
                return s
        }
        if err := json.Unmarshal(data, &s); err != nil {
                log.Printf("[WARN] settings.json 을 읽을 수 없어 기본값을 씁니다: %v", err)
                return Settings{DesktopKeepCount: defaultKeepCount, DesktopKeepDays: defaultKeepDays}
        }
        if s.DesktopKeepCount <= 0 {
                s.DesktopKeepCount = defaultKeepCount
        }
        if s.DesktopKeepDays <= 0 {
                s.DesktopKeepDays = defaultKeepDays
        }
        return s
}

// saveLocked는 호출자가 이미 잠금을 잡고 있다고 가정한다.
func (l *Library) saveLocked() {
        out, err := json.MarshalIndent(l.entries, "", "  ")
        if err != nil {
                return
        }
        path := filepath.Join(BaseDir, indexFileName)
        tmp := path + ".tmp"
        if err := os.WriteFile(tmp, out, 0644); err != nil {
                log.Printf("[WARN] 음악 목록 저장 실패: %v", err)
                return
        }
        if err := os.Rename(tmp, path); err != nil {
                os.Remove(tmp)
                log.Printf("[WARN] 음악 목록 저장 실패: %v", err)
        }
}

// ===== 중복 확인 =====

// find는 이미 받아 둔 같은 영상을 찾는다.
// 파일이 지워졌으면 목록에서 빼고 없는 것으로 취급한다 — 다시 받을 수 있어야 한다.
func (l *Library) find(videoID string) (LibraryEntry, bool) {
        if videoID == "" {
                return LibraryEntry{}, false
        }
        l.mu.Lock()
        defer l.mu.Unlock()

        for i := range l.entries {
                if l.entries[i].VideoID != videoID {
                        continue
                }
                if _, err := os.Stat(l.entries[i].Path); err == nil {
                        return l.entries[i], true
                }
                // 어무이가 지우셨다 — 목록에서 빼고 다시 받게 한다.
                log.Printf("[INFO] 목록에 있으나 파일이 없어 다시 받습니다: %s", l.entries[i].Path)
                l.entries = append(l.entries[:i], l.entries[i+1:]...)
                l.saveLocked()
                return LibraryEntry{}, false
        }
        return LibraryEntry{}, false
}

func (l *Library) add(e LibraryEntry) {
        l.mu.Lock()
        l.entries = append(l.entries, e)
        l.saveLocked()
        l.mu.Unlock()
}

// ===== 정리 =====

// archiveDir는 받은 시점에 해당하는 보관 폴더 경로를 만든다.
// 월을 0으로 채워야 이름순 정렬이 곧 시간순이 된다 (2026년 07월 < 2026년 10월).
func archiveDir(t time.Time) string {
        return filepath.Join(DesktopDir, archiveFolderName,
                fmt.Sprintf("%d년 %02d월", t.Year(), int(t.Month())))
}

// organize는 바탕화면에 최근 곡만 남기고 나머지를 보관함으로 내린다.
// 목록에 있는 파일만 옮기므로 어무이가 직접 둔 파일은 절대 건드리지 않는다.
func (l *Library) organize() {
        l.mu.Lock()
        defer l.mu.Unlock()

        // 최신순. 같은 시각이면 순서를 유지한다.
        sort.SliceStable(l.entries, func(i, j int) bool {
                return l.entries[i].Downloaded.After(l.entries[j].Downloaded)
        })

        cutoff := time.Now().AddDate(0, 0, -l.settings.DesktopKeepDays)
        var kept []LibraryEntry
        moved, vanished := 0, 0

        for i := range l.entries {
                e := l.entries[i]

                if _, err := os.Stat(e.Path); err != nil {
                        // 어무이가 옮기셨거나 지우셨다. 목록에서 뺀다.
                        vanished++
                        continue
                }

                // 최근 N일 이내이면서 최신 N곡 안에 들어야 바탕화면에 남는다.
                onDesktop := i < l.settings.DesktopKeepCount && e.Downloaded.After(cutoff)

                wantDir := DesktopDir
                if !onDesktop {
                        wantDir = archiveDir(e.Downloaded)
                }

                if !samePath(filepath.Dir(e.Path), wantDir) {
                        newPath, err := moveInto(e.Path, wantDir)
                        if err != nil {
                                // 재생 중이라 잠겨 있을 수 있다. 다음 정리 때 다시 시도한다.
                                log.Printf("[WARN] 옮기지 못했습니다(%s): %v", e.Filename, err)
                                kept = append(kept, e)
                                continue
                        }
                        e.Path = newPath
                        e.Filename = filepath.Base(newPath)
                        moved++
                }
                kept = append(kept, e)
        }

        if moved > 0 || vanished > 0 {
                log.Printf("[INFO] 음악 정리: %d곡 이동, %d곡은 파일이 없어 목록에서 제외", moved, vanished)
        }
        l.entries = kept
        l.saveLocked()
}

// moveInto는 파일을 dir 로 옮기고 최종 경로를 돌려준다.
func moveInto(src, dir string) (string, error) {
        if err := os.MkdirAll(dir, 0755); err != nil {
                return "", err
        }
        dst := uniquePath(dir, filepath.Base(src))

        if err := os.Rename(src, dst); err == nil {
                return dst, nil
        }
        // 다른 드라이브면 rename 이 안 된다. 복사 후 원본을 지운다.
        if err := copyFile(src, dst); err != nil {
                return "", err
        }
        if err := os.Remove(src); err != nil {
                log.Printf("[WARN] 원본을 지우지 못했습니다: %v", err)
        }
        return dst, nil
}

// uniquePath는 같은 이름이 있으면 "제목 (2).mp3" 처럼 번호를 붙인다.
func uniquePath(dir, name string) string {
        ext := filepath.Ext(name)
        base := strings.TrimSuffix(name, ext)

        candidate := filepath.Join(dir, name)
        for n := 2; ; n++ {
                if _, err := os.Stat(candidate); os.IsNotExist(err) {
                        return candidate
                }
                candidate = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, n, ext))
        }
}

// ===== 기존 파일 한 번에 정리 =====

// adopt는 목록에 없는 바탕화면의 MP3 를 관리 대상으로 등록한다.
// 파일을 직접 옮기지는 않는다 — 배치는 organize 가 하므로
// 평소와 똑같은 규칙(최근 N일 · 최대 N곡)이 적용되고, 최근 곡은 바탕화면에 남는다.
// 트레이 메뉴에서 사람이 직접 실행한다. 반환값은 (등록 수, 실패 수).
func (l *Library) adoptLooseDesktopFiles() (int, int) {
        known := l.knownPaths()

        entries, err := os.ReadDir(DesktopDir)
        if err != nil {
                log.Printf("[WARN] 바탕화면을 읽을 수 없습니다: %v", err)
                return 0, 0
        }

        l.mu.Lock()
        defer l.mu.Unlock()

        added, failed := 0, 0
        for _, de := range entries {
                if de.IsDir() || !strings.EqualFold(filepath.Ext(de.Name()), ".mp3") {
                        continue
                }
                path := filepath.Join(DesktopDir, de.Name())
                if known[path] {
                        continue
                }
                info, err := de.Info()
                if err != nil {
                        failed++
                        continue
                }
                // 언제 받았는지 알 수 없으니 파일 날짜를 쓴다.
                // VideoID는 비워 둔다 — 중복 판정에는 쓰이지 않는다.
                l.entries = append(l.entries, LibraryEntry{
                        Title:      strings.TrimSuffix(de.Name(), filepath.Ext(de.Name())),
                        Filename:   de.Name(),
                        Path:       path,
                        Downloaded: info.ModTime(),
                })
                added++
        }
        l.saveLocked()

        log.Printf("[INFO] 바탕화면 기존 MP3 %d곡을 관리 대상에 등록 (실패 %d곡)", added, failed)
        return added, failed
}

// SongView는 웹 화면에 보여 줄 한 곡의 정보다.
type SongView struct {
        Title      string `json:"title"`
        Path       string `json:"path"`
        Where      string `json:"where"`      // "바탕화면" 또는 "2026년 07월"
        Downloaded string `json:"downloaded"` // 2026-07-29
}

// page는 최신순으로 정렬한 곡 목록의 일부를 돌려준다.
// 수백 곡이어도 첫 화면이 짧도록 나눠서 보낸다.
func (l *Library) page(offset, limit int) ([]SongView, int) {
        l.mu.Lock()
        defer l.mu.Unlock()

        sort.SliceStable(l.entries, func(i, j int) bool {
                return l.entries[i].Downloaded.After(l.entries[j].Downloaded)
        })

        total := len(l.entries)
        if offset < 0 || offset >= total {
                return []SongView{}, total
        }
        end := offset + limit
        if limit <= 0 || end > total {
                end = total
        }

        out := make([]SongView, 0, end-offset)
        for _, e := range l.entries[offset:end] {
                where := "바탕화면"
                if dir := filepath.Dir(e.Path); !samePath(dir, DesktopDir) {
                        where = filepath.Base(dir)
                }
                out = append(out, SongView{
                        Title:      strings.TrimSuffix(e.Filename, filepath.Ext(e.Filename)),
                        Path:       e.Path,
                        Where:      where,
                        Downloaded: e.Downloaded.Format("2006-01-02"),
                })
        }
        return out, total
}

// hasPath는 이 경로가 우리가 관리하는 곡인지 확인한다.
// 재생 요청에서 아무 경로나 열어 주지 않기 위한 검사다.
func (l *Library) hasPath(path string) bool {
        if path == "" {
                return false
        }
        l.mu.Lock()
        defer l.mu.Unlock()

        for i := range l.entries {
                if samePath(l.entries[i].Path, path) {
                        return true
                }
        }
        return false
}

// counts는 지금 바탕화면에 있는 곡 수와 보관함에 있는 곡 수를 센다.
func (l *Library) counts() (onDesktop, archived int) {
        l.mu.Lock()
        defer l.mu.Unlock()

        for i := range l.entries {
                if samePath(filepath.Dir(l.entries[i].Path), DesktopDir) {
                        onDesktop++
                } else {
                        archived++
                }
        }
        return
}

// countLooseDesktopMP3s는 정리 대상이 몇 곡인지 미리 센다 (확인 창에 쓴다).
func (l *Library) countLooseDesktopMP3s() int {
        known := l.knownPaths()

        entries, err := os.ReadDir(DesktopDir)
        if err != nil {
                return 0
        }
        n := 0
        for _, de := range entries {
                if de.IsDir() || !strings.EqualFold(filepath.Ext(de.Name()), ".mp3") {
                        continue
                }
                if !known[filepath.Join(DesktopDir, de.Name())] {
                        n++
                }
        }
        return n
}

func (l *Library) knownPaths() map[string]bool {
        l.mu.Lock()
        defer l.mu.Unlock()

        known := make(map[string]bool, len(l.entries))
        for i := range l.entries {
                known[l.entries[i].Path] = true
        }
        return known
}

// musicFolder는 트레이 "음악 폴더 열기"가 열 위치다.
func musicFolder() string {
        dir := filepath.Join(DesktopDir, archiveFolderName)
        if _, err := os.Stat(dir); err == nil {
                return dir
        }
        return DesktopDir
}
