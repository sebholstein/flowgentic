package portutil

import (
	"fmt"
	"net"
)

// FindFreePort asks the OS for a free TCP port and returns it.
func FindFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}

// FindFreePortFrom tries to bind the given port. If it's already in use, it
// increments the port number up to maxAttempts times before falling back to
// an OS-assigned port.
func FindFreePortFrom(preferred int, maxAttempts int) (int, error) {
	for i := range maxAttempts {
		port := preferred + i
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			continue
		}
		l.Close()
		return port, nil
	}
	return FindFreePort()
}
