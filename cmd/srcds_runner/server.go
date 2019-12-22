// Code taken from https://stackoverflow.com/a/55329317 by Paul Donohue

package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/user"
	"strconv"

	"github.com/galexrt/srcds_controller/pkg/config"
	"golang.org/x/sys/unix"
)

var conns = make(map[string]net.Conn)

type connSaveListener struct {
	net.Listener
}

func NewConnSaveListener(wrap net.Listener) net.Listener {
	return connSaveListener{wrap}
}

func (self connSaveListener) Accept() (net.Conn, error) {
	conn, err := self.Listener.Accept()
	ptrStr := fmt.Sprintf("%d", &conn)
	conns[ptrStr] = conn
	return remoteAddrPtrConn{conn, ptrStr}, err
}

func GetConn(r *http.Request) net.Conn {
	return conns[r.RemoteAddr]
}

func ConnStateEvent(conn net.Conn, event http.ConnState) {
	if event == http.StateHijacked || event == http.StateClosed {
		delete(conns, conn.RemoteAddr().String())
	}
}

type remoteAddrPtrConn struct {
	net.Conn
	ptrStr string
}

func (self remoteAddrPtrConn) RemoteAddr() net.Addr {
	return remoteAddrPtr{self.ptrStr}
}

type remoteAddrPtr struct {
	ptrStr string
}

func (remoteAddrPtr) Network() string {
	return ""
}

func (self remoteAddrPtr) String() string {
	return self.ptrStr
}

func NewUnixListener(path string) (net.Listener, error) {
	if err := unix.Unlink(path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	l, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}

	if err := os.Chmod(path, 0660); err != nil {
		l.Close()
		return nil, err
	}

	return l, nil
}

func checkACL(r *http.Request) (bool, error) {
	conn := GetConn(r)
	if unixConn, isUnix := conn.(*net.UnixConn); isUnix {
		f, err := unixConn.File()
		if err != nil {
			return false, err
		}
		defer f.Close()

		pcred, err := unix.GetsockoptUcred(int(f.Fd()), unix.SOL_SOCKET, unix.SO_PEERCRED)
		if err != nil {
			return false, err
		}

		return checkPCREDAgainstACL(pcred, config.Cfg.Server.ACL)
	}
	return false, nil
}

func checkPCREDAgainstACL(cred *unix.Ucred, acl *config.ACL) (bool, error) {
	if acl == nil {
		return false, fmt.Errorf("no ACLs found, disallowing any access")
	}

	for _, u := range acl.Users {
		if uint32(u) == cred.Uid {
			return true, nil
		}
	}

	// Convert user ID to string
	userID := strconv.FormatUint(uint64(cred.Uid), 10)

	// Get Linux user groups
	userInfo, err := user.LookupId(userID)
	if err != nil {
		return false, err
	}
	userGroups, err := userInfo.GroupIds()
	if err != nil {
		return false, err
	}

	for _, g := range acl.Groups {
		aclG := strconv.Itoa(g)
		for _, ug := range userGroups {
			if aclG == ug {
				return true, nil
			}
		}
	}

	return false, nil
}
