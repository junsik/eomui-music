# 외부 구성 요소

설치 프로그램에 함께 담아 배포하는 프로그램들입니다.
각각 자체 라이선스를 따르며, 이 프로젝트의 라이선스와 무관합니다.

## 함께 배포하는 실행 파일

| 프로그램 | 용도 | 라이선스 | 출처 |
|---|---|---|---|
| yt-dlp | YouTube 음원 추출 | Unlicense (퍼블릭 도메인) | [github.com/yt-dlp/yt-dlp](https://github.com/yt-dlp/yt-dlp) |
| FFmpeg (essentials build) | MP3 변환 | **GPL v3** | [gyan.dev/ffmpeg/builds](https://www.gyan.dev/ffmpeg/builds/) |
| Deno | yt-dlp가 쓰는 JavaScript 런타임 | MIT | [github.com/denoland/deno](https://github.com/denoland/deno) |

## FFmpeg 재배포 시 유의사항

gyan.dev의 **essentials 빌드는 GPL v3**입니다. x264 같은 GPL 구성 요소가
포함되어 있기 때문입니다.

GPL 바이너리를 재배포할 때는 라이선스 전문을 함께 제공해야 하며,
요청 시 소스를 제공할 의무가 따릅니다.

- 설치 프로그램은 `ffmpeg-LICENSE.txt`를 설치 폴더에 함께 넣습니다
  (`installer/build.ps1`이 배포 ZIP에서 꺼내 옵니다)
- FFmpeg 소스는 [ffmpeg.org](https://ffmpeg.org/download.html)에서 받을 수 있습니다
- 빌드 구성은 [gyan.dev](https://www.gyan.dev/ffmpeg/builds/) 문서를 따릅니다

가족·개인 사용 범위에서는 실질적인 문제가 없지만,
**이 프로그램을 외부에 배포한다면** 위 의무를 지켜야 합니다.
LGPL만 필요하다면 GPL 구성 요소가 빠진 빌드로 바꾸는 방법도 있습니다.

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
