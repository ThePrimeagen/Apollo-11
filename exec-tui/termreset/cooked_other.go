//go:build !unix

package termreset

import "fmt"

func enableCooked(fd uintptr) error {
	return fmt.Errorf("termreset: cooked restore unsupported on this platform (fd %d)", fd)
}
