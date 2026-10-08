package items

import "testing"

func TestNewUUIDReturnsValidUUID(t *testing.T) {
	t.Parallel()

	id, err := NewUUID()
	if err != nil {
		t.Fatalf("expected UUID generation to succeed, got %v", err)
	}

	if !IsValidUUID(id) {
		t.Fatalf("expected generated UUID to be valid, got %q", id)
	}

	if id[14] != '4' {
		t.Fatalf("expected version 4 UUID, got %q", id)
	}
}

func TestIsValidUUID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "valid lowercase", value: "018fb3b2-0f9d-4f59-8a63-7ef4bb812345", want: true},
		{name: "valid uppercase", value: "018FB3B2-0F9D-4F59-8A63-7EF4BB812345", want: true},
		{name: "missing hyphens", value: "018fb3b20f9d4f598a637ef4bb812345", want: false},
		{name: "invalid hex", value: "018fb3b2-0f9d-4f59-8a63-7ef4bb81234z", want: false},
		{name: "empty", value: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsValidUUID(tt.value); got != tt.want {
				t.Fatalf("expected %t, got %t", tt.want, got)
			}
		})
	}
}
