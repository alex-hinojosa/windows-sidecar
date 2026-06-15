package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func runDoctor() error {
	script, err := findDoctorScript()
	if err != nil {
		return err
	}

	shell, err := findPowerShell()
	if err != nil {
		return err
	}

	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)
	cmd := exec.Command(shell, "-ExecutionPolicy", "Bypass", "-File", script, "-Fix", "-BuildDir", exeDir, "-InstallDir", exeDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func findDoctorScript() (string, error) {
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)
	cwd, _ := os.Getwd()

	candidates := []string{
		filepath.Join(exeDir, "doctor-windows.ps1"),
		filepath.Join(exeDir, "..", "scripts", "doctor-windows.ps1"),
		filepath.Join(cwd, "scripts", "doctor-windows.ps1"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("doctor-windows.ps1 not found next to sidecar.exe or in scripts/")
}

func findPowerShell() (string, error) {
	if shell, err := exec.LookPath("pwsh"); err == nil {
		return shell, nil
	}
	if shell, err := exec.LookPath("powershell"); err == nil {
		return shell, nil
	}
	return "", fmt.Errorf("PowerShell was not found on PATH")
}
