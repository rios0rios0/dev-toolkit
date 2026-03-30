package system

import "runtime"

// IsAndroid reports whether the current platform is Android (Termux).
func IsAndroid() bool {
	return runtime.GOOS == "android"
}

// IsLinux reports whether the current platform is Linux (excludes Android).
func IsLinux() bool {
	return runtime.GOOS == "linux"
}
