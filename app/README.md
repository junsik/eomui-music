# app — Go 데스크톱 앱

창이 없는 Windows GUI 프로그램입니다. 트레이 아이콘만 띄우고
로컬 HTTP 서버(`127.0.0.1:8080`)를 돌리면서 Chrome 확장의 요청을 받습니다.

```powershell
$env:GOOS='windows'; $env:GOARCH='amd64'; $env:CGO_ENABLED='0'
go build -ldflags="-s -w -H windowsgui" -o eomui-music.exe .
go test ./...
```

## 파일 구성

| 파일 | 역할 |
|---|---|
| `main.go` | HTTP 핸들러, 트레이 메뉴, 다운로드 흐름 |
| `bootstrap.go` | yt-dlp/ffmpeg/deno 확인·다운로드, 준비 상태 |
| `library.go` | 받은 곡 목록, 바탕화면 정리, 중복 판정 |
| `updater.go` | yt-dlp 주기적 자동 갱신 |
| `console_*.go` | 콘솔 출력 확보 |
| `dialog_*.go` | 확인·안내 대화상자 |
| `autostart_*.go` | 로그인 시 자동 실행 등록 |
| `instance_*.go` | 중복 실행 방지 |

`_windows.go` / `_other.go` 쌍은 빌드 태그로 나뉩니다.

## 실행 중 생기는 파일

`settings.json` · `music-index.json` · `state.json` · `eomui-music.log` · `tmp/`
— 모두 git에 올리지 않습니다.

## 문서

- 설계 배경과 보안 모델: [docs/ARCHITECTURE.md](../docs/ARCHITECTURE.md)
- 빌드·테스트·릴리스: [docs/DEVELOPMENT.md](../docs/DEVELOPMENT.md)
- 설치와 사용법: [docs/INSTALL.md](../docs/INSTALL.md)
