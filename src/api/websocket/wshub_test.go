package websocket

import (
	"sync"
	"testing"
)

func TestGetRoomIsSafeForConcurrentCreation(t *testing.T) {
	hub := &wsHub{rooms: make(map[string]*wsRoom)}
	const callers = 32
	rooms := make(chan *wsRoom, callers)
	var wait sync.WaitGroup
	wait.Add(callers)
	for index := 0; index < callers; index++ {
		go func() {
			defer wait.Done()
			rooms <- hub.GetRoom("server_status")
		}()
	}
	wait.Wait()
	close(rooms)

	var expected *wsRoom
	for room := range rooms {
		if expected == nil {
			expected = room
			continue
		}
		if room != expected {
			t.Fatal("concurrent callers received different room instances")
		}
	}
}
