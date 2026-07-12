package service

// 通用 R2 图片转存执行（上传/下载）；转存策略见 relay/imagevendor。

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/imagevendor"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

const defaultImageRehostTimeout = 5 * time.Minute

// RehostDetachedContext 在客户端断开连接后仍允许 R2 转存完成（不继承 cancel）。
func RehostDetachedContext(parent context.Context) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	timeout := defaultImageRehostTimeout
	if common.RelayTimeout > 0 {
		timeout = time.Duration(common.RelayTimeout) * time.Second
	}
	ctx, _ := context.WithTimeout(context.WithoutCancel(parent), timeout)
	return ctx
}

// IsBillableImageRehostClientCancel 上游已返回图片，但转存阶段因客户端取消/断开失败——仍应向用户计费。
func IsBillableImageRehostClientCancel(err error) bool {
	if err == nil {
		return false
	}
	hasRehost := false
	for current := err; current != nil; current = errors.Unwrap(current) {
		if strings.Contains(current.Error(), "rehost upstream image") {
			hasRehost = true
			break
		}
	}
	if !hasRehost {
		return false
	}
	return errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled")
}

// ImageRehostAPIError 同步生图转存失败时，若属于客户取消且上游已成功则保留 usage 供后续扣费。
func ImageRehostAPIError(usage *dto.Usage, err error) (*dto.Usage, *types.NewAPIError) {
	if err == nil {
		return usage, nil
	}
	apiErr := types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusBadGateway)
	if usage != nil && IsBillableImageRehostClientCancel(err) {
		return usage, apiErr
	}
	return nil, apiErr
}

// ClientDisconnected 客户端已取消或断开 HTTP 连接。
func ClientDisconnected(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	return c.Request.Context().Err() != nil
}

// ImageRehostDeliveredClientCancelErr 转存已完成但客户端已断开，仍应扣费并在日志中保留链接。
func ImageRehostDeliveredClientCancelErr(c *gin.Context) error {
	if !ClientDisconnected(c) {
		return nil
	}
	return fmt.Errorf("rehost upstream image delivered: %w", context.Canceled)
}

// CollectRehostedImageURLs 从转存后的 ImageData 提取 R2 公网 URL。
func CollectRehostedImageURLs(images []dto.ImageData) []string {
	urls := make([]string, 0, len(images))
	for _, item := range images {
		u := strings.TrimSpace(item.Url)
		if u == "" || !strings.HasPrefix(u, "http") {
			continue
		}
		urls = append(urls, u)
	}
	return urls
}

// ExtractRehostedImageURLsFromJSON 从 OpenAI 生图 JSON 响应中提取 R2 公网 URL。
func ExtractRehostedImageURLsFromJSON(body []byte) []string {
	if len(body) == 0 {
		return nil
	}
	var payload struct {
		Data []dto.ImageData `json:"data"`
	}
	if err := common.Unmarshal(body, &payload); err != nil {
		return nil
	}
	return CollectRehostedImageURLs(payload.Data)
}

// ImageRehostLogContent 生成 consume 日志中的图片链接条目。
func ImageRehostLogContent(urls []string) []string {
	if len(urls) == 0 {
		return nil
	}
	lines := make([]string, 0, len(urls))
	for i, u := range urls {
		if len(urls) == 1 {
			lines = append(lines, "图片链接 "+u)
		} else {
			lines = append(lines, fmt.Sprintf("图片链接 %d %s", i+1, u))
		}
	}
	return lines
}

// RecordRehostedImageURLs 记录转存后的公网链接，供 consume 日志输出。
func RecordRehostedImageURLs(info *relaycommon.RelayInfo, images []dto.ImageData) {
	if info == nil {
		return
	}
	info.RehostedImageURLs = CollectRehostedImageURLs(images)
}

