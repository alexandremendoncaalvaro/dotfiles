package profile

import "testing"

func TestByName_ValidProfiles(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"full", "full"},
		{"minimal", "minimal"},
		{"server", "server"},
		{"wsl", "wsl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prof, err := ByName(tt.name)
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if prof.Name != tt.expected {
				t.Errorf("esperava perfil %s, obteve %s", tt.expected, prof.Name)
			}
		})
	}
}

func TestByName_InvalidProfile(t *testing.T) {
	_, err := ByName("typo")
	if err == nil {
		t.Fatal("esperava erro para perfil invalido, obteve nil")
	}
}

func TestAll_ReturnsAllProfiles(t *testing.T) {
	profiles := All()
	if len(profiles) != 4 {
		t.Fatalf("esperava 4 perfis, obteve %d", len(profiles))
	}

	names := map[string]bool{}
	for _, p := range profiles {
		names[p.Name] = true
	}

	for _, name := range []string{"full", "minimal", "server", "wsl"} {
		if !names[name] {
			t.Errorf("perfil %s ausente em All()", name)
		}
	}
}
