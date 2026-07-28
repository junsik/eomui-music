---
Task ID: 1
Agent: Main Agent
Task: Chrome 확장 프로그램 + Next.js 랜딩 페이지 제작

Work Log:
- fullstack-dev 스킬 로드 및 프로젝트 초기화 완료
- Chrome 확장 프로그램 파일 5개 + 아이콘 3개 생성 (extension/ 폴더)
  - manifest.json (Manifest V3)
  - content.js (YouTube 페이지에 MP3 버튼 주입, SPA navigation 감지)
  - content.css (빨간색 pill 버튼, 토스트 메시지, 다크모드 호환)
  - background.js (Service Worker, fetch→Blob→chrome.downloads.download)
  - popup.html (사용법 3단계 안내)
  - icons/ (16x16, 48x48, 128x128 PNG)
- Python 스크립트로 확장 프로그램 아이콘 3종 생성
- Next.js 랜딩 페이지 제작 (src/app/page.tsx)
  - 히어로: "어무이를 위한 음악 다운로드" 타이틀
  - "왜 브라우저 확장인가요?" 설명 카드
  - 방법 1: 확장 프로그램 (추천) - 접었다 폈다 카드, YouTube 모킹 UI, 3단계 설치, 특징 3칸
  - 방법 2: 북마크릿 - 접었다 폈다 카드, 3단계 설치, 클립보드 복사 기능
  - 비교표 (접었다 폈다) - 확장 vs 북마크릿 vs 서버
  - 푸터: "직접 만든 무료 서비스 · 광고 없음 · 저작권 준수" (sticky)
- layout.tsx 한국어(lang="ko") 설정 및 메타데이터 업데이트
- 사용하지 않는 API 라우트 제거 (/api/route.ts)
- 확장 프로그램 ZIP 패키징 (download/eomui-music-downloader-extension.zip)
- ESLint 통과 (0 에러)
- Agent Browser로 렌더링 검증 완료 (모든 요소 정상, 콘솔 에러 0)

Stage Summary:
- Chrome 확장 프로그램 완성: extension/ 폴더 (5개 파일 + 아이콘)
- Next.js 랜딩 페이지 완성: 한국어, rose/red 테마, shadcn/ui 활용
- ZIP 패키지: download/eomui-music-downloader-extension.zip
- 렌더링 검증 통과, 에러 없음

---
Task ID: 2-fix-silent-go
Agent: Main Agent
Task: Go 버전이 여전히 "붙여넣기" HTML 페이지를 호출하는 문제 수정 + MP3 버튼 위치를 YouTube 액션바로 이동

Work Log:
- main.go에서 openBrowser() 호출 제거 (브라우저 자동 열기 안 함)
- handleHTML() 제거, handleRoot() 추가 — 붙여넣기 폼 대신 "서버 실행 중" 확인 페이지만 제공
- HtmlFile 변수 제거 (더 이상 HTML 파일 읽지 않음)
- 액션바 inline 버튼으로 content.js 재작성 — YouTube #top-level-buttons-computed 찾아서 append
- YouTube SPA 네비게이션 시 액션바가 늦게 나타나면 500ms 간격으로 30회 재시도
- content.css 재작성 — fixed positioning 제거, inline pill button 스타일 (height 42px, 빨강 gradient)
- 상태 표시 overlay는 화면 하단 중앙에 별도 fixed로 유지
- Go exe 재빌드: 5.6MB Windows amd64 (cross-compile from Linux)
- eomui-extension-v2.zip 재패키지 (9.9KB)
- eomui-standalone-go.zip 재패키지 (2.4MB)

Stage Summary:
- Go exe 더블클릭 시 더 이상 브라우저 자동 열리지 않음 (백그라운드에서 조용히 실행)
- 사용자가 수동으로 http://localhost:8080 방문해도 빨간 "MP3 다운로드" 폼 대신 "서버 실행 중" 안내 페이지만 표시
- YouTube 영상 페이지에서 MP3 버튼이 좋아요/공유 버튼 옆 액션바에 inline으로 표시됨 (빨간색으로 눈에 뜀)
- 파일 위치: /home/z/my-project/download/eomui-standalone-go.zip, /home/z/my-project/download/eomui-extension-v2.zip

---
Task ID: 3-fix-console-and-tray
Agent: Main Agent
Task: Go 서버가 콘솔로 뜨는 문제 + 확장이 Go 서버를 못 찾는 문제 + 백그라운드 종료 방법 제공

