// Package grpcstatus is the generic wire helper that maps the platform's
// canonical error model (errs.Class / errs.Code) onto gRPC status codes and
// back, so a service can expose its operations over gRPC without re-deriving a
// per-RPC translation. It is the gRPC analogue of the HTTP rendering the API
// host already performs: the same coarse taxonomy that errs.Class.HTTPStatus
// projects onto HTTP statuses is projected here onto the gRPC status-code
// enumeration, keeping a single source of truth for how a class is observed on
// either transport.
//
// L1 baseplate constraint: plugfy-common is stdlib-only, so this package does
// NOT import google.golang.org/grpc. Instead it names the gRPC status codes as a
// local [Code] enum whose integer values are the canonical, frozen gRPC wire
// numbers (OK=0 … Unauthenticated=16). A service that depends on grpc converts a
// [Code] to a google.golang.org/grpc/codes.Code with a single
// codes.Code(uint32(c)) — they are the same integers by definition — and wraps a
// [Status] into a grpc status.Status. The mapping logic, the taxonomy and the
// stable-code envelope live here, once, as pure data; the transport binding is a
// trivial cast in the one unit that owns a gRPC server.
//
// The mapping is deliberately total and round-trippable at the class level:
// every errs.Class has exactly one canonical [Code] (Class -> Code), and every
// [Code] maps back to exactly one errs.Class (Code -> Class), with the inverse
// chosen so that Class -> Code -> Class is the identity for every defined class.
// Codes that have no dedicated class (e.g. Canceled, Unavailable) fold into the
// nearest class in the taxonomy.
//
// On the wire the stable, machine-readable errs.Code string and the human
// message are preserved verbatim in the [Status] message envelope, so a client
// that receives a status can reconstruct the original *errs.Error — class, code
// and message — rather than collapsing every failure to a bare gRPC code.
package grpcstatus

import (
	"errors"
	"strings"

	"github.com/PlugfyOS/plugfy-common/errs"
)

// Code is a gRPC status code. Its integer values are the canonical, frozen gRPC
// wire numbers, so a consumer that imports google.golang.org/grpc/codes converts
// with codes.Code(uint32(c)) and back with grpcstatus.Code(uint32(code)) — no
// table, no drift. Naming them here keeps the L1 baseplate stdlib-only while
// still letting it own the class<->code taxonomy.
type Code uint32

// The canonical gRPC status codes (https://grpc.io/docs/guides/status-codes/).
// The integer values are fixed by the gRPC wire protocol and MUST NOT change.
const (
	CodeOK                 Code = 0
	CodeCanceled           Code = 1
	CodeUnknown            Code = 2
	CodeInvalidArgument    Code = 3
	CodeDeadlineExceeded   Code = 4
	CodeNotFound           Code = 5
	CodeAlreadyExists      Code = 6
	CodePermissionDenied   Code = 7
	CodeResourceExhausted  Code = 8
	CodeFailedPrecondition Code = 9
	CodeAborted            Code = 10
	CodeOutOfRange         Code = 11
	CodeUnimplemented      Code = 12
	CodeInternal           Code = 13
	CodeUnavailable        Code = 14
	CodeDataLoss           Code = 15
	CodeUnauthenticated    Code = 16
)

// String returns the canonical gRPC code name (matching
// google.golang.org/grpc/codes.Code.String), so logs and the [Status] envelope
// read the same on either side of the wire.
func (c Code) String() string {
	switch c {
	case CodeOK:
		return "OK"
	case CodeCanceled:
		return "Canceled"
	case CodeUnknown:
		return "Unknown"
	case CodeInvalidArgument:
		return "InvalidArgument"
	case CodeDeadlineExceeded:
		return "DeadlineExceeded"
	case CodeNotFound:
		return "NotFound"
	case CodeAlreadyExists:
		return "AlreadyExists"
	case CodePermissionDenied:
		return "PermissionDenied"
	case CodeResourceExhausted:
		return "ResourceExhausted"
	case CodeFailedPrecondition:
		return "FailedPrecondition"
	case CodeAborted:
		return "Aborted"
	case CodeOutOfRange:
		return "OutOfRange"
	case CodeUnimplemented:
		return "Unimplemented"
	case CodeInternal:
		return "Internal"
	case CodeUnavailable:
		return "Unavailable"
	case CodeDataLoss:
		return "DataLoss"
	case CodeUnauthenticated:
		return "Unauthenticated"
	default:
		return "Code(" + itoa(uint32(c)) + ")"
	}
}

