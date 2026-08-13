package audiovendor

import (
	"strings"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

type RequestPatchResult struct{}

type PatchRequestFunc func(originModel string, request *dto.AudioGenerationRequest) (RequestPatchResult, error)

type MatchRelayFunc func(info *relaycommon.RelayInfo) bool

type ValidateRelayRequestFunc func(c *gin.Context, info *relaycommon.RelayInfo, request *dto.AudioGenerationRequest) error

type PatchRelayRequestFunc func(info *relaycommon.RelayInfo, request *dto.AudioGenerationRequest) (RequestPatchResult, error)

type Descriptor struct {
	Name              string
	Match             func(originModel string) bool
	MatchRelay        MatchRelayFunc
	ValidateRequest   ValidateRelayRequestFunc
	PatchRequest      PatchRequestFunc
	PatchRelayRequest PatchRelayRequestFunc
}

var descriptors []Descriptor

func register(d Descriptor) {
	descriptors = append(descriptors, d)
}

func ValidateRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.AudioGenerationRequest) error {
	for _, d := range descriptors {
		if d.ValidateRequest == nil || !descriptorMatchesRelay(d, info) {
			continue
		}
		return d.ValidateRequest(c, info, request)
	}
	return nil
}

func ApplyRequestPatch(info *relaycommon.RelayInfo, request *dto.AudioGenerationRequest) (RequestPatchResult, error) {
	for _, d := range descriptors {
		if d.PatchRelayRequest == nil || !descriptorMatchesRelay(d, info) {
			continue
		}
		return d.PatchRelayRequest(info, request)
	}
	originModel := ""
	if info != nil {
		originModel = info.OriginModelName
	}
	for _, d := range descriptors {
		if d.PatchRequest == nil || !d.Match(originModel) {
			continue
		}
		return d.PatchRequest(originModel, request)
	}
	return RequestPatchResult{}, nil
}

func descriptorMatchesRelay(d Descriptor, info *relaycommon.RelayInfo) bool {
	if info == nil {
		return false
	}
	if d.MatchRelay != nil {
		return d.MatchRelay(info)
	}
	return d.Match(info.OriginModelName)
}

func IsGeminiMusicOriginModel(originModel string) bool {
	name := strings.ToLower(strings.TrimSpace(originModel))
	if name == "gemini-music" {
		return true
	}
	return strings.HasSuffix(name, "-gemini-music") || strings.Contains(name, "gemini-music")
}