Work Log:
- main.go에 github.com/getlantern/systray 의존성 추가 (Windows용 순수 syscall, cgo 불필요)
- icon.ico 파일 생성 (기존 icon128.png를 PIL로 변환, 5개 사이즈 포함)
- //go:embed 지시어로 icon.ico를 exe에 임베드
- onTrayReady() / onTrayExit() 콜백 추가 — 트레이 메뉴: 브라우저 열기, 로그 폴더 열기, 종료
- main() 재구성 — HTTP 서버는 goroutine으로, systray.Run()이 메인 스레드 블록
- 모든 HTTP 핸들러에 addCORS() 추가 — Access-Control-Allow-Origin: * (확장 fetch 차단 해결)
- handleDownload에 OPTIONS preflight 처리 추가
- 빌드 플래그 변경: -ldflags="-s -w -H windowsgui" (콘솔 창 완전 제거)
- file 명령어로 확인: "PE32+ executable for MS Windows 6.01 (GUI)" — GUI subsystem 적용됨
- exe 크기: 5.6MB → 5.9MB (systray + icon 추가로 0.3MB 증가)
- extension-v2 패키징 수정: -j 플래그 대신 -r 플래그 사용하여 icons/ 하위 디렉터리 보존
  (이전 빌드는 icon16.png 등이 루트에 평면 배치되어 manifest의 "icons/icon16.png" 경로와 불일치)
- manifest version 1.2.0 → 1.3.0 으로 상향
- popup.html 사용법 텍스트 수정: "우측 상단" → "좋아요/공유 버튼 옆"
- README.md 재작성 — 트레이 아이콘 종료 방법, 문제 해결 섹션 추가

Stage Summary:
- Go exe 더블클릭 시 콘솔 창 뜨지 않음 (GUI subsystem)
- 작업 표시줄 트레이에 🎵 음표 아이콘으로 표시, 우클릭 → "종료"로 완전히 종료
- 모든 HTTP 응답에 CORS 헤더 추가되어 Chrome 확장에서 안정적 호출 가능
- 확장 ZIP 구조 수정 — icons/ 폴더 보존, manifest와 경로 일치
- 파일 위치: /home/z/my-project/download/eomui-standalone-go.zip (2.5MB), /home/z/my-project/download/eomui-extension-v2.zip (11KB)

---
Task ID: 4-package-source
Agent: Main Agent
Task: 소스 코드 ZIP 패키징

Work Log:
- /tmp/eomui-src/ 스테이징 디렉터리 생성
- extension-v2/ 소스 복사 (manifest, content.js, content.css, popup.html, icons/)
- standalone-go/ 소스 복사 (main.go, go.mod, go.sum, icon.ico, public/, README.md)
  - eomui-music.exe (5.9MB 빌드 산출물)은 제외
- standalone/ 소스 복사 (server.ts, build.bat, public/)
- scripts/generate_icons.py 복사
- worklog.md 복사
- 최상단 README.md 작성 — 디렉터리 구조, 빌드 방법, 아키텍처 다이어그램 포함
- zip -r로 폴더 구조 보존하며 패키징 (47KB)
- 검증: 압축 해제 시 18개 파일, 디렉터리 구조 정상

Stage Summary:
- 소스 ZIP 위치: /home/z/my-project/download/eomui-source.zip (47KB)
- exe 빌드 산출물 제외, 모든 소스 코드 + 빌드 스크립트 + 문서 포함
- README.md에 빌드 명령어 포함: GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -H windowsgui" -o eomui-music.exe .

---
Task ID: 5-auto-download-deps
Agent: Main Agent
Task: Go 서버 실행 시 yt-dlp / ffmpeg / deno 가 같은 폴더에 없으면 자동 다운로드

Work Log:
- bootstrap.go 신규 작성 — 실행 시점 의존성 자동 설치
  - ensureTools(): exe 폴더 확인 후 없는 것만 내려받음. goroutine으로 돌려 트레이 아이콘은 즉시 표시
  - downloadTo(): .part 파일에 먼저 쓰고 완료 후 rename → 중간에 끊겨도 반쪽 exe 안 남음
  - extractFromZip(): ZIP 엔트리 이름(path.Base)으로 찾아 추출.
    ffmpeg 는 ffmpeg-<버전>-essentials_build/bin/ffmpeg.exe, deno 는 루트 deno.exe — 둘 다 대응
  - progressWriter: io.Copy 경유 바이트를 세어 2초마다 진행률 갱신
  - renameWithRetry(): Windows 백신이 방금 쓴 exe 를 잠그는 현상 대응 (10회 백오프 재시도)
    — 실제로 4회 중 1회 재현되던 간헐 실패였음
  - deno 는 node/deno 둘 다 없을 때만 다운로드. 실패해도 치명적이지 않아 경고만 남기고 진행
