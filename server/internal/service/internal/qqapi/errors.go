package qqapi

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ErrorKind string

const (
	Throttle            ErrorKind = "throttle"
	InvalidToken        ErrorKind = "invalid_token"
	AuthBlocked         ErrorKind = "auth_blocked"
	Forbidden           ErrorKind = "forbidden"
	GroupNotFound       ErrorKind = "group_not_found"
	PushRejected        ErrorKind = "push_rejected"
	InvalidContent      ErrorKind = "invalid_content"
	TemporaryNotApplied ErrorKind = "temporary_not_applied"
	Ambiguous           ErrorKind = "ambiguous"
)

// OperationError contains only classified, bounded metadata. It does not expose
// remote error messages, HTTP request/response bodies, URLs, or I/O errors.
// HTTPStatus or Code is zero when not available. RetryAfter is relative to receipt.
type OperationError struct {
	Kind       ErrorKind
	HTTPStatus int
	Code       int
	TraceID    string
	RetryAfter time.Duration
}

func (e *OperationError) Error() string {
	return fmt.Sprintf("qqapi: %s (HTTP %d, code %d)", e.Kind, e.HTTPStatus, e.Code)
}

type operation uint8

const (
	refreshOp operation = iota
	readOp
	sendOp
	decisionOp
)

func (op operation) unconfirmed() ErrorKind {
	if op == sendOp || op == decisionOp {
		return Ambiguous
	}
	return TemporaryNotApplied
}

type apiResponse struct {
	body        []byte
	status      int
	code        int
	traceID     string
	retryAfter  time.Duration
	valid       bool
	decisionAck bool
}

func (r *apiResponse) parse(op operation) {
	if r.status == http.StatusNoContent && len(r.body) == 0 && op == decisionOp {
		r.valid = true
		r.decisionAck = true
		return
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(r.body, &fields) != nil || fields == nil {
		return
	}
	if r.traceID == "" {
		var traceID string
		if json.Unmarshal(fields["trace_id"], &traceID) == nil {
			r.traceID = validTraceID(traceID)
		}
	}
	code, hasCode := responseCode(fields, "code")
	errCode, hasErrCode := responseCode(fields, "err_code")
	if !hasCode || !hasErrCode || (code != 0 && errCode != 0 && code != errCode) {
		return
	}
	if errCode != 0 {
		code = errCode
	}
	r.code = code
	r.valid = true
	if op == decisionOp && r.status == http.StatusOK {
		// Only the documented empty acknowledgement (optionally with generic
		// envelope metadata) confirms a 200 response to a write.
		r.decisionAck = true
		for key := range fields {
			switch key {
			case "code", "err_code", "trace_id":
			default:
				r.decisionAck = false
			}
		}
	}
}

func responseCode(fields map[string]json.RawMessage, key string) (int, bool) {
	raw, exists := fields[key]
	if !exists {
		return 0, true
	}
	code, err := strconv.ParseInt(string(raw), 10, 32)
	return int(code), err == nil && code >= 0
}

func (r apiResponse) classify(op operation) ErrorKind {
	if r.status == http.StatusTooManyRequests {
		return Throttle
	}
	if r.valid && r.code != 0 {
		if op == sendOp && (r.status == http.StatusCreated || r.status == http.StatusAccepted) && (r.code == 304023 || r.code == 304024) {
			return ""
		}
		if kind := codeKind(op, r.code); kind != "" {
			return kind
		}
	}
	switch r.status {
	case http.StatusUnauthorized:
		if op == refreshOp {
			return AuthBlocked
		}
		return InvalidToken
	case http.StatusForbidden, http.StatusNotFound, http.StatusMethodNotAllowed:
		return Forbidden // Bare 404 means API not found, not proof the bot left its group.
	}
	if r.valid && r.code == 0 && (r.status == http.StatusOK || r.status == http.StatusNoContent) {
		return ""
	}
	return op.unconfirmed()
}

// Only documented codes that establish non-application allow a write to be
// classified as safe to retry. Generic downstream 5xx/send failures do not.
func codeKind(op operation, code int) ErrorKind {
	if op == refreshOp {
		switch code {
		case 100001:
			return Throttle
		case 100004, 100007, 100016, 10004:
			return AuthBlocked
		}
	}
	switch code {
	case 20028, 304045, 1100100, 40034100:
		return Throttle
	case 11241, 11243:
		return InvalidToken
	case 11251, 11254, 11261, 11262, 11265, 11275:
		return AuthBlocked
	case 11253, 11264, 11274, 11282, 40054002:
		return Forbidden
	case 1100104, 40034101, 40054003:
		return GroupNotFound
	case 304022, 304046, 304048, 40034105:
		return PushRejected
	case 12002, 22006, 50006, 304003, 304061, 1100101, 40034006, 40054007, 40054010:
		return InvalidContent
	case 11242, 11252, 11263, 11281:
		return TemporaryNotApplied
	default:
		return ""
	}
}

func (r apiResponse) failure(kind ErrorKind) *OperationError {
	return &OperationError{Kind: kind, HTTPStatus: r.status, Code: r.code, TraceID: r.traceID, RetryAfter: r.retryAfter}
}

const maxDurationSeconds = math.MaxInt64 / int64(time.Second)

func retryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if n, ok := decimalInt([]byte(value)); ok && n <= maxDurationSeconds {
		return time.Duration(n) * time.Second
	}
	if date, err := http.ParseTime(value); err == nil && date.After(now) {
		return date.Sub(now)
	}
	return 0
}

func decimalInt(raw []byte) (int64, bool) {
	if len(raw) == 0 || len(raw) > 19 {
		return 0, false
	}
	for _, b := range raw {
		if b < '0' || b > '9' {
			return 0, false
		}
	}
	n, err := strconv.ParseInt(string(raw), 10, 64)
	return n, err == nil
}

func validTraceID(value string) string {
	if len(value) == 0 || len(value) > 128 {
		return ""
	}
	for i := 0; i < len(value); i++ {
		b := value[i]
		if b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '-' || b == '_' || b == '.' {
			continue
		}
		return ""
	}
	return value
}
