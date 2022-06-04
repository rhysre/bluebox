package main

var initTemplate string = `package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		return err
	}

	// Preserve file permissions.
	if err := os.Chmod(dst, sourceFileStat.Mode()); err != nil {
		return err
	}

	return nil
}

func moveElements() error {
	return filepath.WalkDir("/", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "/init" && !d.IsDir() {
			return nil
		}

		if d.IsDir() {
			if path == "/bluebox" || path == "/dev" {
				// /dev is provided by the kernel. /bluebox will be the new
				// root fs and destination of moving all elements.
				return fs.SkipDir
			}
			return os.MkdirAll(filepath.Join("/bluebox", path), 0o755)
		}

		// Move the file into the new FS.
		if err := copyFile(path, filepath.Join("/bluebox", path)); err != nil {
			return err
		}

		// With the file in the new FS remove it from the old one.
		return os.Remove(path)
	})
}

func prepareNewRoot() error {
	if err := os.Mkdir("/bluebox", 0o750); err != nil {
		return err
	}

	if err := syscall.Mount("bluebox", "/bluebox", "tmpfs", uintptr(0), ""); err != nil {
		return err
	}

	if err := moveElements(); err != nil {
		return err
	}

	return nil
}

func switchRoot() error {
	if err := os.Chdir("/bluebox"); err != nil {
		return err
	}

	if err := syscall.Mount(".", "/", "", syscall.MS_MOVE, ""); err != nil {
		return err
	}

	if err := syscall.Chroot("."); err != nil {
		return err
	}

	if err := os.Chdir("/"); err != nil {
		return err
	}

	return nil
}

func main() {
	// Safe guard to make sure this dynamically created executable does not harm the system
	// when executed by accident.
	if os.Getpid() != 1 {
		fmt.Fprintf(os.Stderr, "%s must only run with PID 1 for safety\n", os.Args[0])
		return
	}

	if err := prepareNewRoot(); err != nil {
		fmt.Fprintf(os.Stderr, "prepareNewRoot: %v\n", err)
		return
	}

	if err := switchRoot(); err != nil {
		fmt.Fprintf(os.Stderr, "switchRoot: %v\n", err)
		return
	}

	// Create a minimal environment for the Linux kernel
{{- block "environment" .Environment}}
{{range .}}{{ print . }}{{end}}
{{- end}}

	// This call never returns.
	// Hand over to new init. This call never returns.
	if err := syscall.Exec("./bluebox-init", []string{}, []string{}); err != nil {
		fmt.Fprintf(os.Stderr, "exec: %v\n", err)
		return
	}
}
`

var blueboxTemplate string = `package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

const (
	// TMPFS_MAGIC from Linux kernel include/uapi/linux/magic.h
	TMPFS_MAGIC = 0x1021994
)

var execs []string = []string{
{{block "executables" .Executables}}{{range .}}{{printf "\t\"%s\",\n" .}}{{end}}{{end -}}
}

var exeArg [][]string = [][]string{
{{block "arguments" .Arguments}}{{range . -}}
{{printf "\t{"}} {{- range . -}}{{printf " \"%s\"," .}}{{end}}
{{- printf "},\n"}}
{{- end}}{{end -}}
}

func printClose(r io.ReadCloser, prefix string) {
	defer r.Close()
	slurp, _ := io.ReadAll(r)
	if len(slurp) == 0 {
		return
	}
	fmt.Printf("[            ] %s\n%s", prefix, slurp)
}

func checkFS() error {
	stat := syscall.Statfs_t{}
	if err := syscall.Statfs("/", &stat); err != nil {
		return fmt.Errorf("statfs: %v\n", err)
	}
	if stat.Type != TMPFS_MAGIC {
		fmt.Printf("[            ]\tExpected to be executed on TMPFS but is 0x%x\n", stat.Type)
	}
	return nil
}

func main() {
	if err := checkFS(); err != nil {
		fmt.Fprintf(os.Stderr, "[            ]\tFailed to check FS: %v\n", err)
	}

	// Execute the testing executables
	for i, exe := range execs {
		cmd := exec.Command(fmt.Sprintf("./%s", exe), strings.Join(exeArg[i], ", "))
		fmt.Printf("[            ]\t%s %s\n", cmd.Path, strings.Join(exeArg[i], ", "))
		stderr, err := cmd.StderrPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[            ]\tFailed to redirect stderr for '%s': %v\n", exe, err)
			continue
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[            ]\tFailed to redirect stdout for '%s': %v\n", exe, err)
			continue
		}
		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "[            ]\tFailure starting %s: %v\n", exe, err)
			printClose(stderr, "stderr")
			stdout.Close()
			continue
		}
		for {
			var s syscall.WaitStatus
			var r syscall.Rusage
			if p, err := syscall.Wait4(-1, &s, 0, &r); p == cmd.Process.Pid {
				fmt.Fprintf(os.Stderr, "[            ]\t%s exited, exit status %d\n", cmd.Path, s.ExitStatus())
				printClose(stderr, "stderr")
				break
			} else if p != -1 {
				fmt.Fprintf(os.Stderr, "[            ]\tReaped PID %d, exit status %d\n", p, s.ExitStatus())
				printClose(stderr, "stderr")
				break
			} else {
				fmt.Fprintf(os.Stderr, "[            ]\tError from Wait4 for orphaned child: %v\n", err)
				printClose(stderr, "stderr")
				break
			}
		}

		if err := cmd.Process.Release(); err != nil {
			fmt.Fprintf(os.Stderr, "[            ]\tError releasing process %v: %v\n", cmd, err)
		}
		printClose(stdout, "stdout")
	}

	// Shut VM down
	if err := syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF); err != nil {
		fmt.Printf("[            ]\tPower off failed: %v\n", err)
	}
}
`