- main.go 연결
  - FFmpeg 전역 추가, trayReady 채널 + setTrayTooltip() — 트레이 준비 전 툴팁 갱신 방지
  - handleDownload: 준비 완료 전 요청은 503 + 진행 상황 안내 메시지
  - handleStatus: ready / setup / ytDlp / ffmpeg 노출
  - 안내 페이지에 /api/status 폴링 스크립트 추가 — 준비 중이면 노란색 + 진행률 표시
  - --ffmpeg-location 인자 추가 — 내려받은 ffmpeg 는 PATH 에 없으므로 위치를 직접 전달
- 버그 수정: --js-runtimes 인자 형식 오류
  - yt-dlp 의 --js-runtimes 는 RUNTIME[:PATH] 형식인데 경로만 넘기고 있었음
  - yt-dlp 가 "C:\..." 를 ':' 로 쪼개 런타임 이름을 'c' 로 읽고
    "Ignoring unsupported JavaScript runtime(s): c" 경고와 함께 무시 → JS 런타임 탐지가 통째로 무효
  - getJsRuntime() 이 (이름, 경로) 를 반환하도록 변경, "node:C:\...\node.exe" 형태로 전달
  - yt-dlp 2026.07.04 로 실제 확인: 기존 형식은 경고 후 무시, 수정 형식은 정상 동작
  - JsRuntime/JsRuntimeName 은 준비 goroutine 이 나중에 바꾸므로 RWMutex 로 보호
- 정리: /home/z/.local/bin/yt-dlp 하드코딩 제거 (non-Windows 는 PATH 조회), go mod tidy
- 검증
  - go vet 통과, GOOS=windows 빌드 성공 (7.13MB)
  - 실제 배포 ZIP 2종(ffmpeg 110MB, deno 43MB)으로 extractFromZip 추출 테스트 6/6 통과
  - 실제 URL 에서 downloadTo 다운로드 테스트 통과 (18MB, .part 잔재 없음)
  - httptest 로 준비 전 503 게이팅 / status JSON 검증 통과
  - 검증용 테스트 파일은 확인 후 제거

Stage Summary:
- eomui-music.exe 하나만 배포하면 됨. 첫 실행 시 필요한 것만 자동으로 받음 (합계 약 170MB)
- 이미 폴더에 있는 파일은 건드리지 않음. Node.js 가 있으면 deno 는 받지 않음
- 진행률은 트레이 툴팁 + http://localhost:8080 페이지 + 로그 파일에 표시
- 준비 완료 전 MP3 버튼 클릭 시 503 과 함께 진행 상황을 안내
- --js-runtimes 형식 오류 수정으로 JS 런타임이 실제로 적용되기 시작함

---
Task ID: 6-review-fixes
Agent: Main Agent
Task: 코드 리뷰에서 나온 지적사항 개선

Work Log:
- 보안: 서버가 외부에 열려 있던 문제
  - net.Listen(":8080") → "127.0.0.1:8080". 기존에는 0.0.0.0 바인딩이라
    같은 공유기의 다른 기기가 바탕화면에 파일을 쓸 수 있었음
  - Access-Control-Allow-Origin: * 제거 → youtube.com 계열 + chrome-extension:// 만 허용
  - /api/download 에 CSRF 게이트 추가 — X-Eomui-Client 헤더 또는 허용된 Origin 필요.
    <img src="http://localhost:8080/api/download?url=..."> 는 둘 다 만족시킬 수 없음
  - content.js 에 X-Eomui-Client 헤더 추가, manifest 1.3.0 → 1.4.0
  - Origin 검사만으로는 부족 — <img>/최상위 이동은 Origin 을 아예 안 보냄.
    헤더 OR Origin 조건이라 예전 확장과도 호환됨
