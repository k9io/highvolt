//go:build windows

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/kardianos/service"
	l "github.com/k9io/highvolt/internal/logger"
)

const taskName = "Highvolt Voltage"

const taskXML = `<?xml version="1.0" encoding="UTF-8"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>Highvolt Voltage - daily sensitive document scanner</Description>
  </RegistrationInfo>
  <Triggers>
    <CalendarTrigger>
      <StartBoundary>2026-01-01T02:00:00</StartBoundary>
      <Enabled>true</Enabled>
      <ScheduleByDay>
        <DaysInterval>1</DaysInterval>
      </ScheduleByDay>
    </CalendarTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <LogonType>InteractiveToken</LogonType>
      <RunLevel>HighestAvailable</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <StartWhenAvailable>true</StartWhenAvailable>
    <ExecutionTimeLimit>PT4H</ExecutionTimeLimit>
    <Enabled>true</Enabled>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>{{.ExePath}}</Command>
      <Arguments>--once</Arguments>
    </Exec>
  </Actions>
</Task>`

func exePath() (string, error) {

	p, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot determine executable path: %w", err)
	}

	p, err = filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("cannot resolve executable path: %w", err)
	}

	return p, nil
}

func writeTaskXML() (string, error) {

	exe, err := exePath()
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("task").Parse(taskXML)
	if err != nil {
		return "", fmt.Errorf("cannot parse task XML template: %w", err)
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, struct{ ExePath string }{exe}); err != nil {
		return "", fmt.Errorf("cannot render task XML: %w", err)
	}

	tmp, err := os.CreateTemp("", "voltage-task-*.xml")
	if err != nil {
		return "", fmt.Errorf("cannot create temp file: %w", err)
	}

	if _, err = tmp.WriteString(buf.String()); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}

	tmp.Close()
	return tmp.Name(), nil
}

func schtasks(args ...string) error {
	cmd := exec.Command("schtasks", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func InstallScheduler(_ service.Service) error {

	xmlPath, err := writeTaskXML()
	if err != nil {
		return err
	}
	defer os.Remove(xmlPath)

	if err = schtasks("/Create", "/TN", taskName, "/XML", xmlPath, "/F"); err != nil {
		return fmt.Errorf("schtasks /Create failed: %w", err)
	}

	l.Logger(l.NOTICE, "Scheduled task '%s' installed (daily at 02:00, runs on wake).", taskName)
	return nil
}

func UninstallScheduler(_ service.Service) error {

	if err := schtasks("/Delete", "/TN", taskName, "/F"); err != nil {
		return fmt.Errorf("schtasks /Delete failed: %w", err)
	}

	l.Logger(l.NOTICE, "Scheduled task '%s' removed.", taskName)
	return nil
}

func StartScheduler(_ service.Service) error {

	if err := schtasks("/Run", "/TN", taskName); err != nil {
		return fmt.Errorf("schtasks /Run failed: %w", err)
	}

	l.Logger(l.NOTICE, "Scheduled task '%s' started.", taskName)
	return nil
}

func StopScheduler(_ service.Service) error {

	if err := schtasks("/End", "/TN", taskName); err != nil {
		return fmt.Errorf("schtasks /End failed: %w", err)
	}

	l.Logger(l.NOTICE, "Scheduled task '%s' stopped.", taskName)
	return nil
}

func RestartScheduler(s service.Service) error {

	if err := StopScheduler(s); err != nil {
		return err
	}

	return StartScheduler(s)
}
