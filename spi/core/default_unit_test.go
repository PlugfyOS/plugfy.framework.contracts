package core

import (
	"errors"
	"testing"
)

// TestDefaultUnitDispatch proves DefaultUnit routes Invoke to the registered
// Handler by method name, and returns a classifiable ErrMethodNotFound for an
// unregistered method (including on a zero-value DefaultUnit).
func TestDefaultUnitDispatch(t *testing.T) {
	var u DefaultUnit
	u.Desc = UnitDescriptor{ID: "com.acme.greeter", Version: "1.0.0", Kind: KindTool}
	u.Method("greet", func(_ UnitContext, in map[string]any) (map[string]any, error) {
		return map[string]any{"message": "hello " + in["name"].(string)}, nil
	}).Method("boom", func(_ UnitContext, _ map[string]any) (map[string]any, error) {
		return nil, &UnitError{Code: "x", Class: "transient", Message: "kaboom"}
	})

	tests := []struct {
		name       string
		method     string
		in         map[string]any
		wantOut    string // expected out["message"], "" if none
		wantErr    bool
		wantNotFnd bool   // expect ErrMethodNotFound
		wantClass  string // expected UnitError.Class on error
	}{
		{name: "registered method dispatches", method: "greet", in: map[string]any{"name": "ada"}, wantOut: "hello ada"},
		{name: "handler error propagates", method: "boom", wantErr: true, wantClass: "transient"},
		{name: "unknown method -> ErrMethodNotFound", method: "missing", wantErr: true, wantNotFnd: true, wantClass: "permanent"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := u.Invoke(nil, tc.method, tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (out=%v)", out)
				}
				var ue *UnitError
				if !errors.As(err, &ue) {
					t.Fatalf("expected *UnitError, got %T: %v", err, err)
				}
				if ue.ErrorClass() != tc.wantClass {
					t.Fatalf("class = %q, want %q", ue.ErrorClass(), tc.wantClass)
				}
				if tc.wantNotFnd {
					if ue.Code != "method_not_found" {
						t.Fatalf("code = %q, want method_not_found", ue.Code)
					}
					// ErrMethodNotFound must name the missing method.
					if got := ErrMethodNotFound(tc.method).Message; got != ue.Message {
						t.Fatalf("ErrMethodNotFound message = %q, want %q", ue.Message, got)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got, _ := out["message"].(string); got != tc.wantOut {
				t.Fatalf("out[message] = %q, want %q", got, tc.wantOut)
			}
		})
	}
}

// TestZeroValueDefaultUnitIsUnit proves an un-initialized DefaultUnit is already
// a usable Unit (every method returns ErrMethodNotFound, no panic).
func TestZeroValueDefaultUnitIsUnit(t *testing.T) {
	var u DefaultUnit
	if _, err := u.Invoke(nil, "anything", nil); err == nil {
		t.Fatal("zero-value DefaultUnit should return ErrMethodNotFound, got nil")
	}
	var _ Unit = &u // also asserts the Provider surface compiles on the zero value
	if u.Name() != "" || u.Kind() != "" {
		t.Fatalf("zero descriptor should yield empty Name/Kind, got %q/%q", u.Name(), u.Kind())
	}
}
