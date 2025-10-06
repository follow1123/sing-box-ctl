package platform

import (
	"fmt"
	"os"
	"os/exec"
)

func OpenUrl(url string) error {
	cmd := exec.Command("xdg-open", url)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run command '%s' error:\n\t%w", cmd.String(), err)
	}
	return nil
}

func LogFile(path string) error {
	command := fmt.Sprintf("tail -f %s", path)
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run command '%s' error:\n\t%w", cmd.String(), err)
	}
	return nil
}
