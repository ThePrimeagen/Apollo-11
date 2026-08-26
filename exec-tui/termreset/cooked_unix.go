//go:build unix

package termreset

import "golang.org/x/sys/unix"

func enableCooked(fd uintptr) error {
	termios, err := unix.IoctlGetTermios(int(fd), ioctlReadTermios)
	if err != nil {
		return err
	}
	termios.Oflag |= unix.OPOST | unix.ONLCR
	return unix.IoctlSetTermios(int(fd), ioctlWriteTermios, termios)
}
