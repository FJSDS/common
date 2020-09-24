// +build windows

package reuseport

import "net"

func Listen(network, addr string) (l net.Listener, err error) {
	return net.Listen(network, addr)
}
