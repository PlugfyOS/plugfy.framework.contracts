package grpcstatus

import (
	"errors"
	"fmt"
	"testing"

	"github.com/PlugfyOS/plugfy.framework.contracts/errs"
)

// allClasses is the full set of canonical error classes the taxonomy defines.
var allClasses = []errs.Class{
	errs.ClassValidation,
	errs.ClassUnauthorized,
	errs.ClassForbidden,
	errs.ClassNotFound,
	errs.ClassConflict,
	errs.ClassRateLimit,
	errs.ClassUpstream,
	errs.ClassTimeout,
	errs.ClassInternal,
}

// TestCanonicalCodeNumbers freezes the integer values of the local Code enum to
// the canonical gRPC wire numbers. A consumer converts Code<->codes.Code by a
// bare integer cast, so these MUST match google.golang.org/grpc/codes exactly;
// this test is the contract that lets the baseplate name the codes without
// importing grpc.
func TestCanonicalCodeNumbers(t *testing.T) {
	want := map[Code]uint32{
		CodeOK:                 0,
		CodeCanceled:           1,
		CodeUnknown:            2,
		CodeInvalidArgument:    3,
		CodeDeadlineExceeded:   4,
		CodeNotFound:           5,
		CodeAlreadyExists:      6,
		CodePermissionDenied:   7,
		CodeResourceExhausted:  8,
		CodeFailedPrecondition: 9,
		CodeAborted:            10,
		CodeOutOfRange:         11,
		CodeUnimplemented:      12,
		CodeInternal:           13,
		CodeUnavailable:        14,
		CodeDataLoss:           15,
		CodeUnauthenticated:    16,
	}
	for c, n := range want {
		if uint32(c) != n {
			t.Errorf("Code %s = %d, want canonical gRPC %d", c, uint32(c), n)
		}
	}
}

func TestCodeString(t *testing.T) {
	if CodeNotFound.String() != "NotFound" {
		t.Errorf("CodeNotFound.String() = %q, want %q", CodeNotFound.String(), "NotFound")
	}
	if got := Code(99).String(); got != "Code(99)" {
		t.Errorf("Code(99).String() = %q, want %q", got, "Code(99)")
	}
}

func TestCodeFor(t *testing.T) {
	want := map[errs.Class]Code{
		errs.ClassValidation:   CodeInvalidArgument,
		errs.ClassUnauthorized: CodeUnauthenticated,
		errs.ClassForbidden:    CodePermissionDenied,
		errs.ClassNotFound:     CodeNotFound,
		errs.ClassConflict:     CodeAlreadyExists,
		errs.ClassRateLimit:    CodeResourceExhausted,
		errs.ClassUpstream:     CodeUnavailable,
		errs.ClassTimeout:      CodeDeadlineExceeded,
		errs.ClassInternal:     CodeInternal,
		errs.Class("garbage"):  CodeInternal,
	}
	for class, code := range want {
		if got := CodeFor(class); got != code {
			t.Errorf("CodeFor(%q) = %v, want %v", class, got, code)
		}
	}
}

// TestClassCodeRoundTrip pins the key invariant: every defined class projects to
// a code that projects back to the same class.
func TestClassCodeRoundTrip(t *testing.T) {
	for _, class := range allClasses {
		if got := ClassFor(CodeFor(class)); got != class {
			t.Errorf("ClassFor(CodeFor(%q)) = %q, want %q", class, got, class)
		}
	}
}

func TestClassForFoldsUnmappedCodes(t *testing.T) {
	cases := map[Code]errs.Class{
		CodeOK:                 errs.ClassInternal,
		CodeCanceled:           errs.ClassTimeout,
		CodeUnknown:            errs.ClassInternal,
		CodeOutOfRange:         errs.ClassValidation,
		CodeFailedPrecondition: errs.ClassValidation,
		CodeAborted:            errs.ClassConflict,
		CodeUnimplemented:      errs.ClassInternal,
		CodeDataLoss:           errs.ClassInternal,
		Code(99):               errs.ClassInternal,
	}
	for code, class := range cases {
		if got := ClassFor(code); got != class {
			t.Errorf("ClassFor(%v) = %q, want %q", code, got, class)
		}
	}
}

// TestCodeForMatchesHTTPTaxonomy asserts the gRPC mapping is the same coarse
// taxonomy as the HTTP one: a class that is a client error over HTTP (4xx) is a
// client-side code over gRPC, and a server/upstream error (5xx) is a
// server-side code. This is the "reuse the HTTPStatus() taxonomy" guarantee.
func TestCodeForMatchesHTTPTaxonomy(t *testing.T) {
	clientCodes := map[Code]bool{
		CodeInvalidArgument:   true,
		CodeUnauthenticated:   true,
		CodePermissionDenied:  true,
		CodeNotFound:          true,
		CodeAlreadyExists:     true,
		CodeResourceExhausted: true,
	}
	for _, class := range allClasses {
		http4xx := class.HTTPStatus() >= 400 && class.HTTPStatus() < 500
		grpcClient := clientCodes[CodeFor(class)]
		if http4xx != grpcClient {
			t.Errorf("class %q: HTTP %d (4xx=%v) but gRPC %v (client=%v); taxonomies diverge",
				class, class.HTTPStatus(), http4xx, CodeFor(class), grpcClient)
		}
	}
}