- 한국어 파일명 깨짐: name[:100] 바이트 절삭 → []rune 기준 절삭
  - 한글 3바이트라 33자 넘으면 글자 중간에서 잘려 깨진 파일명이 됨
  - 덤으로 Windows 예약어(CON/PRN/AUX/NUL/COM1-9/LPT1-9)와 끝의 마침표·공백 처리 추가
- CP949 디코더가 Windows 에서 죽은 코드였던 문제
  - iconv 를 shell 로 부르는데 Windows 에 iconv 가 없음.
    게다가 실패해도 error 가 nil 이라 깨진 문자열을 그대로 통과시킴
  - golang.org/x/text/encoding/korean 으로 교체 (EUCKR 은 CP949 구현)
  - x/text 는 v0.21.0 으로 고정 — 최신(v0.40.0)은 go 지시자를 1.25 로 올려 버림
- 안정성
  - runYtDlp: exec.CommandContext + 30분 타임아웃. 멈춘 yt-dlp 가 핸들러를 영원히 잡던 문제
  - 동시 다운로드 2개로 제한 (세마포어). 초과 시 429 + 안내. 버튼 연타 대응
  - cleanupOldFiles 가 진행 중인 임시 파일을 지우던 문제 — activeTmpFiles(sync.Map) 로 보호
  - 복사 폴백이 MP3 전체를 메모리에 올리던 것 → io.Copy
  - 다운로드 실패 시 반쪽 임시 파일 제거
- 정보 노출: 실패 응답에 yt-dlp stderr(로컬 경로 포함)가 그대로 나가던 것 →
  정해 둔 문구만 응답, 원본은 로그에만
- 정리: 미사용 appendStrings 제거, 타임아웃(504) 분기 추가
- main_test.go 신규 — 외부 의존성 없는 테스트 9종
  - 한글 절삭/파일명 규칙, CP949 디코딩, Origin 허용 목록
  - 위조 요청 403 거부 (헤더 없음 / 악성 origin / 유사 도메인)
  - 확장 요청 통과, preflight 처리, 준비 전 503, 오류 응답 경로 미노출
- go vet 통과, 전체 테스트 통과, exe 빌드 (7.26MB, PE Subsystem=2 GUI 확인)

Stage Summary:
- eomui-music.exe 빌드 완료: standalone-go/eomui-music.exe (7.26MB)
- 확장 프로그램은 v1.4.0 이상 필요 — 서버가 X-Eomui-Client 헤더를 확인함
- 외부 기기·타 웹사이트에서의 호출 차단, 한글 파일명 정상화, yt-dlp 폭주 방지

---
Task ID: 7-console-output
Agent: Main Agent
Task: 로그를 파일뿐 아니라 콘솔로도 출력

Work Log:
- console_windows.go / console_other.go 신규 (빌드 태그 분리)
  - attachConsole(): 표준 출력이 이미 유효하면(터미널 실행·파이프 리다이렉트) 그대로 사용
  - 아니면 AttachConsole(ATTACH_PARENT_PROCESS) 로 부모 콘솔에 붙음
  - -console 옵션이면 AllocConsole() 로 창을 새로 띄움 (더블클릭 실행용)
  - SetConsoleOutputCP(65001) — 콘솔 한글 깨짐 방지
  - GUI 빌드는 표준 핸들이 비어 있어 CONOUT$ 를 직접 열어 os.Stdout/Stderr 교체
- setupLogging: 콘솔 + 파일 동시 출력
  - io.MultiWriter 는 첫 쓰기 실패에서 멈추므로, 콘솔이 없을 때 os.Stdout 을 넣으면
    파일 기록까지 같이 죽는다. consoleAttached 일 때만 stdout 을 대상에 추가
  - 로그 파일을 못 열어도 콘솔로는 나가도록 분기 정리
- -H windowsgui 는 유지 — 어무이가 더블클릭하면 여전히 콘솔 창 없음
- 검증 중 발견한 버그: 포트 바인딩 실패가 조용히 묻힘
  - 8080 이 이미 사용 중이면 서버가 안 뜨는데 트레이 툴팁은 "준비 완료 — 실행 중"
  - 준비 goroutine 이 나중에 끝나며 오류 문구를 덮어쓰는 경쟁 상태였음
  - fatalFailure(atomic.Bool) + setFatalMsg() 추가 — 한 번 걸리면 이후 문구로 안 덮임
  - ensureTools 도 fatalFailure 면 준비 완료로 표시하지 않음
  - 실제 포트 충돌을 재현해 툴팁 문구가 오류로 고정되는 것 확인
