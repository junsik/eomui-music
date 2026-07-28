//go:build !windows

package main

// 중복 실행 방지는 Windows 전용이다. 다른 OS에서는 포트 바인딩 실패로 드러난다.
func claimSingleInstance() bool { return true }
