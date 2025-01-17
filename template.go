package main

var initTemplate string = `package main

import (
	"fmt"
	"os"
	"io"
	"os/exec"
	"strings"
	"syscall"
)

var execs []string= []string{
{{block "executables" .Executables}}{{range .}}{{printf "\t\"%s\",\n" .}}{{end}}{{end -}}
}

var exeArg [][]string = [][]string {
{{block "arguments" .Arguments}}{{range . -}}
{{printf "\t{"}} {{- range . -}}{{printf " \"%s\"," .}}{{end}}
{{- printf "},\n"}}
{{- end}}{{end -}}
}

func printClose(r io.ReadCloser, prefix string){
	defer r.Close()
	slurp, _ := io.ReadAll(r)
	if len(slurp) == 0 {
		return
	}
	fmt.Printf("[            ] %s\n%s", prefix, slurp)
}


func main() {
	// Safe guard to make sure this dynamically created executable does not harm the system
	// when executed by accident.
	if os.Getpid() != 1 {
		fmt.Fprintf(os.Stderr, "%s must only run with PID 1 for safety\n", os.Args[0])
		return
	}

	// Create a minimal environment for the Linux kernel
{{- block "environment" .Environment}}
{{range .}}{{ print . }}{{end}}
{{- end}}
	// Execute the testing executables
	for i, exe := range execs {
		cmd := exec.Command(fmt.Sprintf("./%s", exe), strings.Join(exeArg[i], ", "))
		fmt.Printf("[            ]\t%s\n", cmd.Path)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			fmt.Printf("[            ]\tFailed to redirect stderr for '%s': %v\n", exe, err)
			continue
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Printf("[            ]\tFailed to redirect stdout for '%s': %v\n", exe, err)
			continue
		}
		if err := cmd.Start(); err != nil {
			fmt.Printf("[            ]\tFailure starting %s: %v\n", exe, err)
			printClose(stderr, "stderr")
			stdout.Close()
			continue
		}
		for {
			var s syscall.WaitStatus
			var r syscall.Rusage
			if p, err := syscall.Wait4(-1, &s, 0, &r); p == cmd.Process.Pid {
				fmt.Printf("[            ]\t%s exited, exit status %d\n", cmd.Path, s.ExitStatus())
				printClose(stderr, "stderr")
				break
			} else if p != -1 {
				fmt.Printf("[            ]\tReaped PID %d, exit status %d\n", p, s.ExitStatus())
				printClose(stderr, "stderr")
				break
			} else {
				fmt.Printf("[            ]\tError from Wait4 for orphaned child: %v\n", err)
				printClose(stderr, "stderr")
				break
			}
		}

		if err := cmd.Process.Release(); err != nil {
			fmt.Printf("[            ]\tError releasing process %v: %v\n", cmd, err)
		}
		printClose(stdout, "stdout")
	}

	// Shut VM down
	if err := syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF); err != nil {
		fmt.Printf( "[            ]\tPower off failed: %v\n", err)
	}
}

`
