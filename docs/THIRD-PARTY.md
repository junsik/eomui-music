# 외부 구성 요소

설치 프로그램에 함께 담아 배포하는 프로그램들입니다.
각각 자체 라이선스를 따르며, 이 프로젝트의 MIT 라이선스와 무관합니다.

| 프로그램 | 용도 | 라이선스 | 출처 |
|---|---|---|---|
| yt-dlp | YouTube 음원 추출 | Unlicense (퍼블릭 도메인) | [github.com/yt-dlp/yt-dlp](https://github.com/yt-dlp/yt-dlp) |
| FFmpeg (gyan.dev essentials build) | MP3 변환 | **GPL v3** | [gyan.dev/ffmpeg/builds](https://www.gyan.dev/ffmpeg/builds/) |
| Deno | yt-dlp가 쓰는 JavaScript 런타임 | MIT | [github.com/denoland/deno](https://github.com/denoland/deno) |

## 이 프로그램이 GPL이 아닌 이유

이 프로젝트의 코드는 MIT입니다. FFmpeg를 함께 배포하지만 **링크하지 않습니다.**

앱은 `yt-dlp.exe`를 별도 프로세스로 실행하고, yt-dlp가 다시 `ffmpeg.exe`를
별도 프로세스로 실행합니다. 프로세스 경계를 두고 명령줄로만 주고받는 구조라
GPL이 말하는 파생 저작물이 아니라 단순 병합(mere aggregation)에 해당합니다.

다만 **GPL 바이너리를 재배포하는 의무는 그대로 남습니다.** 아래가 그 이행입니다.

---

## FFmpeg — GPL v3 소스 제공 고지

### 배포하는 바이너리

```
ffmpeg version 8.1.2-essentials_build-www.gyan.dev
configuration: --enable-gpl --enable-version3 --enable-static ...
```

`--enable-gpl --enable-version3` 로 빌드되어 **GPL 버전 3**이 적용됩니다.
전체 configuration 문자열은 `ffmpeg.exe -version` 으로 확인할 수 있습니다.

라이선스 전문은 설치 폴더의 `ffmpeg-LICENSE.txt` 에 함께 설치됩니다.

### 대응 소스를 받는 방법

**1. 상위 소스 (FFmpeg 프로젝트)**

- 릴리스 소스: <https://ffmpeg.org/download.html>
- 소스 저장소: <https://git.ffmpeg.org/ffmpeg.git>
- 이 빌드에 해당하는 태그: `n8.1.2`

**2. 빌드 구성과 스크립트 (gyan.dev)**

이 바이너리는 gyan.dev의 essentials 빌드를 그대로 담은 것입니다.
빌드에 쓰인 구성과 스크립트는 배포자가 공개하고 있습니다.

- 빌드 문서: <https://www.gyan.dev/ffmpeg/builds/>
- 빌드 스크립트: <https://github.com/GyanD/codexffmpeg>

**3. 서면 제공 고지 (GPL v3 §6(b))**

위 경로가 막히거나 사라진 경우를 대비해, 이 릴리스를 받은 누구에게든
**이 소프트웨어를 배포한 날로부터 최소 3년간** 해당 바이너리의 대응 소스
전체를 매체 비용 이하의 실비로 제공합니다.

요청은 이 저장소의 이슈로 남겨 주십시오.

- <https://github.com/junsik/eomui-music/issues>

요청하실 때 받으신 설치 파일의 **버전(릴리스 태그)** 을 알려 주시면
해당 버전에 담긴 정확한 소스를 안내해 드립니다.

### 다르게 하고 싶다면

GPL 의무가 부담스러우면 두 가지 방법이 있습니다.

- **번들에서 빼기** — 앱에 자동 다운로드 기능이 이미 있어서, 설치 파일에
  ffmpeg를 넣지 않으면 첫 실행 때 사용자 컴퓨터가 직접 내려받습니다.
  재배포가 아니므로 의무가 사라집니다. 대신 첫 실행이 느려집니다
  (gyan.dev 다운로드가 매우 느린 것이 관측되었습니다).
- **LGPL 빌드로 바꾸기** — GPL 구성 요소(x264 등)가 빠진 빌드를 쓰면
  조건이 가벼워집니다. 다만 소스 제공 의무 자체는 남습니다.

---

## Go 모듈 의존성

| 모듈 | 라이선스 |
|---|---|
| `github.com/getlantern/systray` | Apache-2.0 |
| `github.com/google/uuid` | BSD-3-Clause |
| `golang.org/x/text` | BSD-3-Clause |
| `golang.org/x/sys` | BSD-3-Clause |

전체 목록과 정확한 버전은 [app/go.mod](../app/go.mod)와
[app/go.sum](../app/go.sum)에 있습니다.

## 저작권에 대해

이 프로그램은 사용자가 **직접 선택한 영상**을 개인 감상용으로 내려받는 도구입니다.
내려받는 콘텐츠의 저작권과 각 서비스 이용약관 준수 책임은 사용자에게 있습니다.
