package audio

import "github.com/QuantumNous/new-api/relay/channel"

var getAdaptor func(apiType int) channel.Adaptor

// SetGetAdaptor 由 relay 包 init 注入，避免 audio ↔ relay 循环 import。
func SetGetAdaptor(fn func(apiType int) channel.Adaptor) {
	getAdaptor = fn
}
