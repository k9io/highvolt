//go:build !windows

package main

import (
	"github.com/kardianos/service"
	l "github.com/k9io/highvolt/internal/logger"
)

func InstallScheduler(s service.Service) error {
	if err := service.Control(s, "install"); err != nil {
		return err
	}
	l.Logger(l.NOTICE, "Service installed.")
	return nil
}

func UninstallScheduler(s service.Service) error {
	if err := service.Control(s, "uninstall"); err != nil {
		return err
	}
	l.Logger(l.NOTICE, "Service uninstalled.")
	return nil
}

func StartScheduler(s service.Service) error {
	if err := service.Control(s, "start"); err != nil {
		return err
	}
	l.Logger(l.NOTICE, "Service started.")
	return nil
}

func StopScheduler(s service.Service) error {
	if err := service.Control(s, "stop"); err != nil {
		return err
	}
	l.Logger(l.NOTICE, "Service stopped.")
	return nil
}

func RestartScheduler(s service.Service) error {
	if err := service.Control(s, "stop"); err != nil {
		return err
	}
	if err := service.Control(s, "start"); err != nil {
		return err
	}
	l.Logger(l.NOTICE, "Service restarted.")
	return nil
}
