package main

import (
	"fmt"
	"net"
	"net/http"
	"os/user"
	"strconv"

	"github.com/galexrt/srcds_controller/pkg/config"
	"golang.org/x/sys/unix"
)

func checkACL(r *http.Request) (bool, error) {
	conn := GetConn(r)
	if unixConn, isUnix := conn.(*net.UnixConn); isUnix {
		f, err := unixConn.File()
		if err != nil {
			return false, err
		}
		if f == nil {
			return false, fmt.Errorf("net connection fd is nil")
		}
		defer f.Close()

		pcred, err := unix.GetsockoptUcred(int(f.Fd()), unix.SOL_SOCKET, unix.SO_PEERCRED)
		if err != nil {
			return false, err
		}

		return checkPCREDAgainstACL(pcred, getServerACL())
	}
	return false, nil
}

func getServerACL() config.ACL {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()
	if config.Cfg.Server.ACL == nil {
		return config.ACL{}
	}
	return *config.Cfg.Server.ACL
}

func checkPCREDAgainstACL(cred *unix.Ucred, acl config.ACL) (bool, error) {
	for _, u := range acl.Users {
		if u == int(cred.Uid) {
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

	return false, fmt.Errorf("request user (%s) / groups (%+v) did not match with ACL", userID, userGroups)
}
