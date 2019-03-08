/*
Copyright 2019 Alexander Trost. All rights reserved.

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

package chcloser

import (
	"sync"
)

// ChannelCloser channel closer struct
type ChannelCloser struct {
	IsClosed bool
	sync.Mutex
}

// Close closes a given channel.
func (ch *ChannelCloser) Close(channelToClose chan<- struct{}) {
	ch.Mutex.Lock()
	if !ch.IsClosed {
		close(channelToClose)
		ch.IsClosed = true
	}
	ch.Mutex.Unlock()
}
