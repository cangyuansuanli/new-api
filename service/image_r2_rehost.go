package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

// ImageModelUsesURLRehost：上游回 url 时需转存 R2，避免暴露渠道地址。
// - 4K 档位（别名后缀 -4k）
// - Geek2API FLUX 系列（flux-pro-2 等，前缀 flux-）
func ImageModelUsesURLRehost(originModel string) bool {
	name := strings.ToLower(strings.TrimSpace(originModel))
	return strings.HasSuffix(name, "-4k") || strings.HasPrefix(name, "flux-")
}

// imageAsyncAcceptsGulieStyleURL：cy-img1-/gulie- 及 public gpt-image-2 系列；上游回 url 时异步转存 R2。
func imageAsyncAcceptsGulieStyleURL(originModel string) bool {
	name := strings.ToLower(strings.TrimSpace(originModel))
	if strings.HasPrefix(name, "cy-img1-") || strings.HasPrefix(name, "gulie-") {
		return true
	}
	return name == "gpt-image-2" || strings.HasPrefix(name, "gpt-image-2-")
}

// ImageSyncPreferUpstreamB64JSON：同步生图对客户要 url 时，对内改请求上游 b64_json，避免 loopback URL 二次下载挂住。
func ImageSyncPreferUpstreamB64JSON(originModel string) bool {
	return imageAsyncAcceptsGulieStyleURL(originModel)
}

// ImageAsyncAcceptsUpstreamURL：同步/异步生图允许上游回 url（如 Gulie loopback、4K），转存 R2 后返回。
func ImageAsyncAcceptsUpstreamURL(originModel string) bool {
	if ImageModelUsesURLRehost(originModel) {
		return true
	}
	return imageAsyncAcceptsGulieStyleURL(originModel)
}

// RewriteLoopbackUpstreamImageURL 将上游 loopback 图片地址（如 Gulie 127.0.0.1:3001）
// 映射为渠道主机名 + 原端口，便于服务端下载。
func RewriteLoopbackUpstreamImageURL(channelBaseURL, imageURL string) string {
	channelBaseURL = strings.TrimSpace(channelBaseURL)
	if channelBaseURL == "" {
		return imageURL
	}
	img, err := url.Parse(imageURL)
	if err != nil {
		return imageURL
	}
	host := strings.ToLower(img.Hostname())
	if host != "127.0.0.1" && host != "localhost" {
		return imageURL
	}
	base, err := url.Parse(channelBaseURL)
	if err != nil || base.Hostname() == "" {
		return imageURL
	}
	port := img.Port()
	if port == "" {
		if img.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	out := &url.URL{
		Scheme:   base.Scheme,
		Host:     net.JoinHostPort(base.Hostname(), port),
		Path:     img.Path,
		RawQuery: img.RawQuery,
	}
	return out.String()
}

func imageDataNeedsURLRehost(images []dto.ImageData) bool {
	for _, item := range images {
		if strings.TrimSpace(item.Url) != "" && strings.TrimSpace(item.B64Json) == "" {
			return true
		}
	}
	return false
}

func imageDataHasB64(images []dto.ImageData) bool {
	for _, item := range images {
		if strings.TrimSpace(item.B64Json) != "" {
			return true
		}
	}
	return false
}

func decodeImageB64Payload(b64 string) ([]byte, string, error) {
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return nil, "", err
	}
	mimeType := http.DetectContentType(data)
	if !strings.HasPrefix(mimeType, "image/") {
		mimeType = "image/png"
	}
	return data, mimeType, nil
}

func syncImageNeedsRehost(originModel string, images []dto.ImageData, clientWantsURL bool) bool {
	if !ImageAsyncAcceptsUpstreamURL(originModel) || len(images) == 0 {
		return false
	}
	if imageDataNeedsURLRehost(images) {
		return true
	}
	return clientWantsURL && imageDataHasB64(images)
}

// RehostImageDataForClient 将上游 b64 或（4K/FLUX）url 转存 R2；clientWantsURL 时清除 b64_json 仅留公网 url。
func RehostImageDataForClient(ctx context.Context, userID int, storeID, channelBaseURL, originModel string, images []dto.ImageData, clientWantsURL bool) ([]dto.ImageData, error) {
	if !syncImageNeedsRehost(originModel, images, clientWantsURL) {
		return images, nil
	}
	if getR2Config() == nil {
		return nil, fmt.Errorf("R2 not configured")
	}
	storeID = strings.TrimSpace(storeID)
	if storeID == "" {
		storeID = model.GenerateTaskID()
	}
	out := make([]dto.ImageData, len(images))
	copy(out, images)
	for index := range out {
		item := &out[index]
		if b64 := strings.TrimSpace(item.B64Json); b64 != "" {
			data, mimeType, err := decodeImageB64Payload(b64)
			if err != nil {
				return nil, err
			}
			uploaded, err := UploadGeneratedImageBytes(ctx, userID, storeID, index, data, mimeType)
			if err != nil {
				return nil, fmt.Errorf("rehost upstream image b64: %w", err)
			}
			item.Url = uploaded.PublicURL
			if clientWantsURL {
				item.B64Json = ""
			}
			continue
		}
		if strings.TrimSpace(item.Url) == "" {
			continue
		}
		if !ImageModelUsesURLRehost(originModel) {
			continue
		}
		downloadURL := RewriteLoopbackUpstreamImageURL(channelBaseURL, item.Url)
		uploaded, err := UploadGeneratedImageFromURL(ctx, userID, storeID, index, downloadURL)
		if err != nil {
			return nil, fmt.Errorf("rehost upstream image url: %w", err)
		}
		item.Url = uploaded.PublicURL
	}
	return out, nil
}

// RehostImageDataURLs 将需转存的模型上游 url 落 R2；未命中或无 url 时原样返回。
func RehostImageDataURLs(ctx context.Context, userID int, storeID, channelBaseURL, originModel string, images []dto.ImageData) ([]dto.ImageData, error) {
	return RehostImageDataForClient(ctx, userID, storeID, channelBaseURL, originModel, images, false)
}

// RehostSyncImageResponseBody 同步生图 JSON 响应：b64 或 url 转存 R2 后返回公网 URL。
func RehostSyncImageResponseBody(ctx context.Context, userID int, originModel, channelBaseURL string, responseBody []byte, clientWantsURL bool) ([]byte, error) {
	if len(responseBody) == 0 {
		return responseBody, nil
	}
	var payload struct {
		Data []dto.ImageData `json:"data"`
	}
	if err := common.Unmarshal(responseBody, &payload); err != nil || len(payload.Data) == 0 {
		return responseBody, nil
	}
	if !syncImageNeedsRehost(originModel, payload.Data, clientWantsURL) {
		return responseBody, nil
	}
	rehosted, err := RehostImageDataForClient(ctx, userID, model.GenerateTaskID(), channelBaseURL, originModel, payload.Data, clientWantsURL)
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := common.Unmarshal(responseBody, &raw); err != nil {
		return responseBody, nil
	}
	dataJSON, err := common.Marshal(rehosted)
	if err != nil {
		return nil, err
	}
	raw["data"] = dataJSON
	return common.Marshal(raw)
}
