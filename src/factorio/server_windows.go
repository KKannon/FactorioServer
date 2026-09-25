package factorio

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"time"
)

func sendCtrlCToPid(pid int) error {
	d, e := syscall.LoadDLL("kernel32.dll")
	if e != nil {
		return fmt.Errorf("load kernel32.dll: %w", e)
	}
	p, e := d.FindProc("GenerateConsoleCtrlEvent")
	if e != nil {
		return fmt.Errorf("find GenerateConsoleCtrlEvent: %w", e)
	}
	r, _, e := p.Call(uintptr(syscall.CTRL_C_EVENT), uintptr(pid))
	if r == 0 {
		return fmt.Errorf("generate console Ctrl+C event: %w", e)
	}
	return nil
}

func setCtrlHandlingIsDisabledForThisProcess(disabled bool) error {
	disabledInt := 0
	if disabled {
		disabledInt = 1
	}

	d, e := syscall.LoadDLL("kernel32.dll")
	if e != nil {
		return fmt.Errorf("load kernel32.dll: %w", e)
	}
	p, e := d.FindProc("SetConsoleCtrlHandler")
	if e != nil {
		return fmt.Errorf("find SetConsoleCtrlHandler: %w", e)
	}
	r, _, e := p.Call(uintptr(0), uintptr(disabledInt))
	if r == 0 {
		return fmt.Errorf("set console Ctrl+C handler: %w", e)
	}
	return nil
}

func (server *Server) Kill() error {
	if server.Cmd == nil || server.Cmd.Process == nil {
		return errors.New("Factorio process is not available")
	}
	server.SetState(StateStopping, "")
	err := server.Cmd.Process.Signal(os.Kill)
	if err != nil {
		if err.Error() == "os: process already finished" {
			server.SetRunning(false)
			return err
		}
		server.SetState(StateRunning, err.Error())
		log.Printf("Error sending SIGKILL to Factorio process: %s", err)
		return err
	}
	log.Println("Sent SIGKILL to Factorio process. Factorio forced to exit.")
	if err := server.CloseRCON(); err != nil {
		log.Printf("Error close rcon connection: %s", err)
	}

	return nil
}

func (server *Server) Stop() error {
	if server.Cmd == nil || server.Cmd.Process == nil {
		return errors.New("Factorio process is not available")
	}
	server.SetState(StateStopping, "")
	// Disable our own handling of CTRL+C, so we don't close when we send it to the console.
	if err := setCtrlHandlingIsDisabledForThisProcess(true); err != nil {
		server.SetState(StateRunning, err.Error())
		return err
	}

	// Send CTRL+C to all processes attached to the console (ourself, and the factorio server instance)
	if err := sendCtrlCToPid(0); err != nil {
		_ = setCtrlHandlingIsDisabledForThisProcess(false)
		server.SetState(StateRunning, err.Error())
		return err
	}
	log.Println("Sent SIGINT to Factorio process. Factorio shutting down...")
	if err := server.CloseRCON(); err != nil {
		log.Printf("Error close rcon connection: %s", err)
	}
	time.Sleep(20 * time.Millisecond)
	// Re-enable handling of CTRL+C after we're sure that the factorio server is shut down.
	if err := setCtrlHandlingIsDisabledForThisProcess(false); err != nil {
		return err
	}

	return nil
}

func (server *Server) checkProcessHealth(text string) {
	// check if the output indicates a server shutdown
	if strings.Contains(text, "ServerMultiplayerManager.cpp:783: updateTick(0) changing state from(Disconnected) to(Closed)") {
		// Somehow, the Factorio devs managed to code the game to react appropriately to CTRL+C, including
		// saving the game, but not actually exit. So, we still have to manually kill the process, and
		// for extra fun, there's no way to know when the server save has actually completed (unless we want
		// to inject filesystem logic into what should be a process-level Stop() routine), so our best option
		// is to just wait an arbitrary amount of time and hope that the save is successful in that time.
		server.Cmd.Process.Signal(os.Kill)
	}
}
