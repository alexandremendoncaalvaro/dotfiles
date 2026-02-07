package module

import "testing"

func TestStatusKind_String(t *testing.T) {
	tests := []struct {
		kind StatusKind
		want string
	}{
		{Installed, "instalado"},
		{Missing, "ausente"},
		{Partial, "parcial"},
		{Skipped, "pulado"},
		{StatusKind(99), "desconhecido"},
	}

	for _, tt := range tests {
		got := tt.kind.String()
		if got != tt.want {
			t.Errorf("StatusKind(%d).String() = %q, esperava %q", tt.kind, got, tt.want)
		}
	}
}