// RecordRehostedImageURLsFromJSON 从 OpenAI 生图 JSON 响应记录 R2 公网链接。
func RecordRehostedImageURLsFromJSON(info *relaycommon.RelayInfo, body []byte) {
	if info == nil {
		return
	}
	info.RehostedImageURLs = ExtractRehostedImageURLsFromJSON(body)
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

// RewriteGeneratedUpstreamURLToChannelBase maps self-hosted /generated/* assets
// back to the configured channel base. Self-hosted image relays may return
// an external hostname while the real reachable upstream is the channel base URL.
func RewriteGeneratedUpstreamURLToChannelBase(channelBaseURL, imageURL string) (string, bool) {
	channelBaseURL = strings.TrimSpace(channelBaseURL)
	imageURL = strings.TrimSpace(imageURL)
	if channelBaseURL == "" || imageURL == "" {
		return imageURL, false
	}
	img, err := url.Parse(imageURL)
	if err != nil || img.Scheme == "" || img.Host == "" {
		return imageURL, false
	}
	if img.Scheme != "http" && img.Scheme != "https" {
		return imageURL, false
	}
	if !strings.HasPrefix(img.Path, "/generated/") {
		return imageURL, false
	}
	base, err := url.Parse(channelBaseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return imageURL, false
	}
	if base.Scheme != "http" && base.Scheme != "https" {
		return imageURL, false
	}
	if !generatedURLRewriteAllowed(base, img) {
		return imageURL, false
	}
	rewritten := *img
	rewritten.Scheme = base.Scheme
	rewritten.Host = base.Host
	out := rewritten.String()
	return out, true
}

func generatedURLRewriteAllowed(base, img *url.URL) bool {
	if base == nil || img == nil {
		return false
	}
	if base.Port() == "6001" {
		return true
	}
	return strings.Contains(strings.ToLower(base.Hostname()), "adobe")
}

func rewriteGeneratedImageDataURLsToChannelBase(channelBaseURL string, images []dto.ImageData) ([]dto.ImageData, bool) {
	if len(images) == 0 {
		return images, false
	}
	out := make([]dto.ImageData, len(images))
	copy(out, images)
	changed := false
	for index := range out {
		rewritten, ok := RewriteGeneratedUpstreamURLToChannelBase(channelBaseURL, out[index].Url)
		if !ok {
			continue
		}
		if rewritten != out[index].Url {
			out[index].Url = rewritten
			changed = true
		}
	}
	return out, changed
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
	policy := imagevendor.ResolveRehostPolicy(originModel)
	if !policy.AcceptUpstreamURL || len(images) == 0 {
		return false
	}
	if imageDataNeedsURLRehost(images) {
		return true
	}
	return clientWantsURL && imageDataHasB64(images)
}

// RehostImageDataForClient 将上游 b64 或（4K/FLUX）url 转存 R2；clientWantsURL 时清除 b64_json 仅留公网 url。
func RehostImageDataForClient(ctx context.Context, userID int, storeID, channelBaseURL, originModel string, images []dto.ImageData, clientWantsURL bool) ([]dto.ImageData, error) {
	images, _ = rewriteGeneratedImageDataURLsToChannelBase(channelBaseURL, images)
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
		if !imagevendor.ResolveRehostPolicy(originModel).AcceptUpstreamURL {
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
	rehosted, err := RehostImageDataForClient(ctx, userID, model.GenerateTaskID(), channelBaseURL, originModel, payload.Data, clientWantsURL)
	if err != nil {
		return nil, err
	}
	if reflect.DeepEqual(rehosted, payload.Data) {
		return responseBody, nil
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

// DecodeImageDataItem 解析 ImageData：b64_json、data URI（url 字段）或上游 http(s) url。
// 有字节数据时返回 (data, mimeType, nil)；仅有 URL 时返回 (nil, url, nil)。
func DecodeImageDataItem(item dto.ImageData) ([]byte, string, error) {
	if item.B64Json != "" {
		data, err := base64.StdEncoding.DecodeString(item.B64Json)
		if err != nil {
			return nil, "", err
		}
		return data, detectImageBytesMimeType(data), nil
	}
	if item.Url == "" {
		return nil, "", fmt.Errorf("image item has no url or b64_json")
	}
	if strings.HasPrefix(item.Url, "data:") {
		data, mime, err := decodeDataURI(item.Url)
		if err != nil {
			return nil, "", err
		}
		return data, mime, nil
	}
	return nil, item.Url, nil
}

func decodeDataURI(uri string) ([]byte, string, error) {
	comma := strings.Index(uri, ",")
	if comma < 0 {
		return nil, "", fmt.Errorf("invalid data uri")
	}
	meta := uri[5:comma]
	payload := uri[comma+1:]
	mimeType := "image/png"
	if semi := strings.Index(meta, ";"); semi > 0 {
		mimeType = meta[:semi]
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, "", err
	}
	if !strings.HasPrefix(mimeType, "image/") {
		mimeType = detectImageBytesMimeType(data)
	}
	return data, mimeType, nil
}

func detectImageBytesMimeType(data []byte) string {
	if len(data) == 0 {
		return "image/png"
	}
	mimeType := http.DetectContentType(data)
	if strings.HasPrefix(mimeType, "image/") {
		return mimeType
	}
	return "image/png"
}

// RehostTaskImageResultURLs 异步 task 专用：转存后返回公网 URL 列表。
// 支持 b64_json、data URI（url 字段）、上游 http(s) url（受 policy 约束）。
func RehostTaskImageResultURLs(ctx context.Context, userID int, storeID, channelBaseURL, originModel string, images []dto.ImageData) ([]string, error) {
	acceptUpstreamURL := imagevendor.ImageAsyncAcceptsUpstreamURL(originModel)
	resultURLs := make([]string, 0, len(images))
	storeID = strings.TrimSpace(storeID)
	for index, item := range images {
		data, mimeOrURL, err := DecodeImageDataItem(item)
		if err != nil {
			return nil, err
		}
		if len(data) > 0 {
			if getR2Config() == nil {
				return nil, fmt.Errorf("R2 not configured")
			}
			mimeType := mimeOrURL
			if !strings.HasPrefix(mimeType, "image/") {
				mimeType = "image/png"
			}
			uploaded, err := UploadGeneratedImageBytes(ctx, userID, storeID, index, data, mimeType)
			if err != nil {
				return nil, err
			}
			resultURLs = append(resultURLs, uploaded.PublicURL)
			continue
		}
		if mimeOrURL != "" {
			if rewritten, ok := RewriteGeneratedUpstreamURLToChannelBase(channelBaseURL, mimeOrURL); ok {
				mimeOrURL = rewritten
			}
			if acceptUpstreamURL {
				if getR2Config() == nil {
					return nil, fmt.Errorf("R2 not configured")
				}
				downloadURL := RewriteLoopbackUpstreamImageURL(channelBaseURL, mimeOrURL)
				uploaded, err := UploadGeneratedImageFromURL(ctx, userID, storeID, index, downloadURL)
				if err != nil {
					return nil, fmt.Errorf("rehost upstream image url: %w", err)
				}
				resultURLs = append(resultURLs, uploaded.PublicURL)
				continue
			}
			return nil, fmt.Errorf("upstream returned url without b64_json; use response_format=b64_json")
		}
	}
	if len(resultURLs) == 0 {
		return nil, fmt.Errorf("no image results from upstream")
	}
	return resultURLs, nil
}
