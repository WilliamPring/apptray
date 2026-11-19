package desktop

import (
	"reflect"
	"testing"
)

func TestCommand(t *testing.T) {
	tests := []struct {
		name string
		exec string
		want []string
	}{
		{
			"a field code is dropped, not passed along",
			"firefox %u",
			[]string{"firefox"},
		},
		{
			"flags are kept, the field code is not",
			"code --new-window %F",
			[]string{"code", "--new-window"},
		},
		{
			"a quoted path may contain spaces",
			`"/opt/my app/run" --flag`,
			[]string{"/opt/my app/run", "--flag"},
		},
		{
			"backslash escapes a quote inside quotes",
			`sh -c "echo \"hi\""`,
			[]string{"sh", "-c", `echo "hi"`},
		},
		{
			"a wrapper command is just argv[0]",
			"env GDK_BACKEND=x11 inkscape %F",
			[]string{"env", "GDK_BACKEND=x11", "inkscape"},
		},
		{
			"%% is a literal percent sign",
			"print 100%%",
			[]string{"print", "100%"},
		},
		{
			"runs of spaces do not produce empty arguments",
			"vlc   --fullscreen",
			[]string{"vlc", "--fullscreen"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Command(tt.exec)
			if err != nil {
				t.Fatalf("Command(%q) returned error: %v", tt.exec, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Command(%q) = %q, want %q", tt.exec, got, tt.want)
			}
		})
	}
}

func TestCommandErrors(t *testing.T) {
	tests := []struct {
		name string
		exec string
	}{
		{"empty", ""},
		{"nothing but a field code", "%U"},
		{"unterminated quote", `foo "bar`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Better a logged error than exec'ing something half-parsed.
			if _, err := Command(tt.exec); err == nil {
				t.Errorf("Command(%q) returned no error", tt.exec)
			}
		})
	}
}
