package clienterror

// ErrorContext carries stable error metadata when the caller has it. Raw-only
// rules read Raw; channel rules may additionally use provider error codes.
type ErrorContext struct {
	Model      string
	Raw        string
	Payload    []byte
	ErrorCode  string
	ErrorType  string
	HTTPStatus int
}

// Normalizer maps a preprocessed upstream error to client-facing copy.
// Return ok=false to defer to the next registered rule.
type Normalizer func(preferChinese bool, failure ErrorContext) (msg string, ok bool)
type RawNormalizer func(preferChinese bool, raw string) (msg string, ok bool)

var normalizers []Normalizer

// Register appends a vendor/domain normalizer. Registration order is defined in normalize.go init().
func Register(fn Normalizer) {
	normalizers = append(normalizers, fn)
}

// RegisterRaw adapts existing message-only rules to the structured pipeline.
func RegisterRaw(fn RawNormalizer) {
	Register(func(preferChinese bool, failure ErrorContext) (string, bool) {
		return fn(preferChinese, failure.Raw)
	})
}

func runNormalizers(preferChinese bool, failure ErrorContext) (string, bool) {
	for _, fn := range normalizers {
		if msg, ok := fn(preferChinese, failure); ok {
			return msg, true
		}
	}
	return "", false
}
