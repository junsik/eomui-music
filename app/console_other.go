//go:build !windows

package main

// Windows가 아니면 표준 출력이 이미 유효하다.
func attachConsole(bool) bool { return true }
