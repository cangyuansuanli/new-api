package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
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

// ImageAsyncAcceptsUpstreamURL：异步 worker 落库时允许上游回 url（如 Gulie loopback、4K），转存 R2 后返回。
func ImageAsyncAcceptsUpstreamURL(originModel string) bool {
	if ImageModelUsesURLRehost(originModel) {
		return true
	}
	name := strings.ToLower(strings.TrimSpace(originModel))
	return strings.HasPrefix(name, "cy-img1-") || strings.HasPrefix(name, "gulie-")
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

// RehostImageDataURLs 将需转存的模型上游 url 落 R2；未命中或无 url 时原样返回。
func RehostImageDataURLs(ctx context.Context, userID int, storeID, channelBaseURL, originModel string, images []dto.ImageData) ([]dto.ImageData, error) {
	if !ImageModelUsesURLRehost(originModel) || len(images) == 0 || !imageDataNeedsURLRehost(images) {
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
		if strings.TrimSpace(item.Url) == "" || strings.TrimSpace(item.B64Json) != "" {
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

// RehostSyncImageResponseBody 同步生图 JSON 响应：命中模型且 data[].url 时替换为 R2 公网 URL。
func RehostSyncImageResponseBody(ctx context.Context, userID int, originModel, channelBaseURL string, responseBody []byte) ([]byte, error) {
	if !ImageModelUsesURLRehost(originModel) || len(responseBody) == 0 {
		return responseBody, nil
	}
	var payload struct {
		Data []dto.ImageData `json:"data"`
	}
	if err := common.Unmarshal(responseBody, &payload); err != nil || len(payload.Data) == 0 {
		return responseBody, nil
	}
	if !imageDataNeedsURLRehost(payload.Data) {
		return responseBody, nil
	}
	rehosted, err := RehostImageDataURLs(ctx, userID, model.GenerateTaskID(), channelBaseURL, originModel, payload.Data)
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
