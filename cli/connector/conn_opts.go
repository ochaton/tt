package connector

import (
	"strings"
)

const (
	TCPNetwork  = "tcp"
	UnixNetwork = "unix"
)

// ConnOpts describes the connection to a tarantool instance.
type ConnOpts struct {
	// Network is a characteristic of a connection like "type" ("tcp" and "unix" are used).
	Network string
	// Address of an instance.
	Address string
	// Username of the tarantool user.
	Username string
	// Password of the user.
	Password string
}

// GetConnOpts returns the connection parameters according to the passed arguments.
func GetConnOpts(connString, username, password string) *ConnOpts {
	connOpts := ConnOpts{
		Username: username,
		Password: password,
	}

	connStringParts := strings.SplitN(connString, "@", 2)
	address := connStringParts[len(connStringParts)-1]

	if len(connStringParts) > 1 {
		authString := connStringParts[0]
		authStringParts := strings.SplitN(authString, ":", 2)

		if connOpts.Username == "" {
			connOpts.Username = authStringParts[0]
		}
		if len(authStringParts) > 1 && connOpts.Password == "" {
			connOpts.Password = authStringParts[1]
		}
	}

	addrLen := len(address)
	switch {
	case addrLen > 0 && (address[0] == '.' || address[0] == '/'):
		connOpts.Network = UnixNetwork
		connOpts.Address = address
	case addrLen >= 7 && address[0:7] == "unix://":
		connOpts.Network = UnixNetwork
		connOpts.Address = address[7:]
	case addrLen >= 5 && address[0:5] == "unix:":
		connOpts.Network = UnixNetwork
		connOpts.Address = address[5:]
	case addrLen >= 6 && address[0:6] == "unix/:":
		connOpts.Network = UnixNetwork
		connOpts.Address = address[6:]
	case addrLen >= 6 && address[0:6] == "tcp://":
		connOpts.Network = TCPNetwork
		connOpts.Address = address[6:]
	case addrLen >= 4 && address[0:4] == "tcp:":
		connOpts.Network = TCPNetwork
		connOpts.Address = address[4:]
	default:
		connOpts.Network = TCPNetwork
		connOpts.Address = address
	}

	return &connOpts
}
