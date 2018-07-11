package process

import (
	"code.cloudfoundry.org/cfdev/config"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type HyperV struct {
	Config  config.Config
	Launchd Launchd
}

func (h *HyperV) CreateVM() error {
	var vmName = "cfdev"
	var cfdevEfiIso = filepath.Join(h.Config.CacheDir, "cfdev-efi.iso")
	var cfDepsIso = filepath.Join(h.Config.CacheDir, "cf-deps.iso")
	var cfDevVHD = filepath.Join(h.Config.CFDevHome, "cfdev.vhd")

	cmd := exec.Command("powershell.exe", "-Command", fmt.Sprintf("New-VM -Name %s -Generation 2 -NoVHD", vmName))
	err := cmd.Run()
	if err != nil {
		return err
	}

	cmd = exec.Command("powershell.exe", "-Command", fmt.Sprintf("Set-VM -Name %s "+
		"-AutomaticStartAction Nothing "+
		"-AutomaticStopAction ShutDown "+
		"-CheckpointType Disabled "+
		"-MemoryStartupBytes 5GB "+
		"-StaticMemory "+
		"-ProcessorCount 4", vmName))
	err = cmd.Run()
	if err != nil {
		return err
	}

	err = addVhdDrive(cfdevEfiIso, vmName)
	if err != nil {
		return err
	}

	err = addVhdDrive(cfDepsIso, vmName)
	if err != nil {
		return err
	}

	cmd = exec.Command("powershell.exe", "-Command", fmt.Sprintf("Remove-VMNetworkAdapter "+
		"-VMName %s "+
		"-Name 'Network Adapter'",
		vmName))
	err = cmd.Run()
	if err != nil {
		return err
	}

	if _, err := os.Stat(cfDevVHD); err == nil {
		err := os.RemoveAll(cfDevVHD)
		if err != nil {
			return err
		}
	}

	cmd = exec.Command("powershell.exe", "-Command", fmt.Sprintf("New-VHD -Path %s "+
		"-SizeBytes '200000000000' "+
		"-Dynamic", cfDevVHD))
	err = cmd.Run()
	if err != nil {
		return err
	}

	cmd = exec.Command("powershell.exe", "-Command", fmt.Sprintf("Add-VMHardDiskDrive -VMName %s "+
		"-Path %s", vmName, cfDevVHD))
	err = cmd.Run()
	if err != nil {
		return err
	}

	cmd = exec.Command("powershell.exe", "-Command", fmt.Sprintf("Set-VMFirmware "+
		"-VMName %s "+
		"-EnableSecureBoot Off "+
		"-FirstBootDevice $cdrom",
		vmName))
	err = cmd.Run()
	if err != nil {
		return err
	}

	cmd = exec.Command("powershell.exe", "-Command", fmt.Sprintf("Set-VMComPort "+
		"-VMName %s "+
		"-number 1 "+
		"-Path \\\\.\\pipe\\cfdev-com",
		vmName))
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func addVhdDrive(isoPath string, vmName string) error {

	fmt.Println(fmt.Sprintf("Add-VMDvdDrive -VMName %s -Path %s", vmName, isoPath))

	cmd := exec.Command("powershell.exe", "-Command", fmt.Sprintf("Add-VMDvdDrive -VMName %s -Path %s", vmName, isoPath))
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (h *HyperV) Start(vmName string) error {
	cmd := exec.Command("powershell.exe", "-Command", fmt.Sprintf("Start-VM -Name %s", vmName))
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}