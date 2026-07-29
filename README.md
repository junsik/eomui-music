# 어무이 음악 다운로더

YouTube 영상을 MP3로 바꿔 어머니 바탕화면에 저장하는 Windows 프로그램.

시니어 사용자를 위해 만들었습니다. **주소를 복사하거나 붙여넣을 필요가 없습니다.**
YouTube 영상 페이지에서 빨간 버튼 한 번이면 끝납니다.

```
[YouTube 영상 페이지]
        │  좋아요·공유 옆의 빨간 "MP3 다운로드" 버튼 클릭
        ▼
[Chrome 확장]  ──fetch──▶  [로컬 서버 (트레이에서 실행 중)]
                                    │  yt-dlp + ffmpeg 실행
                                    ▼
                          [바탕화면에 MP3 저장]
```

## 이 프로그램이 신경 쓰는 것

- **원클릭** — 주소 복사·붙여넣기 없음
- **바탕화면에 바로 저장** — 브라우저 다운로드 폴더가 아님
- **큰 글씨 음악 목록** — 바탕화면 아이콘을 누르면 받은 곡이 한 줄씩 뜨고,
  **듣기** 버튼으로 바로 재생. 폴더를 찾아 들어갈 필요가 없음
- **창이 안 뜸** — 트레이 아이콘으로만 조용히 상주
- **자동 정리** — 바탕화면이 MP3로 뒤덮이지 않도록 최근 곡만 남기고 월별 보관함으로 이동
- **설치 한 번으로 끝** — 필요한 실행 파일이 모두 들어 있어 첫 실행부터 동작
- **스스로 최신 유지** — yt-dlp를 주기적으로 갱신해 YouTube 변경으로 멈추지 않음

## 폴더 구성

| 폴더 | 내용 |
|---|---|
| [app/](app/) | Go 데스크톱 앱 — HTTP 서버 + 트레이 + yt-dlp 래퍼 + 음악 정리 |
| [extension/](extension/) | Chrome 확장 (Manifest V3) — YouTube 액션바에 버튼 주입 |
| [installer/](installer/) | Inno Setup 설치 프로그램 스크립트와 빌드 스크립트 |
| [scripts/](scripts/) | 아이콘 생성 등 보조 스크립트 |
| [docs/](docs/) | 설치·사용·구조·개발 문서 |

## 문서

- **[설치와 사용](docs/INSTALL.md)** — 설치 방법, 확장 등록, 문제 해결
- **[구조](docs/ARCHITECTURE.md)** — 구성 요소, 보안 모델, 음악 정리 규칙
- **[개발](docs/DEVELOPMENT.md)** — 빌드, 테스트, 릴리스
- **[외부 구성 요소](docs/THIRD-PARTY.md)** — 함께 배포하는 프로그램과 라이선스
- **[변경 이력](CHANGELOG.md)**
- **[개발 기록](docs/worklog.md)** — 작업 로그

## 빠르게 만들어 보기

```powershell
# 설치 프로그램까지 한 번에 (필요한 외부 파일은 자동으로 내려받음)
powershell -ExecutionPolicy Bypass -File installer\build.ps1
# → installer\output\eomui-music-setup.exe
```

앱만 빌드하려면:

```powershell
cd app
$env:GOOS='windows'; $env:GOARCH='amd64'; $env:CGO_ENABLED='0'
go build -ldflags="-s -w -H windowsgui" -o eomui-music.exe .
```

자세한 내용은 [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)를 참고하세요.

## 라이선스

이 저장소의 코드는 **MIT** 라이선스입니다 — [LICENSE](LICENSE).

설치 프로그램에는 **FFmpeg(GPL v3)** · yt-dlp(Unlicense) · Deno(MIT)가
함께 담겨 있고, 이들은 각자의 라이선스를 따릅니다.
FFmpeg의 대응 소스를 받는 방법을 포함한 자세한 내용은
[docs/THIRD-PARTY.md](docs/THIRD-PARTY.md)에 있습니다.

이 프로그램은 FFmpeg와 링크하지 않고 별도 프로세스로 실행하므로
코드 자체는 GPL의 적용을 받지 않습니다. 다만 GPL 바이너리를 재배포하는 의무는
별개로 이행합니다.
