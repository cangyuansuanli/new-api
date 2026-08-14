package seedancehuabu

import (
	"strings"
	"sync"
)

// pendingDirectRehostURL bridges ParseTaskResult and ResolveTaskResultSource within
// one polling round: when upstream returns result_url, rehost must keep that direct
// CDN URL instead of the generic /content override (Douyin Referer ACL on redirect).
var pendingDirectRehostURL sync.Map

func noteDirectRehostURL(upstreamTaskID, directURL string) {
	upstreamTaskID = strings.TrimSpace(upstreamTaskID)
	directURL = strings.TrimSpace(directURL)
	if upstreamTaskID == "" {
		return
	}
	if directURL == "" {
		pendingDirectRehostURL.Delete(upstreamTaskID)
		return
	}
	pendingDirectRehostURL.Store(upstreamTaskID, directURL)
}

func consumeDirectRehostURL(upstreamTaskID string) (directURL string, ok bool) {
	upstreamTaskID = strings.TrimSpace(upstreamTaskID)
	if upstreamTaskID == "" {
		return "", false
	}
	value, loaded := pendingDirectRehostURL.LoadAndDelete(upstreamTaskID)
	if !loaded {
		return "", false
	}
	directURL, ok = value.(string)
	if !ok || strings.TrimSpace(directURL) == "" {
		return "", false
	}
	return directURL, true
}