- 검증
  - stdout 리다이렉트 실행으로 콘솔 출력 + 파일 기록 동시 동작 확인
  - 포트 충돌 재현 테스트, TestFatalMsgIsNotOverwritten 추가
  - go vet 통과, 테스트 11종 통과, exe 재빌드 (7.26MB)

Stage Summary:
- 터미널에서 실행하면 로그가 바로 보임. 더블클릭은 예전대로 조용히 트레이만
- 포트 충돌 시 트레이 툴팁에 원인이 표시됨 (예전엔 정상처럼 보였음)
- 관찰: gyan.dev 의 ffmpeg 다운로드가 매우 느림 (2~3% 에서 수 분 정체).
  BtbN/FFmpeg-Builds 미러 대체를 검토할 만함

---
Task ID: 8-auto-library
Agent: Main Agent
Task: 바탕화면 음악 자동 정리 (어무이가 폴더 관리를 못 하심)

전제 조건 (사용자 확인):
- 파일명은 제목만 (날짜 접두사 없음)
- 바탕화면 자체를 음악 창고로 사용, 월별 폴더 구성
- 이미 MP3 가 수백 개 쌓여 있음
- 화면 배율 150% — 바탕화면에 한눈에 보이는 아이콘이 10~20개뿐

Work Log:
- 설계 판단: "최근 30일" 만으로는 부족
  - 많이 받은 달에는 30일치가 50곡이 넘어 화면이 넘침 = 지금과 동일한 문제
  - 날짜 조건에 **개수 상한**을 함께 걸어야 보이는 양에 천장이 생김
  - 최종 규칙: 최근 30일 이내 AND 최신 12곡까지만 바탕화면
