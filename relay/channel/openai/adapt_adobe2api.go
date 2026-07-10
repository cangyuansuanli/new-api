package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
)

var adobe2APIImageModelPrefixes = []string{
	"nano-banana",
	"gpt-image",
	"adobe-nano-banana",
	"adobe-gpt-image",
	"adobe2api-nano-banana",
	"adobe2api-gpt-image",
	"firefly-nano-banana",
	"firefly-gpt-image",
}

var adobe2APIVideoModelPrefixes = []string{
	"sora2",
	"veo31",
	"kling3",
	"kling-o3",
	"seedance2",
	"adobe-sora2",
	"adobe-veo31",
	"adobe-kling3",
	"adobe-kling-o3",
	"adobe-seedance2",
	"adobe2api-sora2",
	"adobe2api-veo31",
	"adobe2api-kling3",
	"adobe2api-kling-o3",
	"adobe2api-seedance2",
	"firefly-sora",
	"firefly-veo",
	"firefly-kling",
	"firefly-seedance",
}

func IsAdobe2APIImageOriginModel(model string) bool {
	return hasAdobe2APIPrefix(model, adobe2APIImageModelPrefixes)
}

func IsAdobe2APIVideoChatOriginModel(model string) bool {
	return hasAdobe2APIPrefix(model, adobe2APIVideoModelPrefixes)
}

func IsAdobe2APIImageRelay(info *relaycommon.RelayInfo) bool {
	if info == nil {
		return false
	}
	if !isAdobe2APIChannel(info) {
		return false
	}
	if IsAdobe2APIImageOriginModel(info.OriginModelName) {
		return true
	}
	if info.ChannelMeta != nil && IsAdobe2APIImageOriginModel(info.UpstreamModelName) {
		return true
	}
	return false
}

func IsAdobe2APIVideoChatRelay(info *relaycommon.RelayInfo) bool {
	if info == nil {
		return false
	}
	if !isAdobe2APIChannel(info) {
		return false
	}
	if IsAdobe2APIVideoChatOriginModel(info.OriginModelName) {
		return true
	}
	if info.ChannelMeta != nil && IsAdobe2APIVideoChatOriginModel(info.UpstreamModelName) {
		return true
	}
	return false
}

