# 개발

## 필요한 것

| 도구 | 용도 | 비고 |
|---|---|---|
| Go 1.23+ | 앱 빌드 | cgo 불필요 (`CGO_ENABLED=0`) |
| Inno Setup 6 | 설치 프로그램 | `winget install JRSoftware.InnoSetup` |
| Python + Pillow | 아이콘 생성 | [scripts/](../scripts/) 쓸 때만 |

Inno Setup은 winget으로 설치하면 `%LOCALAPPDATA%\Programs\Inno Setup 6\`에 들어갑니다.
`build.ps1`이 그 경로도 찾습니다.

## 앱만 빌드

```powershell
cd app
$env:GOOS='windows'; $env:GOARCH='amd64'; $env:CGO_ENABLED='0'
go build -ldflags="-s -w -H windowsgui" -o eomui-music.exe .
```

`-H windowsgui`가 콘솔 창을 없앱니다. 확인:

```powershell
$b = [System.IO.File]::ReadAllBytes('eomui-music.exe')
$pe = [BitConverter]::ToInt32($b, 0x3C)
[BitConverter]::ToUInt16($b, $pe + 0x5C)   # 2 = Windows GUI
```

## 테스트

```powershell
cd app
go vet ./...
go test ./...
```

테스트는 외부 네트워크 없이 돌고, 기본 실행은 **시스템 상태를 바꾸지 않습니다.**

- 정리 로직 테스트는 임시 폴더를 바탕화면으로 삼습니다. 실제 바탕화면은 건드리지 않습니다.
- `autostart_windows_test.go`는 실제 HKCU Run 키를 건드리므로 **기본으로 건너뜁니다.**
  `installer/build.ps1`이 매번 `go test`를 돌리는데, 중간에 중단되면
  레지스트리에 값이 남기 때문입니다.

자동 실행 등록까지 확인하려면 명시적으로 켜세요.

```powershell
$env:EOMUI_TEST_REGISTRY='1'
go test -run TestAutostart -v .
Remove-Item Env:\EOMUI_TEST_REGISTRY
```

실행 후 값이 남지 않았는지 확인:

```powershell
Get-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run' -Name EomuiMusic
```

## 설치 프로그램 빌드

```powershell
powershell -ExecutionPolicy Bypass -File installer\build.ps1
```

세 단계로 돕니다.

1. **번들 준비** — `installer/bundle/`에 `yt-dlp.exe` · `ffmpeg.exe` · `deno.exe`가
   없으면 내려받습니다 (약 207MB). 이미 있으면 건너뜁니다.
2. **앱 빌드** — `go vet` → `go test` → `go build` 순서. 하나라도 실패하면 중단합니다.
3. **컴파일** — `installer/output/eomui-music-setup.exe` (약 74MB)

`bundle/`과 `output/`은 git에 올라가지 않습니다.

### 번들을 갱신하려면

`installer/bundle/`의 해당 파일을 지우고 다시 빌드하면 최신을 받습니다.
`ffmpeg`·`deno`는 거의 바꿀 일이 없고, `yt-dlp`는 앱이 스스로 갱신하므로
번들이 조금 낡아도 문제되지 않습니다.

설치 시 `onlyifdoesntexist` 플래그를 쓰므로 **재설치해도 이미 최신으로
갱신된 yt-dlp를 낡은 번들 버전으로 되돌리지 않습니다.**

## 무인 설치 테스트

실제 설치·제거를 확인할 때 씁니다. 임시 폴더에 설치하고 옵션은 모두 끕니다.

```powershell
$dest = "$env:TEMP\eomui-test"
Start-Process installer\output\eomui-music-setup.exe -Wait -ArgumentList `
  '/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART','/TASKS=""',"/DIR=$dest"

# 확인 후 제거
Start-Process "$dest\unins000.exe" -Wait -ArgumentList '/VERYSILENT','/SUPPRESSMSGBOXES'
```

제거 후 폴더가 완전히 비는지 확인하세요. 앱이 실행 중에 만드는 파일
(`state.json` 등)은 `[UninstallDelete]`에 넣어야 남지 않습니다.

## 로그 보기

GUI 서브시스템이지만 콘솔 출력이 됩니다.

```powershell
.\eomui-music.exe                 # 터미널에서 실행 → 그 콘솔에 출력
.\eomui-music.exe > out.txt 2>&1  # 리다이렉트
.\eomui-music.exe -console        # 콘솔 창을 새로 띄움
```

콘솔이 없으면 `os.Stdout` 쓰기가 실패합니다. `io.MultiWriter`는 첫 실패에서
멈추므로, 콘솔을 확보한 경우에만 로그 대상에 넣습니다.
그렇지 않으면 **파일 기록까지 같이 죽습니다.**

## 코드 규칙

- 들여쓰기는 **공백 8칸**입니다. Go 표준(탭)이 아니지만 기존 파일을 따릅니다.
  `gofmt -w .`를 돌리면 전부 탭으로 바뀌니 주의하세요.
- 주석은 한국어로 씁니다. **무엇을** 하는지보다 **왜** 그렇게 했는지를 적습니다.
- Windows 전용 코드는 `_windows.go` / `_other.go`로 나눕니다.

## 의존성

| 모듈 | 용도 |
|---|---|
| `github.com/getlantern/systray` | 트레이 아이콘 (Windows는 순수 syscall, cgo 불필요) |
| `github.com/google/uuid` | 임시 파일 이름 |
| `golang.org/x/text` | CP949 디코딩 (yt-dlp 출력) |
| `golang.org/x/sys` | 레지스트리 접근 |

`golang.org/x/text`는 **v0.21.0으로 고정**했습니다.
최신 버전은 `go.mod`의 `go` 지시자를 1.25로 올려 버립니다.

## 릴리스

1. `app/`에서 `go test ./...` 통과 확인
2. 버전 올리기 — `installer/eomui-music.iss`의 `AppVersion`,
   `extension/manifest.json`의 `version`
3. `installer\build.ps1` 실행
4. 무인 설치 테스트로 설치·제거 확인
5. [CHANGELOG.md](../CHANGELOG.md) 갱신

서버가 확장의 `X-Eomui-Client` 헤더를 확인하므로,
**확장과 앱은 함께 갱신해야 합니다.** 앱만 새로 깔고 예전 확장을 두면
`Origin` 조건으로 통과하긴 하지만, 확장도 다시 불러오는 편이 안전합니다.
