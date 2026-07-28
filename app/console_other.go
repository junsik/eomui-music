//go:build !windows

package main

import "os/exec"

// Windows가 아니면 표준 출력이 이미 유효하다.
func attachConsole(bool) bool { return true }

// 자식 프로세스가 콘솔 창을 띄우는 것은 Windows 만의 문제다.
func hideConsole(*exec.Cmd) {}
