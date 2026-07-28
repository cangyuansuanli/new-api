package clienterror

import "net/http"

func normalizeTransport(preferChinese bool, failure ErrorContext) (string, bool) {
	if failure.HTTPStatus == http.StatusTooManyRequests {
		return localized(preferChinese, UpstreamSaturatedMessageZH, UpstreamSaturatedMessageEN), true
	}
	return "", false
}
