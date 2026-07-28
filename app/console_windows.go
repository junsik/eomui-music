//go:build windows

package main

import (
        "os"
        "os/exec"
        "syscall"
)

// CREATE_NO_WINDOW — 콘솔 프로그램을 창 없이 실행한다.
const createNoWindow = 0x08000000

// hideConsole은 자식 프로세스의 콘솔 창이 뜨지 않게 한다.
//
// 이 프로그램은 GUI 서브시스템이라 자기 콘솔이 없다. 그래서 yt-dlp 나 ffmpeg
// 같은 콘솔 프로그램을 실행하면 Windows 가 새 콘솔 창을 만들어 준다.
// 다운로드할 때마다 검은 창이 깜빡이는 원인이다.
//
// yt-dlp 가 안에서 부르는 ffmpeg 도 이 설정을 물려받는다.
func hideConsole(cmd *exec.Cmd) {
        if cmd.SysProcAttr == nil {
                cmd.SysProcAttr = &syscall.SysProcAttr{}
        }
        cmd.SysProcAttr.HideWindow = true
        cmd.SysProcAttr.CreationFlags |= createNoWindow
}

// GUI 서브시스템(-H windowsgui)으로 빌드하면 표준 출력 핸들이 없다.
// 터미널에서 실행한 경우엔 부모 콘솔에 붙어서 로그가 그대로 보이게 하고,
// -console 로 실행하면 콘솔 창을 새로 띄운다.
var (
        kernel32               = syscall.NewLazyDLL("kernel32.dll")
        procAttachConsole      = kernel32.NewProc("AttachConsole")
        procAllocConsole       = kernel32.NewProc("AllocConsole")
        procSetConsoleOutputCP = kernel32.NewProc("SetConsoleOutputCP")
)

const (
        attachParentProcess = uintptr(0xFFFFFFFF) // ATTACH_PARENT_PROCESS
        codePageUTF8        = uintptr(65001)
)

// attachConsole은 출력할 곳을 확보한다.
// alloc이 true면 붙을 콘솔이 없을 때 창을 새로 만든다.
// 아무 데도 못 쓰면 false를 돌려주고, 이때 stdout에 쓰면 실패한다.
func attachConsole(alloc bool) bool {
        // 표준 출력이 이미 살아 있으면(파일·파이프로 리다이렉트했거나
        // 콘솔 핸들을 물려받았으면) 그대로 쓴다.
        if stdoutUsable() {
                procSetConsoleOutputCP.Call(codePageUTF8)
                return true
        }

        r, _, _ := procAttachConsole.Call(attachParentProcess)
        if r == 0 && alloc {
                r, _, _ = procAllocConsole.Call()
        }
        if r == 0 {
                return false
        }

        // 한글이 깨지지 않도록 콘솔 코드페이지를 UTF-8로 맞춘다.
        procSetConsoleOutputCP.Call(codePageUTF8)

        // GUI 빌드는 표준 핸들이 비어 있으므로 콘솔 출력 장치를 직접 연다.
        f, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
        if err != nil {
                return false
        }
        os.Stdout = f
        os.Stderr = f
        return true
}

// GUI 서브시스템에서 콘솔 없이 실행하면 표준 출력 핸들이 비어 있고,
// 여기에 쓰면 실패한다. Stat으로 쓸 수 있는 핸들인지 확인한다.
func stdoutUsable() bool {
        if os.Stdout == nil {
                return false
        }
        _, err := os.Stdout.Stat()
        return err == nil
}
