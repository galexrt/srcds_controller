/*
Copyright 2020 Alexander Trost <galexrt@googlemail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

// NewUnixListener create a new unix socket listener
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

// Code taken from https://stackoverflow.com/a/55329317 by Paul Donohue
// Thanks for this code snippet!

type contextKey struct {
	key string
}

// ConnContextKey context key name for saving the connection in the request context
var ConnContextKey = &contextKey{"http-conn"}

// SaveConnInContext save connection in the context function for http.Server.ConnContext
func SaveConnInContext(ctx context.Context, c net.Conn) context.Context {
	return context.WithValue(ctx, ConnContextKey, c)
}

// GetConn get http connection for request
func GetConn(r *http.Request) net.Conn {
	return r.Context().Value(ConnContextKey).(net.Conn)
}

func listenAndServe(r *gin.Engine) {
	// Make sure no stale sockets present
	os.Remove(ListenAddress)

	http.HandleFunc("/", r.ServeHTTP)

	l, err := NewUnixListener(ListenAddress)
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(ListenAddress)

	server := http.Server{
		ConnContext: SaveConnInContext,
	}
	if err := server.Serve(l); err != nil {
		logger.Error(errors.Wrap(err, "error during http serve"))
		return
	}
}
