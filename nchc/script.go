package nchc

import (
	"fmt"
	"os/exec"
	"strings"
)

func runScript(sName string, args []string) ([]byte, error) {
	fmt.Printf("Running %s script: < %s >\n", sName, strings.Join(args, " , "))
	cmd := exec.Command(sName, args...)
	return cmd.CombinedOutput()
}