func isAdobe2APIChannel(info *relaycommon.RelayInfo) bool {
	if info == nil {
		return false
	}
	baseURL := ""
	if info.ChannelMeta != nil {
		if info.ChannelMeta.ChannelId == 75 {
			return true
		}
		baseURL = info.ChannelMeta.ChannelBaseUrl
	}
	baseURL = strings.TrimSpace(strings.ToLower(baseURL))
	if baseURL == "" {
		return false
	}
	if strings.Contains(baseURL, "adobe2api") || strings.Contains(baseURL, "gongju") {
		return true
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	return parsed.Host == "45.67.221.45:6001"
}

func hasAdobe2APIPrefix(model string, prefixes []string) bool {
	name := strings.ToLower(strings.TrimSpace(model))
	name = strings.TrimPrefix(name, "adobe2api/")
	name = strings.TrimPrefix(name, "adobe/")
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func resolveAdobe2APIUpstreamModel(info *relaycommon.RelayInfo, fallback string) string {
	if info != nil && info.ChannelMeta != nil && strings.TrimSpace(info.UpstreamModelName) != "" {
		return strings.TrimSpace(info.UpstreamModelName)
	}
	name := strings.TrimSpace(fallback)
	for _, prefix := range []string{"adobe2api/", "adobe/", "adobe2api-", "adobe-"} {
		if strings.HasPrefix(strings.ToLower(name), prefix) {
			return strings.TrimSpace(name[len(prefix):])
		}
	}
	if info != nil && strings.TrimSpace(info.OriginModelName) != "" {
		return strings.TrimSpace(info.OriginModelName)
	}
	return name
}

func ConvertAdobe2APIImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	if info != nil &&
		info.RelayMode == relayconstant.RelayModeImagesEdits &&
		hasAdobe2APIMultipartImageFiles(c, request) {
		return BuildAdobe2APIImageEditMultipart(c, info, request)
	}

	modelName := resolveAdobe2APIUpstreamModel(info, request.Model)
	if modelName == "" {
		return nil, fmt.Errorf("model is required")
	}
	body := map[string]any{
		"model":  modelName,
		"prompt": request.Prompt,
	}
	if request.N != nil && *request.N > 0 {
		body["n"] = *request.N
	}
	if size := adobe2APIOpenAIImageSize(request); size != "" {
		body["size"] = size
	}
	if quality := strings.TrimSpace(request.Quality); quality != "" {
		body["quality"] = quality
	}
	adobe2APICopyImageRawFields(body, request)
	imageSize := adobe2APIImageSize(info, request)
	if imageSize != "" {
		body["image_size"] = imageSize
		body["output_resolution"] = imageSize
	}
	if aspectRatio := adobe2APIAspectRatio(request); aspectRatio != "" {
		body["aspect_ratio"] = aspectRatio
	}
	refs, err := adobe2APIReferenceImages(c, request)
	if err != nil {
		return nil, err
	}
	if len(refs) > 0 {
		body["reference_images"] = refs
	}
	return body, nil
}

// BuildAdobe2APIImageEditMultipart 将 multipart 图生图转为 Adobe2API /v1/images/edits 表单。
// 多图参考重复字段名 image（不用 image[]）；URL 参考图走 JSON reference_images 路径。
func BuildAdobe2APIImageEditMultipart(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (*bytes.Buffer, error) {
	if info != nil {
		info.Adobe2APIImageEditMultipart = true
	}
	imageFiles, err := collectAdobe2APIMultipartImageFiles(c)
	if err != nil {
		return nil, err
	}
	if len(imageFiles) == 0 {
		return nil, fmt.Errorf("image is required")
	}

	modelName := resolveAdobe2APIUpstreamModel(info, request.Model)
	if modelName == "" {
		return nil, fmt.Errorf("model is required")
	}

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	writeAdobe2APIImageEditFormFields(writer, info, request, modelName)

	for i, fileHeader := range imageFiles {
		file, err := fileHeader.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open image file %d: %w", i, err)
		}
		mimeType := detectImageMimeType(fileHeader.Filename)
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="image"; filename="%s"`, fileHeader.Filename))
		h.Set("Content-Type", mimeType)
		part, err := writer.CreatePart(h)
		if err != nil {
			_ = file.Close()
			return nil, fmt.Errorf("create form part failed for image %d: %w", i, err)
		}
		if _, err := io.Copy(part, file); err != nil {
			_ = file.Close()
			return nil, fmt.Errorf("copy file failed for image %d: %w", i, err)
		}
		_ = file.Close()
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}
	if c != nil && c.Request != nil {
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	}
	return &requestBody, nil
}

func writeAdobe2APIImageEditFormFields(writer *multipart.Writer, info *relaycommon.RelayInfo, request dto.ImageRequest, modelName string) {
	_ = writer.WriteField("model", modelName)
	if prompt := strings.TrimSpace(request.Prompt); prompt != "" {
		_ = writer.WriteField("prompt", prompt)
	}
	if request.N != nil && *request.N > 0 {
		_ = writer.WriteField("n", strconv.FormatUint(uint64(*request.N), 10))
	}
	if quality := strings.TrimSpace(request.Quality); quality != "" {
		_ = writer.WriteField("quality", quality)
	}
	if aspectRatio := adobe2APIAspectRatio(request); aspectRatio != "" {
		_ = writer.WriteField("aspect_ratio", aspectRatio)
	}
	if imageSize := adobe2APIImageSize(info, request); imageSize != "" {
		_ = writer.WriteField("image_size", imageSize)
		_ = writer.WriteField("output_resolution", imageSize)
	}
}

func hasAdobe2APIMultipartImageFiles(c *gin.Context, request dto.ImageRequest) bool {
	if c == nil || c.Request == nil || isJSONRequest(c) {
		return false
	}
	if len(parseJSONStringList(request.Image)) > 0 || len(parseJSONStringList(request.Images)) > 0 {
		return false
	}
	files, err := collectAdobe2APIMultipartImageFiles(c)
	return err == nil && len(files) > 0
}

func collectAdobe2APIMultipartImageFiles(c *gin.Context) ([]*multipart.FileHeader, error) {
	if c == nil || c.Request == nil {
		return nil, nil
	}
	if err := ensureMultipartFormParsed(c); err != nil {
		return nil, err
	}
	mf := c.Request.MultipartForm
	if mf == nil || mf.File == nil {
		return nil, nil
	}

	var imageFiles []*multipart.FileHeader
	if files, ok := mf.File["image"]; ok {
		imageFiles = append(imageFiles, files...)
	}
	if files, ok := mf.File["image[]"]; ok {
		imageFiles = append(imageFiles, files...)
	}
	for fieldName, files := range mf.File {
		if strings.HasPrefix(fieldName, "image[") && len(files) > 0 {
			imageFiles = append(imageFiles, files...)
		}
	}
	return imageFiles, nil
}

func adobe2APIOpenAIImageSize(request dto.ImageRequest) string {
	size := strings.TrimSpace(request.Size)
	if size == "" {
		return ""
	}
	if strings.EqualFold(size, "auto") || looksLikeImageDimensions(size) {
		return size
	}
	return ""
}

func adobe2APICopyImageRawFields(body map[string]any, request dto.ImageRequest) {
	rawFields := []struct {
		key string
		raw json.RawMessage
	}{
		{"background", request.Background},
		{"moderation", request.Moderation},
		{"output_format", request.OutputFormat},
		{"output_compression", request.OutputCompression},
		{"partial_images", request.PartialImages},
		{"input_fidelity", request.InputFidelity},
		{"response_format", json.RawMessage(strconvQuoteIfNotEmpty(request.ResponseFormat))},
	}
	for _, field := range rawFields {
		if value, ok := rawJSONValue(field.raw); ok {
			body[field.key] = value
		}
	}
	if request.Stream != nil {
		body["stream"] = *request.Stream
	}
	if request.Watermark != nil {
		body["watermark"] = *request.Watermark
	}

	for _, key := range []string{"background", "moderation", "output_format", "output_compression", "partial_images", "input_fidelity", "response_format"} {
		if _, exists := body[key]; exists {
			continue
		}
		if value, ok := adobe2APIImageOptionValue(request, key, camelizeSnakeKey(key)); ok {
			body[key] = value
		}
	}
}

func adobe2APIImageSize(info *relaycommon.RelayInfo, request dto.ImageRequest) string {
	for _, key := range []string{"image_size", "output_resolution", "resolution"} {
		if value := adobe2APIImageOptionString(request, key, camelizeSnakeKey(key)); value != "" {
			if key == "resolution" && isAdobe2APIVideoResolution(value) {
				continue
			}
			if normalized := normalizeAdobe2APIImageSize(value); normalized != "" {
				return normalized
			}
		}
	}
	if hint := adobe2APIResolutionHintFromRequest(request); hint != "" {
		return hint
	}
	size := strings.ToUpper(strings.TrimSpace(request.Size))
	if size == "1K" || size == "2K" || size == "4K" {
		return size
	}
	if inferred := adobe2APIImageSizeFromDimensions(request.Size); inferred != "" {
		return inferred
	}
	switch strings.ToLower(strings.TrimSpace(request.Quality)) {
	case "high", "hd", "4k":
		return "4K"
	case "medium", "2k":
		return "2K"
	case "low", "standard", "1k":
		return "1K"
	default:
		if info != nil && strings.HasSuffix(strings.ToLower(strings.TrimSpace(info.OriginModelName)), "-4k") {
			return "4K"
		}
		return ""
	}
}

func adobe2APIAspectRatio(request dto.ImageRequest) string {
	if value := adobe2APIImageOptionString(request, "aspect_ratio", "aspectRatio", "ratio"); value != "" {
		if ratio, _ := parseAdobe2APIAspectRatioInput(value); ratio != "" {
			return ratio
		}
	}
	if ratio, _ := parseAdobe2APIAspectRatioInput(strings.TrimSpace(request.Size)); ratio != "" {
		return ratio
	}
	value := strings.TrimSpace(request.Size)
	switch strings.ToLower(value) {
	case "1024x1024", "2048x2048", "4096x4096":
		return "1:1"
	case "1536x1024":
		return "3:2"
	case "1024x1536":
		return "2:3"
	case "1792x1024", "1920x1080":
		return "16:9"
	case "1024x1792", "1080x1920":
		return "9:16"
	default:
		return aspectRatioFromImageDimensions(value)
	}
}

func adobe2APIImageOptionString(request dto.ImageRequest, keys ...string) string {
	value, ok := adobe2APIImageOptionValue(request, keys...)
	if !ok {
		return ""
	}
	return strings.TrimSpace(anyToAdobe2APIString(value))
}

func adobe2APIImageOptionValue(request dto.ImageRequest, keys ...string) (any, bool) {
	for _, key := range keys {
		if raw, ok := request.Extra[key]; ok {
			if value, exists := rawJSONValue(raw); exists {
				return value, true
			}
		}
	}
	for _, containerKey := range []string{"metadata", "extra_body"} {
		container, ok := rawJSONObject(request.Extra[containerKey])
		if !ok {
			continue
		}
		for _, key := range keys {
			if value, exists := container[key]; exists {
				return value, true
			}
		}
		if google, ok := container["google"].(map[string]any); ok {
			if imageConfig, ok := google["image_config"].(map[string]any); ok {
				for _, key := range keys {
					if value, exists := imageConfig[key]; exists {
						return value, true
					}
				}
			}
		}
		if imageConfig, ok := container["image_config"].(map[string]any); ok {
			for _, key := range keys {
				if value, exists := imageConfig[key]; exists {
					return value, true
				}
			}
		}
	}
	return nil, false
}

func rawJSONObject(raw json.RawMessage) (map[string]any, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var obj map[string]any
	if err := common.Unmarshal(raw, &obj); err == nil && obj != nil {
		return obj, true
	}
	var jsonString string
	if err := common.Unmarshal(raw, &jsonString); err != nil || strings.TrimSpace(jsonString) == "" {
		return nil, false
	}
	if err := common.Unmarshal([]byte(jsonString), &obj); err != nil || obj == nil {
		return nil, false
	}
	return obj, true
}

func rawJSONValue(raw json.RawMessage) (any, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var value any
	if err := common.Unmarshal(raw, &value); err != nil {
		return nil, false
	}
	return value, true
}

func normalizeAdobe2APIImageSize(value string) string {
	if isAdobe2APIVideoResolution(value) {
		return ""
	}
	upper := strings.ToUpper(strings.TrimSpace(value))
	switch upper {
	case "1K", "2K", "4K":
		return upper
	default:
		return ""
	}
}

func isAdobe2APIVideoResolution(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "720p", "1080p", "480p", "2160p":
		return true
	default:
		return false
	}
}

func adobe2APIResolutionHintFromRequest(request dto.ImageRequest) string {
	for _, key := range []string{"aspect_ratio", "aspectRatio", "ratio"} {
		if value := adobe2APIImageOptionString(request, key); value != "" {
			if _, hint := parseAdobe2APIAspectRatioInput(value); hint != "" {
				return hint
			}
		}
	}
	if _, hint := parseAdobe2APIAspectRatioInput(strings.TrimSpace(request.Size)); hint != "" {
		return hint
	}
	return ""
}

func parseAdobe2APIAspectRatioInput(value string) (ratio string, resolutionHint string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}
	lower := strings.ToLower(value)
	for _, item := range []struct {
		suffix string
		res    string
	}{
		{"-4k", "4K"},
		{"-2k", "2K"},
		{"-1k", "1K"},
	} {
		if strings.HasSuffix(lower, item.suffix) {
			candidate := strings.TrimSpace(value[:len(value)-len(item.suffix)])
			if normalized := normalizePureAspectRatio(candidate); normalized != "" {
				return normalized, item.res
			}
		}
	}
	if normalized := normalizePureAspectRatio(value); normalized != "" {
		return normalized, ""
	}
	return "", ""
}

func normalizePureAspectRatio(value string) string {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 {
		return ""
	}
	width, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil || width <= 0 || height <= 0 {
		return ""
	}
	divisor := gcd(width, height)
	return fmt.Sprintf("%d:%d", width/divisor, height/divisor)
}

func adobe2APIImageSizeFromDimensions(size string) string {
	width, height, ok := parseImageDimensions(size)
	if !ok {
		return ""
	}
	maxSide := width
	if height > maxSide {
		maxSide = height
	}
	switch {
	case maxSide >= 3000:
		return "4K"
	case maxSide >= 1800:
		return "2K"
	case maxSide >= 900:
		return "1K"
	default:
		return ""
	}
}

func aspectRatioFromImageDimensions(size string) string {
	width, height, ok := parseImageDimensions(size)
	if !ok || width == 0 || height == 0 {
		return ""
	}
	divisor := gcd(width, height)
	return fmt.Sprintf("%d:%d", width/divisor, height/divisor)
}

func parseImageDimensions(size string) (int, int, bool) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(size)), "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	width, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil || width <= 0 || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

func looksLikeImageDimensions(size string) bool {
	_, _, ok := parseImageDimensions(size)
	return ok
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		return -a
	}
	return a
}

func anyToAdobe2APIString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return ""
	}
}

func camelizeSnakeKey(key string) string {
	var b strings.Builder
	upperNext := false
	for _, r := range key {
		if r == '_' {
			upperNext = true
			continue
		}
		if upperNext {
			b.WriteString(strings.ToUpper(string(r)))
			upperNext = false
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func strconvQuoteIfNotEmpty(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	quoted, err := common.Marshal(value)
	if err != nil {
		return ""
	}
	return string(quoted)
}

func adobe2APIReferenceImages(c *gin.Context, request dto.ImageRequest) ([]string, error) {
	refs := make([]string, 0, 6)
	for _, key := range []string{"reference_images", "image_refs"} {
		if raw, ok := request.Extra[key]; ok {
			refs = append(refs, rawJSONStringList(raw)...)
		}
	}
	extracted, err := collectChatImageReferenceURLs(c, request)
	if err != nil {
		return nil, err
	}
	refs = append(refs, extracted...)
	return uniqueNonEmptyStrings(refs), nil
}

func ConvertAdobe2APIOpenAIChatRequest(c *gin.Context, request *dto.GeneralOpenAIRequest, info *relaycommon.RelayInfo) (any, error) {
	if request == nil {
		return nil, fmt.Errorf("request is nil")
	}
	body := map[string]any{}
	if c != nil {
		if storage, err := common.GetBodyStorage(c); err == nil {
			if raw, readErr := storage.Bytes(); readErr == nil && len(raw) > 0 {
				_ = common.Unmarshal(raw, &body)
			}
		}
	}
	body["model"] = resolveAdobe2APIUpstreamModel(info, request.Model)
	body["messages"] = adobe2APIChatMessages(body, request)

	for _, key := range []string{"duration", "aspect_ratio", "resolution", "generate_audio", "reference_mode", "video_reference_mode"} {
		if value, ok := body[key]; ok {
			body[key] = value
		}
	}
	if refMode := strings.TrimSpace(rawMapString(body, "reference_mode")); refMode != "" {
		if strings.TrimSpace(rawMapString(body, "video_reference_mode")) == "" {
			body["video_reference_mode"] = refMode
		}
	}
	delete(body, "image_urls")
	delete(body, "image_url")
	delete(body, "stream")
	return body, nil
}

func adobe2APIChatMessages(body map[string]any, request *dto.GeneralOpenAIRequest) []dto.Message {
	messages := request.Messages
	imageURLs := make([]string, 0, 4)
	imageURLs = append(imageURLs, rawMapStringList(body, "image_urls")...)
	if single := rawMapString(body, "image_url"); single != "" {
		imageURLs = append(imageURLs, single)
	}
	imageURLs = uniqueNonEmptyStrings(imageURLs)
	if len(imageURLs) == 0 || len(messages) == 0 {
		return messages
	}
	last := len(messages) - 1
	content := make([]dto.MediaContent, 0, len(imageURLs)+1)
	if text := strings.TrimSpace(messages[last].StringContent()); text != "" {
		content = append(content, dto.MediaContent{Type: dto.ContentTypeText, Text: text})
	}
	for _, imageURL := range imageURLs {
		content = append(content, dto.MediaContent{
			Type: dto.ContentTypeImageURL,
			ImageUrl: &dto.MessageImageUrl{
				Url: imageURL,
			},
		})
	}
	messages[last].SetMediaContent(content)
	return messages
}

func rawJSONString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := common.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	var value any
	if err := common.Unmarshal(raw, &value); err == nil {
		switch v := value.(type) {
		case float64:
			if v == float64(int64(v)) {
				return fmt.Sprintf("%d", int64(v))
			}
			return strings.TrimSpace(fmt.Sprintf("%v", v))
		case bool:
			return fmt.Sprintf("%t", v)
		}
	}
	return ""
}

func rawJSONStringList(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var single string
	if err := common.Unmarshal(raw, &single); err == nil {
		if single = strings.TrimSpace(single); single != "" {
			return []string{single}
		}
		return nil
	}
	var list []string
	if err := common.Unmarshal(raw, &list); err == nil {
		return uniqueNonEmptyStrings(list)
	}
	return nil
}

func rawMapString(body map[string]any, key string) string {
	if body == nil {
		return ""
	}
	switch v := body[key].(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return strings.TrimSpace(v.String())
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%v", v)
	default:
		return ""
	}
}

func rawMapStringList(body map[string]any, key string) []string {
	if body == nil {
		return nil
	}
	switch v := body[key].(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return uniqueNonEmptyStrings(out)
	case []string:
		return uniqueNonEmptyStrings(v)
	case string:
		parts := strings.FieldsFunc(v, func(r rune) bool {
			return r == ',' || r == '\n' || r == '\r'
		})
		return uniqueNonEmptyStrings(parts)
	default:
		return nil
	}
}

func uniqueNonEmptyStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
