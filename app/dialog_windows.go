//go:build windows

package main

import (
        "syscall"
        "unsafe"
)

var (
        user32          = syscall.NewLazyDLL("user32.dll")
        procMessageBoxW = user32.NewProc("MessageBoxW")
)

const (
        mbOK           = 0x00000000
        mbYesNo        = 0x00000004
        mbIconQuestion = 0x00000020
        mbIconInfo     = 0x00000040
        mbSystemModal  = 0x00001000 // 트레이 앱이라 창이 뒤에 숨지 않도록
        idYes          = 6
)

func messageBox(title, text string, flags uintptr) int {
        t, err := syscall.UTF16PtrFromString(text)
        if err != nil {
                return 0
        }
        c, err := syscall.UTF16PtrFromString(title)
        if err != nil {
                return 0
        }
        r, _, _ := procMessageBoxW.Call(0,
                uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), flags)
        return int(r)
}

// confirmDialog는 되돌리기 어려운 작업 전에 사람에게 물어본다.
func confirmDialog(title, text string) bool {
        return messageBox(title, text, mbYesNo|mbIconQuestion|mbSystemModal) == idYes
}

func infoDialog(title, text string) {
        messageBox(title, text, mbOK|mbIconInfo|mbSystemModal)
}
