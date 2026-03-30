package system

import "os"

// IsAndroid reports whether the current platform is Android (Termux).
func IsAndroid() bool {
	_, err := os.Stat("/system/build.prop")
	return err == nil
}