// CodeFor returns the canonical gRPC status code for an error class. It mirrors
// errs.Class.HTTPStatus: the same coarse taxonomy that drives HTTP statuses
// drives gRPC codes, so a class is observed consistently on either transport. An
// unknown class folds to CodeInternal, exactly as it folds to HTTP 500.
func CodeFor(class errs.Class) Code {
	switch class {
	case errs.ClassValidation:
		return CodeInvalidArgument
	case errs.ClassUnauthorized:
		return CodeUnauthenticated
	case errs.ClassForbidden:
		return CodePermissionDenied
	case errs.ClassNotFound:
		return CodeNotFound
	case errs.ClassConflict:
		return CodeAlreadyExists
	case errs.ClassRateLimit:
		return CodeResourceExhausted
	case errs.ClassUpstream:
		return CodeUnavailable
	case errs.ClassTimeout:
		return CodeDeadlineExceeded
	case errs.ClassInternal:
		return CodeInternal
	default:
		return CodeInternal
	}
}

// ClassFor returns the canonical error class for a gRPC status code. It is the
// inverse of CodeFor on the codes CodeFor can produce, so ClassFor(CodeFor(c))
// == c for every defined class; codes outside that range fold to the nearest
// class in the taxonomy (e.g. Canceled -> timeout, Unavailable -> upstream,
// Unimplemented/Unknown -> internal), so no status escapes the model.
func ClassFor(code Code) errs.Class {
	switch code {
	case CodeOK:
		// OK is not an error; callers should not classify a success, but a
		// defensive default keeps the function total.
		return errs.ClassInternal
	case CodeInvalidArgument, CodeOutOfRange, CodeFailedPrecondition:
		return errs.ClassValidation
	case CodeUnauthenticated:
		return errs.ClassUnauthorized
	case CodePermissionDenied:
		return errs.ClassForbidden
	case CodeNotFound:
		return errs.ClassNotFound
	case CodeAlreadyExists, CodeAborted:
		return errs.ClassConflict
	case CodeResourceExhausted:
		return errs.ClassRateLimit
	case CodeUnavailable:
		return errs.ClassUpstream
	case CodeDeadlineExceeded, CodeCanceled:
		return errs.ClassTimeout
	case CodeInternal, CodeUnknown, CodeUnimplemented, CodeDataLoss:
		return errs.ClassInternal
	default:
		return errs.ClassInternal
	}
}

// Status is the transport-agnostic projection of an error onto the gRPC status
// model: a [Code] plus the message envelope that carries the stable errs.Code
// and the human message. The owning service binds it to a concrete
// google.golang.org/grpc status.Status with
// status.New(codes.Code(uint32(s.Code)), s.Message). It is the pure-data shape
// FromError emits and ToError consumes.
type Status struct {
	// Code is the canonical gRPC status code for the error's class.
	Code Code
	// Message is the wire envelope "<errs.Code>\x1f<message>" (or just the
	// message when the errs.Code is empty), so a peer can recover the stable
	// code and human text via ToError.
	Message string
}

// codeMessageSep separates the embedded stable errs.Code from the human-readable
// message in the status message envelope. It is a control character that never
// appears in a code (lower_snake_case, reverse-DNS-flavored) or a normal English
// message, so splitting on it is unambiguous. The wire shape is
// "<errs.Code>\x1f<message>" when a code is present, or just "<message>" when it
// is empty.
const codeMessageSep = "\x1f"

// FromError converts an arbitrary error into a [Status] carrying the canonical
// gRPC code for the error's class and an envelope that preserves the stable
// errs.Code and human message, so ToError can reconstruct the original
// *errs.Error on the far side. A nil error yields nil (the gRPC OK convention).
// Non-*errs.Error values are treated as ClassInternal with no code, matching
// errs.Classify.
func FromError(err error) *Status {
	if err == nil {
		return nil
	}
	var e *errs.Error
	if errors.As(err, &e) && e != nil {
		return &Status{Code: CodeFor(e.Class), Message: encode(e.Code, e.Message)}
	}
	return &Status{Code: CodeInternal, Message: encode("", err.Error())}
}

// ToError converts a [Status] back into a classified *errs.Error: the class is
// derived from the status code, and the stable errs.Code and message are
// recovered from the envelope FromError produced. A nil status or CodeOK yields a
// nil error (success is not an error). A status produced by a non-Plugfy peer (no
// envelope separator) is still classified by its code, with the whole message
// used as the human message and an empty errs.Code.
func ToError(st *Status) error {
	if st == nil || st.Code == CodeOK {
		return nil
	}
	code, msg := decode(st.Message)
	return errs.New(ClassFor(st.Code), code, msg)
}

// encode packs a stable code and a message into the status message envelope.
func encode(code, message string) string {
	if code == "" {
		return message
	}
	return code + codeMessageSep + message
}

// decode splits a status message envelope back into its stable code and message.
// A message without the separator is returned as-is with an empty code. Only the
// first separator splits, so a human message that itself contains the separator
// byte is recovered intact.
func decode(envelope string) (code, message string) {
	if i := strings.Index(envelope, codeMessageSep); i >= 0 {
		return envelope[:i], envelope[i+len(codeMessageSep):]
	}
	return "", envelope
}

// itoa renders a uint32 without importing strconv, keeping String allocation-
// light for the unreachable-default case.
func itoa(v uint32) string {
	if v == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
