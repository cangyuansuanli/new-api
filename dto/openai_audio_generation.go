package dto

import (
	"strings"

	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// AudioGenerationRequest is the canonical body for POST /v1/audio/generations.
type AudioGenerationRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	ResponseFormat string `json:"response_format,omitempty"`
	Stream         *bool  `json:"stream,omitempty"`
	Async          *bool  `json:"async,omitempty"`
}

func (r *AudioGenerationRequest) GetTokenCountMeta() *types.TokenCountMeta {
	return &types.TokenCountMeta{
		CombineText: strings.TrimSpace(r.Prompt),
		TokenType:   types.TokenTypeTextNumber,
	}
}

func (r *AudioGenerationRequest) IsStream(c *gin.Context) bool {
	return r.Stream != nil && *r.Stream
}

func (r *AudioGenerationRequest) SetModelName(modelName string) {
	if modelName != "" {
		r.Model = modelName
	}
}

type AudioGenerationResponse struct {
	Created int64             `json:"created"`
	Data    []AudioGenerationData `json:"data"`
}

type AudioGenerationData struct {
	URL string `json:"url"`
}
