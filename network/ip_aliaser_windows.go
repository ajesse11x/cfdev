package network

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const loopback = "vEthernet (cfdev)"

func (h *HostNet) RemoveLoopbackAliases(addrs ...string) error {
	exists, err := switchExists()
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	_, err := h.Powershell.Output("Remove-VMSwitch -Name cfdev -force")
	return err
}

func (h *HostNet) AddLoopbackAliases(addrs ...string) error {
	fmt.Println("Setting up IP aliases for the BOSH Director & CF Router (requires administrator privileges)")

	if err := createInterface(); err != nil {
		return err
	}

	for _, addr := range addrs {
		exists, err := aliasExists(addr)

		if err != nil {
			return err
		}

		if exists {
			continue
		}

		err = addAlias(addr)
		if err != nil {
			return err
		}
	}
	return nil
}

func addAlias(alias string) error {
	cmd := exec.Command("netsh", "interface", "ip", "add", "address", loopback, alias, "255.255.255.255")

	if err := cmd.Run(); err != nil {
		return err
	}

	return waitForAlias(alias)
}

func createInterface() error {
	return createSwitchIfNotExist()
}

func (h *HostNet) aliasExists(alias string) (bool, error) {
	output, err := h.Powershell.Output("ipconfig")
	if err != nil {
		return false, err
	}

	return strings.Contains(output, alias), nil
}

func (h *HostNet) createSwitchIfNotExist() error {
	exists, err := switchExists()
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	_, err = h.Powershell.Output("New-VMSwitch -Name cfdev -SwitchType Internal -Notes 'Switch for CF Dev Networking'")
	return err
}

func (h *HostNet) switchExists() (bool, error) {
	output, err := h.Powershell.Output("Get-VMSwitch cfdev*")
	if err != nil {
		return false, err
	} else if output == "" {
		return false, nil
	}

	return true, nil
}

func waitForAlias(addr string) error {
	done := make(chan error)
	go func() {
		for {
			if exists, err := aliasExists(addr); !exists {
				time.Sleep(3 * time.Second)
			} else if err != nil {
				done <- err
				close(done)
				return
			} else {
				close(done)
				return
			}
		}
	}()

	select {
	case err := <-done:
		return err
	case _ = <-time.After(1 * time.Minute):
		return fmt.Errorf("timed out waiting for alias %s", addr)
	}
}