func TestFromErrorNil(t *testing.T) {
	if st := FromError(nil); st != nil {
		t.Errorf("FromError(nil) = %v, want nil", st)
	}
}

func TestFromErrorPlugfyError(t *testing.T) {
	e := errs.New(errs.ClassNotFound, errs.CodeNotFoundEntity, "file not found")
	st := FromError(e)
	if st.Code != CodeNotFound {
		t.Fatalf("code = %v, want %v", st.Code, CodeNotFound)
	}
	// Round-trip back to a classified error: class, code and message survive.
	back, ok := ToError(st).(*errs.Error)
	if !ok {
		t.Fatalf("ToError returned %T, want *errs.Error", ToError(st))
	}
	if back.Class != errs.ClassNotFound {
		t.Errorf("class = %q, want %q", back.Class, errs.ClassNotFound)
	}
	if back.Code != errs.CodeNotFoundEntity {
		t.Errorf("code = %q, want %q", back.Code, errs.CodeNotFoundEntity)
	}
	if back.Message != "file not found" {
		t.Errorf("message = %q, want %q", back.Message, "file not found")
	}
}

// TestRoundTripAllClasses round-trips a coded error of every class through a
// status and asserts the class and code are preserved end-to-end.
func TestRoundTripAllClasses(t *testing.T) {
	for _, class := range allClasses {
		code := string(class) + ".sample_code"
		e := errs.New(class, code, "human message for "+string(class))
		back, ok := ToError(FromError(e)).(*errs.Error)
		if !ok {
			t.Fatalf("class %q: ToError returned non-*errs.Error", class)
		}
		if back.Class != class {
			t.Errorf("class %q: round-tripped class = %q", class, back.Class)
		}
		if back.Code != code {
			t.Errorf("class %q: round-tripped code = %q, want %q", class, back.Code, code)
		}
		if back.Message != e.Message {
			t.Errorf("class %q: round-tripped message = %q, want %q", class, back.Message, e.Message)
		}
	}
}

func TestFromErrorWrappedChain(t *testing.T) {
	inner := errs.New(errs.ClassForbidden, errs.CodeForbiddenScope, "no access")
	wrapped := fmt.Errorf("handler: %w", inner)
	st := FromError(wrapped)
	if st.Code != CodePermissionDenied {
		t.Fatalf("code = %v, want %v", st.Code, CodePermissionDenied)
	}
	back := ToError(st).(*errs.Error)
	if back.Code != errs.CodeForbiddenScope {
		t.Errorf("code = %q, want %q", back.Code, errs.CodeForbiddenScope)
	}
}

func TestFromErrorPlainError(t *testing.T) {
	st := FromError(errors.New("boom"))
	if st.Code != CodeInternal {
		t.Fatalf("code = %v, want %v", st.Code, CodeInternal)
	}
	back := ToError(st).(*errs.Error)
	if back.Class != errs.ClassInternal {
		t.Errorf("class = %q, want %q", back.Class, errs.ClassInternal)
	}
	if back.Code != "" {
		t.Errorf("code = %q, want empty for a plain error", back.Code)
	}
	if back.Message != "boom" {
		t.Errorf("message = %q, want %q", back.Message, "boom")
	}
}

func TestToErrorNilAndOK(t *testing.T) {
	if err := ToError(nil); err != nil {
		t.Errorf("ToError(nil) = %v, want nil", err)
	}
	if err := ToError(&Status{Code: CodeOK, Message: ""}); err != nil {
		t.Errorf("ToError(OK) = %v, want nil", err)
	}
}

// TestToErrorForeignStatus covers a status from a non-Plugfy peer: no envelope
// separator, so the whole message is the human message and the code is empty,
// but the class is still derived from the gRPC code.
func TestToErrorForeignStatus(t *testing.T) {
	st := &Status{Code: CodeUnavailable, Message: "upstream down"}
	back := ToError(st).(*errs.Error)
	if back.Class != errs.ClassUpstream {
		t.Errorf("class = %q, want %q", back.Class, errs.ClassUpstream)
	}
	if back.Code != "" {
		t.Errorf("code = %q, want empty for a foreign status", back.Code)
	}
	if back.Message != "upstream down" {
		t.Errorf("message = %q, want %q", back.Message, "upstream down")
	}
}

// TestEnvelopeWithSeparatorInMessage guards the decode against a human message
// that itself contains the separator byte: only the first separator splits the
// code from the message, so the message is recovered intact.
func TestEnvelopeWithSeparatorInMessage(t *testing.T) {
	msg := "weird" + codeMessageSep + "message"
	e := errs.New(errs.ClassValidation, errs.CodeValidationSchema, msg)
	back := ToError(FromError(e)).(*errs.Error)
	if back.Code != errs.CodeValidationSchema {
		t.Errorf("code = %q, want %q", back.Code, errs.CodeValidationSchema)
	}
	if back.Message != msg {
		t.Errorf("message = %q, want %q", back.Message, msg)
	}
}
