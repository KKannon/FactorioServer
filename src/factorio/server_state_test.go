package factorio

import "testing"

func TestPrepareStartTransitionsServerAtomically(t *testing.T) {
	server := &Server{State: StateStopped}
	if err := server.PrepareStart("world.zip", "0.0.0.0", 34197); err != nil {
		t.Fatalf("PrepareStart returned an error: %v", err)
	}
	if state := server.GetState(); state != StateStarting {
		t.Fatalf("state = %q, want %q", state, StateStarting)
	}
	if !server.GetRunning() {
		t.Fatal("server must be considered busy while starting")
	}
	if err := server.PrepareStart("other.zip", "0.0.0.0", 34198); err == nil {
		t.Fatal("a concurrent start must be rejected")
	}
}

func TestTerminalStatesAreNotRunning(t *testing.T) {
	server := &Server{State: StateRunning, Running: true, RconConnected: true}
	server.SetState(StateError, "boom")
	if server.GetRunning() {
		t.Fatal("error state must not be reported as running")
	}
	if server.RconConnected {
		t.Fatal("RCON must be disconnected in an error state")
	}
	server.SetState(StateStopped, "")
	if server.GetRunning() {
		t.Fatal("stopped state must not be reported as running")
	}
}

func TestRememberProcessErrorKeepsUsefulCause(t *testing.T) {
	server := &Server{}
	server.rememberProcessError("   4.185 Error CommandLineMultiplayer.cpp:183: require_user_verification must be enabled for public games.")
	server.mu.RLock()
	detail := server.processError
	server.mu.RUnlock()
	if detail != "CommandLineMultiplayer.cpp:183: require_user_verification must be enabled for public games." {
		t.Fatalf("process error = %q", detail)
	}
}
