//go:build windows

package main

import (
        "syscall"
        "unsafe"
)

// 이름 있는 뮤텍스로 중복 실행을 막는다.
// 두 벌이 뜨면 두 번째는 포트 8080을 못 잡아 트레이만 떠 있는 상태가 되고,
// 설치·업그레이드 때는 exe 가 잠겨 절반만 덮어써질 수 있다.
//
// 이 이름은 Inno Setup 의 AppMutex 와 같아야 한다.
// 설치 프로그램이 "실행 중이니 종료해 주세요"를 대신 안내해 준다.
const appMutexName = "EomuiMusicSingleInstance"

const errorAlreadyExists = 183

var procCreateMutexW = kernel32.NewProc("CreateMutexW")

// 프로세스가 사는 동안 핸들을 붙잡아 둔다 (GC 되어도 상관없도록 전역).
var appMutexHandle uintptr

// claimSingleInstance는 이 프로세스가 유일한 인스턴스면 true 를 돌려준다.
func claimSingleInstance() bool {
        name, err := syscall.UTF16PtrFromString(appMutexName)
        if err != nil {
                return true // 이름을 못 만들면 막지 않는다
        }
        h, _, lastErr := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(name)))
        if h == 0 {
                return true
        }
        appMutexHandle = h
        return lastErr != syscall.Errno(errorAlreadyExists)
}
