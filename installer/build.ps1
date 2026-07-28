# 어무이 음악 다운로더 - 설치 프로그램 빌드
#
#   powershell -ExecutionPolicy Bypass -File installer\build.ps1
#
# 1) Go exe 를 빌드하고
# 2) Inno Setup 으로 installer\output\eomui-music-setup.exe 를 만든다.

$ErrorActionPreference = 'Stop'

# 콘솔 코드페이지가 949 면 한글 안내가 깨져서 나온다.
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$root = Split-Path -Parent $PSScriptRoot
$goDir = Join-Path $root 'app'
$iss = Join-Path $PSScriptRoot 'eomui-music.iss'
$bundle = Join-Path $PSScriptRoot 'bundle'

# ── 번들 파일 준비 ────────────────────────────────────────────────
# 어무이 PC 에서 첫 실행 때 170MB 를 받게 하지 않는다.
# gyan.dev 의 ffmpeg 다운로드는 실제로 2~3% 에서 수 분씩 정체하는 것이 관측되었다.
# 이미 있으면 다시 받지 않는다.
Write-Host '[1/3] 번들 파일 준비...' -ForegroundColor Cyan
New-Item -ItemType Directory -Force -Path $bundle | Out-Null
$ProgressPreference = 'SilentlyContinue'

function Get-BundleFile($name, $url, $entryName) {
    $dest = Join-Path $bundle $name
    if ((Test-Path $dest) -and (Get-Item $dest).Length -gt 0) {
        Write-Host ("      있음  {0} ({1:N1} MB)" -f $name, ((Get-Item $dest).Length / 1MB))
        return
    }
    Write-Host "      받는 중  $name ..."
    if (-not $entryName) {
        Invoke-WebRequest -Uri $url -OutFile $dest
    }
    else {
        # ZIP 안에서 실행 파일 하나만 꺼낸다.
        $zip = Join-Path $bundle 'tmp-download.zip'
        Invoke-WebRequest -Uri $url -OutFile $zip
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        $archive = [System.IO.Compression.ZipFile]::OpenRead($zip)
        try {
            $entry = $archive.Entries | Where-Object { $_.Name -eq $entryName } | Select-Object -First 1
            if (-not $entry) { throw "$entryName 를 ZIP 에서 찾지 못했습니다" }
            [System.IO.Compression.ZipFileExtensions]::ExtractToFile($entry, $dest, $true)

            # ffmpeg 는 GPL 이라 재배포 시 라이선스를 함께 둔다.
            $lic = $archive.Entries | Where-Object { $_.Name -eq 'LICENSE' } | Select-Object -First 1
            if ($lic) {
                [System.IO.Compression.ZipFileExtensions]::ExtractToFile(
                    $lic, (Join-Path $bundle 'ffmpeg-LICENSE.txt'), $true)
            }
        }
        finally { $archive.Dispose(); Remove-Item $zip -Force -ErrorAction SilentlyContinue }
    }
    Write-Host ("      완료  {0} ({1:N1} MB)" -f $name, ((Get-Item $dest).Length / 1MB)) -ForegroundColor Green
}

Get-BundleFile 'yt-dlp.exe' 'https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe'
Get-BundleFile 'ffmpeg.exe' 'https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip' 'ffmpeg.exe'
Get-BundleFile 'deno.exe'   'https://github.com/denoland/deno/releases/latest/download/deno-x86_64-pc-windows-msvc.zip' 'deno.exe'

Write-Host '[2/3] Go exe 빌드...' -ForegroundColor Cyan
Push-Location $goDir
try {
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'

    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'go vet 실패' }

    go test ./...
    if ($LASTEXITCODE -ne 0) { throw '테스트 실패' }

    go build -ldflags="-s -w -H windowsgui" -o eomui-music.exe .
    if ($LASTEXITCODE -ne 0) { throw 'go build 실패' }

    $exe = Get-Item 'eomui-music.exe'
    Write-Host ("      OK  {0:N2} MB" -f ($exe.Length / 1MB)) -ForegroundColor Green
}
finally { Pop-Location }

Write-Host '[3/3] 설치 프로그램 컴파일...' -ForegroundColor Cyan

# ISCC.exe 찾기
$iscc = $null
foreach ($c in @(
    # winget 은 사용자 범위로 설치해서 여기에 들어간다.
    "$env:LOCALAPPDATA\Programs\Inno Setup 6\ISCC.exe",
    "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe",
    "$env:ProgramFiles\Inno Setup 6\ISCC.exe",
    "${env:ProgramFiles(x86)}\Inno Setup 5\ISCC.exe"
)) { if ($c -and (Test-Path $c)) { $iscc = $c; break } }

if (-not $iscc) {
    $cmd = Get-Command iscc -ErrorAction SilentlyContinue
    if ($cmd) { $iscc = $cmd.Source }
}

if (-not $iscc) {
    Write-Host ''
    Write-Host 'Inno Setup 이 설치되어 있지 않습니다.' -ForegroundColor Yellow
    Write-Host '설치 방법:' -ForegroundColor Yellow
    Write-Host '  winget install JRSoftware.InnoSetup'
    Write-Host '  (또는 https://jrsoftware.org/isdl.php)'
    Write-Host ''
    Write-Host 'Go exe 는 정상 빌드되었습니다:' -ForegroundColor Green
    Write-Host "  $goDir\eomui-music.exe"
    exit 1
}

& $iscc $iss
if ($LASTEXITCODE -ne 0) { throw '설치 프로그램 컴파일 실패' }

$setup = Join-Path $PSScriptRoot 'output\eomui-music-setup.exe'
if (Test-Path $setup) {
    $f = Get-Item $setup
    Write-Host ''
    Write-Host ("완료: {0}  ({1:N2} MB)" -f $f.FullName, ($f.Length / 1MB)) -ForegroundColor Green
}
