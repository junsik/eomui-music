# installer — 설치 프로그램

[Inno Setup](https://jrsoftware.org/isinfo.php)으로 `eomui-music-setup.exe`를 만듭니다.

```powershell
powershell -ExecutionPolicy Bypass -File installer\build.ps1
# → installer\output\eomui-music-setup.exe  (약 74MB)
```

Inno Setup이 없으면 빌드 스크립트가 설치 방법을 안내하고 멈춥니다.

```powershell
winget install JRSoftware.InnoSetup
```

## 파일

| 파일 | 역할 |
|---|---|
| `eomui-music.iss` | Inno Setup 스크립트 |
| `build.ps1` | 번들 준비 → 앱 빌드 → 컴파일 |
| `bundle/` | 함께 담을 외부 실행 파일 (git 제외, 자동 다운로드) |
| `output/` | 결과물 (git 제외) |

`build.ps1`은 **UTF-8 BOM**으로 저장해야 합니다.
Windows PowerShell 5.1은 BOM이 없으면 ANSI로 읽어 한글이 깨지고 파싱이 실패합니다.

## 설치 프로그램이 하는 일

- `%LOCALAPPDATA%\Programs\어무이음악\`에 설치 (관리자 권한 불필요)
- `yt-dlp` · `ffmpeg` · `deno`를 함께 배치 — 첫 실행부터 바로 동작
- Chrome 확장을 `크롬확장\` 폴더에 배치 + 시작 메뉴 바로가기
- 선택 항목: 로그인 시 자동 실행, 바탕화면 아이콘
- 제어판 제거 항목 등록

## 알아 둘 것

**`AppMutex`** 는 앱의 `instance_windows.go`에 있는 이름과 같아야 합니다.
앱이 실행 중일 때 설치를 시도하면 종료를 안내합니다.

**`onlyifdoesntexist`** 플래그로 외부 실행 파일을 설치합니다.
앱이 `yt-dlp`를 스스로 최신으로 갱신하므로, 재설치할 때
낡은 번들 버전으로 되돌리면 안 되기 때문입니다.

**`[UninstallDelete]`** 에 앱이 실행 중 만드는 파일을 모두 넣어야 합니다.
빠뜨리면 제거 후 폴더가 남습니다. 새 상태 파일을 추가할 때 함께 갱신하세요.

**Korean.isl** 은 Inno Setup 6.7부터 기본 포함입니다.
없는 버전이면 영어 화면으로 컴파일되지만, 화면에 나오는 문구는
언어 접두사 없는 `[CustomMessages]`라 한국어로 나옵니다.
