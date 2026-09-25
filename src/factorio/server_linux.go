// use this file only when compiling not windows (all unix systems)
//go:build !windows
// +build !windows

package factorio

import (
	"errors"
	"log"
	"os"
)

// Stubs for windows-only functions

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
	log.Printf("Sent SIGKILL to Factorio process. Factorio forced to exit.")

	if err = server.CloseRCON(); err != nil {
		log.Printf("Error close rcon connection: %s", err)
	}

	return nil
}

func (server *Server) Stop() error {
	if server.Cmd == nil || server.Cmd.Process == nil {
		return errors.New("Factorio process is not available")
	}
	server.SetState(StateStopping, "")
	err := server.Cmd.Process.Signal(os.Interrupt)
	if err != nil {
		if err.Error() == "os: process already finished" {
			server.SetRunning(false)
			return err
		}
		server.SetState(StateRunning, err.Error())
		log.Printf("Error sending SIGINT to Factorio process: %s", err)
		return err
	}
	log.Printf("Sent SIGINT to Factorio process. Factorio shutting down...")

	if err = server.CloseRCON(); err != nil {
		log.Printf("Error close rcon connection: %s", err)
	}

	return nil
}

func (server *Server) checkProcessHealth(text string) {
	// ignore
}
