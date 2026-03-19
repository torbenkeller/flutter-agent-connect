package flutter

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestVMServiceErrorParsing(t *testing.T) {
	tests := []struct {
		name        string
		errorJSON   string
		wantCode    int
		wantMessage string
	}{
		{
			name:        "feature disabled",
			errorJSON:   `{"code":100,"message":"Feature is disabled"}`,
			wantCode:    100,
			wantMessage: "Feature is disabled",
		},
		{
			name:        "isolate reloading",
			errorJSON:   `{"code":108,"message":"Isolate is reloading"}`,
			wantCode:    108,
			wantMessage: "Isolate is reloading",
		},
		{
			name:        "expression compilation error",
			errorJSON:   `{"code":113,"message":"Expression compilation error","data":{"details":"syntax error"}}`,
			wantCode:    113,
			wantMessage: "Expression compilation error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var vmErr VMServiceError
			if err := json.Unmarshal([]byte(tt.errorJSON), &vmErr); err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			if vmErr.Code != tt.wantCode {
				t.Errorf("code: got %d, want %d", vmErr.Code, tt.wantCode)
			}
			if vmErr.Message != tt.wantMessage {
				t.Errorf("message: got %q, want %q", vmErr.Message, tt.wantMessage)
			}
		})
	}
}

func TestVMServiceErrorString(t *testing.T) {
	err := &VMServiceError{Code: 108, Message: "Isolate is reloading"}
	expected := "VM Service error 108: Isolate is reloading"
	if err.Error() != expected {
		t.Errorf("got %q, want %q", err.Error(), expected)
	}
}

func TestVMServiceErrorIsTransient(t *testing.T) {
	tests := []struct {
		code      int
		transient bool
	}{
		{100, false}, // Feature is disabled
		{102, false}, // Cannot add breakpoint
		{105, true},  // Isolate must be runnable
		{108, true},  // Isolate is reloading
		{112, false}, // Service disappeared
		{113, false}, // Expression compilation error
	}

	for _, tt := range tests {
		err := &VMServiceError{Code: tt.code}
		if err.IsTransient() != tt.transient {
			t.Errorf("code %d: IsTransient() = %v, want %v", tt.code, err.IsTransient(), tt.transient)
		}
	}
}

func TestVMServiceErrorUnwrap(t *testing.T) {
	// Verify errors.As works through the chain
	originalErr := &VMServiceError{Code: 108, Message: "Isolate is reloading"}

	var vmErr *VMServiceError
	if !errors.As(originalErr, &vmErr) {
		t.Fatal("errors.As should match VMServiceError")
	}
	if vmErr.Code != 108 {
		t.Errorf("got code %d, want 108", vmErr.Code)
	}
}
