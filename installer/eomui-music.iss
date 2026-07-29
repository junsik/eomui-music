; 어무이 음악 다운로더 - Inno Setup 설치 스크립트
;
; 빌드: installer\build.ps1 실행 (exe 빌드 → ISCC 컴파일)
; 결과: installer\output\eomui-music-setup.exe

#define AppName "어무이 음악 다운로더"
#define AppVersion "1.4.1"
#define AppPublisher "개인 제작"
#define ExeName "eomui-music.exe"
#define SrcDir "..\app"
#define ExtDir "..\extension"

[Setup]
AppId={{9E3F2A41-7C5D-4B18-A6E2-3D9C41B7F0A5}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
VersionInfoVersion={#AppVersion}

; 관리자 권한을 요구하지 않는다.
; 어무이 컴퓨터에서 UAC 창이 뜨지 않고, 무엇보다
; 프로그램 폴더에 yt-dlp/ffmpeg(약 170MB)를 내려받아야 하므로
; Program Files(쓰기 불가)에 설치하면 첫 실행이 실패한다.
PrivilegesRequired=lowest
DefaultDirName={localappdata}\Programs\어무이음악
DisableDirPage=auto
UsePreviousAppDir=yes

DefaultGroupName={#AppName}
DisableProgramGroupPage=yes

; 실행 중이면 설치를 막고 종료를 안내한다.
; 이 이름은 instance_windows.go 의 appMutexName 과 같아야 한다.
AppMutex=EomuiMusicSingleInstance

OutputDir=output
OutputBaseFilename=eomui-music-setup
SetupIconFile={#SrcDir}\icon.ico
UninstallDisplayIcon={app}\{#ExeName}
UninstallDisplayName={#AppName}

Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible

[Languages]
; Inno Setup 6.7 에는 Korean.isl 이 포함되어 있다.
; 없는 버전이면 영어 화면으로 컴파일된다 (CompilerPath 는 끝에 역슬래시를 포함).
#if FileExists(CompilerPath + "Languages\Korean.isl")
Name: "korean"; MessagesFile: "compiler:Languages\Korean.isl"
#else
Name: "english"; MessagesFile: "compiler:Default.isl"
#endif

[CustomMessages]
; 언어 접두사를 붙이지 않으면 선택된 언어와 무관하게 적용된다.
; 이 프로그램은 한국어 전용이라 마법사 언어가 무엇이든 본문은 한국어로 둔다.
AutoStartTask=윈도우 시작할 때 자동으로 실행 (권장)
DesktopIconTask=바탕화면에 "어무이 음악" 아이콘 만들기 (권장)
OpenExtFolder=크롬 확장 프로그램 폴더 열기
ExtGuide=크롬 확장 설치 안내
ShowGuide=크롬 확장 등록 방법 보기 (필수 — 안 하면 버튼이 안 나옵니다)
OpenExtFolderNow=크롬 확장 폴더를 탐색기로 열기

[Tasks]
Name: "autostart"; Description: "{cm:AutoStartTask}"
Name: "desktopicon"; Description: "{cm:DesktopIconTask}"

[Files]
Source: "{#SrcDir}\{#ExeName}"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SrcDir}\README.md";  DestDir: "{app}"; Flags: ignoreversion

; 크롬 확장 등록 안내. 크롬 정책상 이 단계는 자동화할 수 없어서,
; 안 알려 주면 설치는 됐는데 YouTube 에 버튼이 안 나타나는 상태가 된다.
; 자기 위치에서 확장 폴더 경로를 계산해 보여 주므로 설치 경로와 무관하게 정확하다.
Source: "extension-guide.html"; DestDir: "{app}"; DestName: "크롬확장-설치안내.html"; \
    Flags: ignoreversion

; 필수 실행 파일을 함께 넣는다. 첫 실행 때 약 170MB 를 내려받지 않아도 된다.
; gyan.dev 의 ffmpeg 다운로드가 매우 느려 어무이 PC 에서 실패할 위험이 크다.
; 이미 있으면 덮어쓰지 않는다 (onlyifdoesntexist) — yt-dlp 는 스스로 최신으로 갱신되므로
; 재설치할 때 오래된 번들 버전으로 되돌리면 안 된다.
Source: "bundle\yt-dlp.exe"; DestDir: "{app}"; Flags: onlyifdoesntexist
Source: "bundle\ffmpeg.exe"; DestDir: "{app}"; Flags: onlyifdoesntexist
Source: "bundle\deno.exe";   DestDir: "{app}"; Flags: onlyifdoesntexist
; ffmpeg essentials 빌드는 GPL v3 라 재배포 시 라이선스 전문과
; 대응 소스를 받는 방법을 함께 제공해야 한다 (GPL v3 제6조).
; 저장소를 안 보고 설치 파일만 받은 사람도 볼 수 있어야 하므로 여기에 넣는다.
Source: "bundle\ffmpeg-LICENSE.txt"; DestDir: "{app}"; Flags: ignoreversion skipifsourcedoesntexist
Source: "ffmpeg-source-offer.txt";  DestDir: "{app}"; Flags: ignoreversion
Source: "..\LICENSE";               DestDir: "{app}"; DestName: "LICENSE.txt"; Flags: ignoreversion
Source: "..\NOTICE";                DestDir: "{app}"; DestName: "NOTICE.txt";  Flags: ignoreversion

; 크롬 확장은 수동으로 불러와야 해서 같이 설치해 둔다.
Source: "{#ExtDir}\*"; DestDir: "{app}\크롬확장"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
; -open 을 붙이면 음악 목록 화면이 브라우저로 열린다.
; 꺼져 있으면 실행하면서 열고, 이미 켜져 있으면 화면만 연다.
; 자동 실행(Run 키)에는 이 옵션을 붙이지 않는다 — 로그인마다 브라우저가 뜨면 안 된다.
Name: "{group}\{#AppName}"; Filename: "{app}\{#ExeName}"; Parameters: "-open"
Name: "{group}\{cm:ExtGuide}"; Filename: "{app}\크롬확장-설치안내.html"
Name: "{group}\{cm:OpenExtFolder}"; Filename: "{app}\크롬확장"
Name: "{group}\{cm:UninstallProgram,{#AppName}}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\{#AppName}"; Filename: "{app}\{#ExeName}"; Parameters: "-open"; Tasks: desktopicon

[Registry]
; 자동 실행. HKCU 라서 관리자 권한이 필요 없고 어무이 세션에서 실행된다.
; uninsdeletevalue 로 제거 시 함께 지워진다.
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; \
    ValueType: string; ValueName: "EomuiMusic"; ValueData: """{app}\{#ExeName}"""; \
    Flags: uninsdeletevalue; Tasks: autostart

[Run]
; 마지막 화면의 체크박스. 위에서부터 순서대로 실행된다.
; 확장 등록은 크롬 정책상 자동화할 수 없으므로, 안내와 폴더를 함께 열어 준다.
Filename: "{app}\크롬확장-설치안내.html"; Description: "{cm:ShowGuide}"; \
    Flags: postinstall shellexec nowait skipifsilent
Filename: "{app}\크롬확장"; Description: "{cm:OpenExtFolderNow}"; \
    Flags: postinstall shellexec nowait skipifsilent
Filename: "{app}\{#ExeName}"; Parameters: "-open"; \
    Description: "{cm:LaunchProgram,{#AppName}}"; \
    Flags: nowait postinstall skipifsilent

[UninstallDelete]
; 첫 실행 때 내려받는 파일들과 작업 흔적. 설치 목록에 없어서 직접 지워야 한다.
; 받은 MP3 는 바탕화면에 있으므로 여기서 지워지지 않는다.
Type: files;          Name: "{app}\yt-dlp.exe"
Type: files;          Name: "{app}\ffmpeg.exe"
Type: files;          Name: "{app}\deno.exe"
Type: files;          Name: "{app}\eomui-music.log"
Type: files;          Name: "{app}\settings.json"
Type: files;          Name: "{app}\music-index.json"
Type: files;          Name: "{app}\state.json"
Type: filesandordirs; Name: "{app}\tmp"
Type: dirifempty;     Name: "{app}"
