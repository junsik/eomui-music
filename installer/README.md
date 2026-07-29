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
| `build.ps1` | 번들 준비 → 앱 빌드 → 컴파일 (3단계) |
| `extension-guide.html` | 설치 후 여는 크롬 확장 등록 안내 |
| `ffmpeg-source-offer.txt` | FFmpeg GPL v3 소스 제공 고지 |
| `bundle/` | 함께 담을 외부 실행 파일 (git 제외, 자동 다운로드) |
| `output/` | 결과물 (git 제외) |

저장소 루트의 `LICENSE` 와 `NOTICE` 도 설치 폴더에 함께 배치됩니다.

`build.ps1`은 **UTF-8 BOM**으로 저장해야 합니다.
Windows PowerShell 5.1은 BOM이 없으면 ANSI로 읽어 한글이 깨지고 파싱이 실패합니다.

## 설치 프로그램이 하는 일

- `%LOCALAPPDATA%\Programs\어무이음악\`에 설치 (관리자 권한 불필요)
- `yt-dlp` · `ffmpeg` · `deno`를 함께 배치 — 첫 실행부터 바로 동작
- Chrome 확장을 `크롬확장\` 폴더에 배치 + 시작 메뉴 바로가기
- 라이선스·고지 파일 배치 (`LICENSE.txt` · `NOTICE.txt` ·
  `ffmpeg-LICENSE.txt` · `ffmpeg-source-offer.txt`)
- 선택 항목: 로그인 시 자동 실행, 바탕화면 아이콘
- 마지막 화면에서 확장 등록 안내와 확장 폴더를 함께 열어 줌
- 제어판 제거 항목 등록

## GPL 고지는 빼면 안 됩니다

번들하는 FFmpeg는 GPL v3(`--enable-gpl --enable-version3`)입니다.
공개 배포하는 이상 라이선스 전문과 대응 소스를 받는 방법을 함께 제공해야 합니다.

`ffmpeg-source-offer.txt`에는 배포 중인 **정확한 빌드 버전**과 상위 소스 태그가
적혀 있습니다. **번들 ffmpeg를 갱신하면 이 파일도 함께 고쳐야 합니다.**
버전이 안 맞으면 고지가 가리키는 소스가 실제 배포본과 달라집니다.

```powershell
installer\bundle\ffmpeg.exe -version   # 버전과 configuration 확인
```

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