- 월별 폴더를 바탕화면에 직접 두지 않고 `음악 보관함\2026년 07월\` 로 한 겹 안에
  - 바탕화면에 직접 두면 1년에 12개씩 폴더가 쌓여 문제가 그대로 재현됨
  - 월은 0 으로 채움 (2026년 07월) — 이름순 정렬이 곧 시간순
- library.go 신규
  - music-index.json: 받은 곡 목록 (VideoID/제목/경로/받은 시각)
  - settings.json: desktopKeepCount / desktopKeepDays. 없으면 기본값으로 생성
  - organize(): 규칙에 따라 바탕화면 <-> 보관함 배치. 시작 시 + 매 다운로드 후 실행
  - **목록에 있는 파일만 이동** — 어무이가 직접 둔 사진/문서/MP3 는 건드리지 않음
  - find(): 영상 ID 로 중복 확인. 파일이 지워졌으면 목록에서 빼고 다시 받게 함
  - adoptLooseDesktopFiles(): 기존 MP3 를 관리 대상에 등록 (파일 날짜 = 받은 시각)
- 중복 방지: 같은 영상 재요청 시 다시 받지 않고 duplicate:true 응답
  - content.js 가 "이미 받으신 노래예요" 로 다르게 표시
- 트레이 메뉴 추가: "음악 폴더 열기", "바탕화면 음악 정리"
  - 정리는 파일을 여러 개 옮기는 작업이라 MessageBox 확인 후 실행
  - dialog_windows.go / dialog_other.go (user32 MessageBoxW, 빌드 태그 분리)
- 일괄 정리 설계 수정: 처음엔 기존 MP3 를 전부 보관함으로 옮겼는데,
  그러면 정리 직후 바탕화면이 텅 비어 평소 규칙과 다르게 동작함.
  등록만 하고 배치는 organize 에 맡기도록 바꿔 최근 12곡은 바탕화면에 남게 함
- 검증
  - library_test.go — 개수 상한, 날짜 조건, 목록 외 파일 불간섭, 중복 판정,
    파일 삭제 후 재다운로드, 폴더명 정렬, 반복 실행 멱등성 (10종)
  - 실제 규모 재현: 400일에 걸친 MP3 300개 + 사진/PDF/메모
    → 바탕화면 12곡 + 폴더 1개, 보관함 월별 14폴더에 288곡, 비 MP3 3개 그대로
  - go vet 통과, 전체 테스트 통과, exe 빌드 7.39MB

Stage Summary:
- 바탕화면 아이콘 수에 상한이 생김 — 몇 년을 써도 "최근 12곡 + 폴더 1개"
- 기존 수백 개는 트레이 "바탕화면 음악 정리" 한 번으로 월별 정리
- 같은 노래 두 번 눌러도 (1), (2) 로 안 쌓임
- 기준은 settings.json 으로 조정 가능 (숫자를 늘리면 보관함에서 다시 올라옴)

---
Task ID: 9-installer-autostart
Agent: Main Agent
Task: 설치 프로그램 제작 + 재부팅 후 자동 실행

설계 판단 (사용자가 제시한 3안 검토):
- **Windows 서비스 — 불가.** Vista 이후 서비스는 세션 0 에 격리되어
  트레이 아이콘 표시도, 사용자 화면에서 음악 앱 실행도 불가능. 이 앱 구조와 맞지 않음
- **작업 스케줄러 — 가능하나 불필요.** 관리자 권한·지연 실행·실패 시 재시작 같은
  장점이 이 앱엔 해당 없고, schtasks 의존과 제거할 작업 항목만 늘어남
- **HKCU Run 키 — 채택.** 관리자 권한 불필요, 사용자 세션에서 실행되어 트레이 정상 동작

설치 위치 판단:
- `%LOCALAPPDATA%\Programs\어무이음악\` (Program Files 아님)
- 이유: BaseDir 에 yt-dlp/ffmpeg 약 170MB 를 내려받고 로그·index·settings 를 쓴다.
  Program Files 는 쓰기 불가라 첫 실행이 실패하는데, -H windowsgui 라 원인도 안 보임
- UAC 창도 뜨지 않음 (VS Code·Slack 과 같은 방식)

Work Log:
- autostart_windows.go / autostart_other.go — HKCU Run 키 등록/해제/상태 확인
  - 등록 값은 따옴표로 감싼 exe 절대경로 (경로에 공백이 있어도 깨지지 않도록)
  - 등록된 경로가 현재 exe 와 다르면(프로그램을 옮긴 경우) 꺼진 것으로 판단
  - golang.org/x/sys 를 direct 로 승격. go 지시자는 1.23 유지 확인
- instance_windows.go / instance_other.go — 이름 있는 뮤텍스로 중복 실행 방지
  - 두 벌이 뜨면 두 번째는 포트를 못 잡아 서버 없이 트레이만 뜨는 상태가 됨
    (Task 7 에서 실제로 관측된 현상)
  - 설치·업그레이드 시 exe 가 잠겨 절반만 덮어써지는 것도 함께 방지
  - 뮤텍스 이름은 Inno Setup 의 AppMutex 와 동일 — 설치 프로그램이 종료를 안내
- 트레이 메뉴에 "윈도우 시작할 때 자동 실행" 체크박스 추가
- installer/eomui-music.iss — Inno Setup 스크립트
  - PrivilegesRequired=lowest, DefaultDirName={localappdata}\Programs\어무이음악
  - AppMutex 로 실행 중 설치 차단
  - Tasks: 자동 실행(Run 키, uninstdeletevalue), 바탕화면 아이콘
  - 크롬 확장을 {app}\크롬확장 에 함께 설치 + 시작 메뉴 바로가기
  - UninstallDelete 로 내려받은 도구·로그·tmp·index 정리 (MP3 는 바탕화면이라 무관)
  - Korean.isl 은 Inno 기본 배포에 없어 #if FileExists 로 분기 (없으면 영어)
- installer/build.ps1 — vet → test → go build → ISCC 순서. ISCC 없으면 설치법 안내
- autostart_windows_test.go — 실제 HKCU Run 키를 쓰되 테스트 전후 상태를 복원
  - 등록/해제, 중복 등록·중복 해제, 따옴표 형식, 옛 경로 감지 (2종)
  - 실행 후 레지스트리가 원래 상태로 복원됨을 별도 확인

Stage Summary:
- Go 쪽은 완료 — 자동 실행 등록/해제, 중복 실행 방지, 트레이 토글 동작 확인
- .iss 스크립트와 빌드 스크립트 작성 완료, exe 빌드까지는 정상 (7.40MB)
- **setup.exe 는 아직 못 만듦** — 이 컴퓨터에 Inno Setup(ISCC.exe)이 없음.
  winget install JRSoftware.InnoSetup 필요 (사용자 동의 대기)
- 미해결: 서명하지 않은 setup.exe 는 Windows 11 SmartScreen 경고가 뜬다.
  어무이가 직접 실행하시면 바이러스 경고처럼 보이므로 아들이 대신 설치하는 편이 좋음
