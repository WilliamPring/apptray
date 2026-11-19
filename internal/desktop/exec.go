package desktop

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

// fieldCodes are the %-placeholders the desktop entry spec allows in Exec=.
// They stand for files, URLs and icons passed to the app; we launch with no
// arguments, so every one of them is dropped rather than expanded.
//
// Leaving them in is not harmless: Firefox given a literal "%u" opens a tab
// for a file named "%u".
var fieldCodes = map[string]bool{
	"%f": true, "%F": true, // a file / a list of files
	"%u": true, "%U": true, // a URL / a list of URLs
	"%d": true, "%D": true, // deprecated
	"%n": true, "%N": true, // deprecated
	"%i": true, // expands to --icon <Icon>
	"%c": true, // the translated Name
	"%k": true, // the .desktop file's own path
	"%v": true, "%m": true, // deprecated
}

// Command splits an Exec= value into argv the way the spec says to: arguments
// separated by spaces, double quotes around arguments that contain spaces,
// backslash escapes inside those quotes, and field codes removed.
//
// It returns an error for an unterminated quote or an Exec= that holds nothing
// but field codes, because both mean we have no program to run.
func Command(execLine string) ([]string, error) {
	var (
		args     []string
		cur      strings.Builder
		inQuote  bool
		hasToken bool // distinguishes `foo ""` (two args) from `foo ` (one)
	)

	for i := 0; i < len(execLine); i++ {
		c := execLine[i]
		switch {
		case inQuote && c == '\\' && i+1 < len(execLine):
			// Inside quotes the spec escapes ", `, $ and \ with a backslash.
			i++
			cur.WriteByte(execLine[i])
			hasToken = true
		case c == '"':
			inQuote = !inQuote
			hasToken = true
		case !inQuote && (c == ' ' || c == '\t'):
			if hasToken {
				args = append(args, cur.String())
				cur.Reset()
				hasToken = false
			}
		default:
			cur.WriteByte(c)
			hasToken = true
		}
	}
	if inQuote {
		return nil, fmt.Errorf("unterminated quote in Exec=%q", execLine)
	}
	if hasToken {
		args = append(args, cur.String())
	}

	argv := make([]string, 0, len(args))
	for _, a := range args {
		if fieldCodes[a] {
			continue
		}
		// "%%" is how the spec writes a literal percent sign.
		argv = append(argv, strings.ReplaceAll(a, "%%", "%"))
	}
	if len(argv) == 0 {
		return nil, errors.New("Exec= has no command to run")
	}
	return argv, nil
}

// Launch starts the app and returns as soon as it has been handed to the OS.
// It does not wait for the app to finish — apptray is about to close.
func (e Entry) Launch() error {
	argv, err := Command(e.Exec)
	if err != nil {
		return fmt.Errorf("%s: %w", e.ID, err)
	}

	cmd := exec.Command(argv[0], argv[1:]...)

	// Setsid puts the app in its own session. Without it the app is a child
	// of apptray: it dies with us, or keeps our terminal alive, depending on
	// how apptray was started. A launcher must outlive neither.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting %s: %w", e.ID, err)
	}

	// Nothing calls Wait, so reap the child in the background rather than
	// leaving a zombie behind if the window happens to stay open.
	go cmd.Wait()
	return nil
}
