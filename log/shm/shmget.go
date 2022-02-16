package shm

import (
	"syscall"
)

const (
	// IPC_CREATE create if key is nonexistent
	IPC_CREATE = 00001000
)

// Shmget 创建shmid
// 参数参见unix的系统调用shmget函数
// 创建一个读写的空间:shmflg = IPC_CREATE|0600
func Shmget(key, size, shmflg uintptr) (shmid uintptr, err syscall.Errno) {
	shmid, _, err = syscall.Syscall(syscall.SYS_SHMGET, key, size, shmflg)
	return
}

// Shmat attach到shmid的地址
// 返回进程可以访问的地址值
func Shmat(shmid uintptr) (addr uintptr, err syscall.Errno) {
	addr, _, err = syscall.Syscall(syscall.SYS_SHMAT, shmid, 0, 0)
	return
}

// Shmdt deattach shmid
func Shmdt(shmid uintptr) syscall.Errno {
	_, _, err := syscall.Syscall(syscall.SYS_SHMDT, shmid, 0, 0)
	return err
}
