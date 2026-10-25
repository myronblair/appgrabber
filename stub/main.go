//go:build windows

// AppGrabber installer stub. The server appends a JSON app list to this exe;
// at run time it reads that list and installs each app silently via winget.
package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const magic = "APGRBR01"

type app struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Winget string `json:"winget"`
	URL    string `json:"url"`
	Args   string `json:"args"`
}

type payload struct {
	Apps []app `json:"apps"`
}

var logf *os.File

func say(format string, a ...any) {
	line := fmt.Sprintf(format, a...)
	fmt.Println(line)
	if logf != nil {
		fmt.Fprintf(logf, "%s  %s\n", time.Now().Format("15:04:05"), line)
	}
}

func readPayload() (*payload, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(exe)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	tail := make([]byte, 16)
	if st.Size() < 16 {
		return nil, errors.New("no payload")
	}
	if _, err := f.ReadAt(tail, st.Size()-16); err != nil {
		return nil, err
	}
	if string(tail[8:]) != magic {
		return nil, errors.New("this installer has no app list; download it from the AppGrabber site")
	}
	n := int64(binary.LittleEndian.Uint64(tail[:8]))
	if n <= 0 || n > st.Size()-16 {
		return nil, errors.New("corrupt payload")
	}
	buf := make([]byte, n)
	if _, err := f.ReadAt(buf, st.Size()-16-n); err != nil {
		return nil, err
	}
	var p payload
	if err := json.Unmarshal(buf, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func isAdmin() bool {
	shell32 := syscall.NewLazyDLL("shell32.dll")
	r, _, _ := shell32.NewProc("IsUserAnAdmin").Call()
	return r != 0
}

func relaunchElevated() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	cwd, _ := syscall.UTF16PtrFromString(filepath.Dir(exe))
	r, _, e := syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW").Call(
		0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)), 0, uintptr(unsafe.Pointer(cwd)), 1)
	if r <= 32 {
		return fmt.Errorf("elevation failed: %v", e)
	}
	return nil
}

func pause() {
	fmt.Print("\nPress Enter to close...")
	bufio.NewReader(os.Stdin).ReadString('\n')
}

func hasWinget() bool {
	_, err := exec.LookPath("winget")
	return err == nil
}

// exit codes winget uses for "nothing to do"
func wingetOK(code int) bool {
	switch uint32(code) {
	case 0, 0x8A15002B, 0x8A150061, 0x8A15010D, 0x8A150109:
		return true
	}
	return false
}

func run(name string, args ...string) (int, error) {
	cmd := exec.Command(name, args...)
	if logf != nil {
		cmd.Stdout, cmd.Stderr = logf, logf
	}
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), err
	}
	return -1, err
}

func direct(a app) error {
	resp, err := http.Get(a.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download HTTP %d", resp.StatusCode)
	}
	ext := strings.ToLower(filepath.Ext(strings.SplitN(a.URL, "?", 2)[0]))
	tmp := filepath.Join(os.TempDir(), "appgrabber-"+a.ID+ext)
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		return err
	}
	defer os.Remove(tmp)
	if ext == ".msi" {
		_, err = run("msiexec", "/i", tmp, "/qn", "/norestart")
		return err
	}
	_, err = run(tmp, strings.Fields(a.Args)...)
	return err
}

func main() {
	logf, _ = os.Create(filepath.Join(os.TempDir(), "AppGrabber.log"))
	fmt.Println("=== AppGrabber ===")

	p, err := readPayload()
	if err != nil {
		say("Error: %v", err)
		pause()
		os.Exit(1)
	}
	if !isAdmin() {
		fmt.Println("Requesting administrator rights...")
		if err := relaunchElevated(); err != nil {
			say("%v", err)
			pause()
		}
		return
	}

	say("Installing %d app(s). Log: %s", len(p.Apps), logf.Name())
	winget := hasWinget()
	if !winget {
		say("winget not found; using direct downloads where available.")
	}

	var failed []string
	for i, a := range p.Apps {
		prefix := fmt.Sprintf("[%d/%d] %s", i+1, len(p.Apps), a.Name)
		var err error
		switch {
		case winget && a.Winget != "":
			code, e := run("winget", "install", "--id", a.Winget, "-e", "--silent",
				"--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity")
			if !wingetOK(code) {
				err = fmt.Errorf("winget exit %d (%v)", code, e)
			}
		case a.URL != "":
			err = direct(a)
		default:
			err = errors.New("needs winget (not installed on this system)")
		}
		if err != nil {
			say("%s ... FAILED: %v", prefix, err)
			failed = append(failed, a.Name)
		} else {
			say("%s ... done", prefix)
		}
	}

	fmt.Println()
	if len(failed) == 0 {
		say("All done. Everything installed.")
	} else {
		say("Finished with %d problem(s): %s", len(failed), strings.Join(failed, ", "))
		say("Details are in %s", logf.Name())
	}
	pause()
}
