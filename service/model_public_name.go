package service

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/gin-gonic/gin"
)

// 渠道注册前缀由 model_channel_prefixes 表配置，RefreshModelPublicNameRegistry 加载到内存。
//
// 入站命名契约（由 middleware.PublicModelName 保证）：
//   - ApplyPublicModelTranslation 之后，请求 body/path/query 中的 model 必为 internal 名
//   - ContextKeyClientModelName 保存客户端传入的 public 名，供出站 Patch 使用
//   - 域内逻辑（relay/service/imagevendor）只应使用 OriginModelName（internal），不得再解析 public 名

type modelPublicRegistry struct {
	internalSet       map[string]struct{}
	publicToInternals map[string][]string
	internalToPublic  map[string]string
	collisions        map[string][]string
	channelPrefixes   []string
}

var (
	modelPublicRegistryMu    sync.RWMutex
	modelPublicRegistryData  modelPublicRegistry
	modelPublicRegistryReady bool
)

func ModelPublicNameEnabled() bool {
	return true
}

func StripChannelRegistrationPrefix(modelName string) string {
	trimmed := strings.TrimSpace(modelName)
	for _, prefix := range getChannelRegistrationPrefixes() {
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(trimmed[len(prefix):])
		}
	}
	return trimmed
}

func HasChannelRegistrationPrefix(modelName string) bool {
	trimmed := strings.TrimSpace(modelName)
	for _, prefix := range getChannelRegistrationPrefixes() {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

func getChannelRegistrationPrefixes() []string {
	registry := getModelPublicRegistry()
	if len(registry.channelPrefixes) == 0 {
		return nil
	}
	out := make([]string, len(registry.channelPrefixes))
	copy(out, registry.channelPrefixes)
	return out
}

func GetModelPublicNameRegistryStatus() (collisions map[string][]string, ready bool) {
	modelPublicRegistryMu.RLock()
	defer modelPublicRegistryMu.RUnlock()
	if !modelPublicRegistryReady {
		return nil, false
	}
	out := make(map[string][]string, len(modelPublicRegistryData.collisions))
	for public, internals := range modelPublicRegistryData.collisions {
		out[public] = append([]string(nil), internals...)
	}
	return out, true
}

func RefreshModelPublicNameRegistry() error {
	models := model.GetEnabledModels()
	aliases, err := model.GetAllModelPublicAliases()
	if err != nil {
		return err
	}
	prefixRows, err := model.GetEnabledModelChannelPrefixes()
	if err != nil {
		return err
	}
	channelPrefixes := make([]string, 0, len(prefixRows))
	for _, row := range prefixRows {
		prefix := model.NormalizeModelChannelPrefix(row.Prefix)
		if prefix == "" {
			continue
		}
		channelPrefixes = append(channelPrefixes, prefix)
	}

	overrideByInternal := make(map[string]string, len(aliases))
	for _, alias := range aliases {
		internal := strings.TrimSpace(alias.InternalName)
		public := strings.TrimSpace(alias.PublicName)
		if internal == "" || public == "" {
			continue
		}
		overrideByInternal[internal] = public
	}

	internalSet := make(map[string]struct{}, len(models))
	publicToInternals := make(map[string][]string)
	internalToPublic := make(map[string]string, len(models))
	collisions := make(map[string][]string)

	for _, internal := range models {
		internal = strings.TrimSpace(internal)
		if internal == "" {
			continue
		}
		internalSet[internal] = struct{}{}
		public := overrideByInternal[internal]
		if public == "" {
			public = stripWithPrefixes(internal, channelPrefixes)
		}
		if public == "" {
			public = internal
		}
		internalToPublic[internal] = public
		publicToInternals[public] = append(publicToInternals[public], internal)
	}

	for public, internals := range publicToInternals {
		if len(internals) > 1 {
			collisions[public] = append([]string(nil), internals...)
		}
	}

	modelPublicRegistryMu.Lock()
	defer modelPublicRegistryMu.Unlock()
	modelPublicRegistryData = modelPublicRegistry{
		internalSet:       internalSet,
		publicToInternals: publicToInternals,
		internalToPublic:  internalToPublic,
		collisions:        collisions,
		channelPrefixes:   channelPrefixes,
	}
	modelPublicRegistryReady = true
	return nil
}

func stripWithPrefixes(modelName string, prefixes []string) string {
	trimmed := strings.TrimSpace(modelName)
	for _, prefix := range prefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(trimmed[len(prefix):])
		}
	}
	return trimmed
}

func getModelPublicRegistry() modelPublicRegistry {
	modelPublicRegistryMu.RLock()
	defer modelPublicRegistryMu.RUnlock()
	return modelPublicRegistryData
}

func ToPublicModelName(internalName string) string {
	internalName = strings.TrimSpace(internalName)
	if internalName == "" {
		return ""
	}
	registry := getModelPublicRegistry()
	if public, ok := registry.internalToPublic[internalName]; ok && public != "" {
		return public
	}
	return StripChannelRegistrationPrefix(internalName)
}

func ResolveInternalModelName(publicOrInternal string) (internal string, clientPublic string, err error) {
	name := strings.TrimSpace(publicOrInternal)
	if name == "" {
		return "", "", errors.New("model is required")
	}

	registry := getModelPublicRegistry()
	if _, ok := registry.internalSet[name]; ok {
		public := registry.internalToPublic[name]
		if public == "" {
			public = StripChannelRegistrationPrefix(name)
		}
		return name, public, nil
	}

	internals, ok := registry.publicToInternals[name]
	if !ok || len(internals) == 0 {
		return "", "", fmt.Errorf("model %s not found", name)
	}
	if len(internals) > 1 {
		return "", "", fmt.Errorf("ambiguous public model name %s", name)
	}
	return internals[0], name, nil
}

func PublicModelNamesFromInternals(internals []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(internals))
	for _, internal := range internals {
		public := ToPublicModelName(internal)
		if public == "" {
			continue
		}
		if _, exists := seen[public]; exists {
			continue
		}
		seen[public] = struct{}{}
		out = append(out, public)
	}
	return out
}

func SetClientModelNameContext(c *gin.Context, clientPublic string) {
	if clientPublic = strings.TrimSpace(clientPublic); clientPublic != "" {
		common.SetContextKey(c, constant.ContextKeyClientModelName, clientPublic)
	}
}

func GetClientModelName(c *gin.Context) string {
	return common.GetContextKeyString(c, constant.ContextKeyClientModelName)
}

func ApplyPublicModelTranslation(c *gin.Context) error {
	if !ModelPublicNameEnabled() {
		return nil
	}
	if !modelPublicRegistryReady {
		if err := RefreshModelPublicNameRegistry(); err != nil {
			return err
		}
	}

	modelName, source, err := extractInboundModelName(c)
	if err != nil {
		return err
	}
	if modelName == "" {
		return nil
	}

	internal, clientPublic, err := ResolveInternalModelName(modelName)
	if err != nil {
		return err
	}
	SetClientModelNameContext(c, clientPublic)

	switch source {
	case "json", "form", "multipart", "query":
		if err := rewriteInboundModel(c, internal, source); err != nil {
			return err
		}
	case "path":
		// RetrieveModel reads path directly; controller handles translation.
	case "gemini_path":
		if err := rewriteGeminiRequestPath(c, internal); err != nil {
			return err
		}
	default:
		if source == "body" && internal != modelName {
			if err := rewriteInboundModel(c, internal, "json"); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractInboundModelName(c *gin.Context) (modelName string, source string, err error) {
	path := c.Request.URL.Path
	if strings.HasPrefix(path, "/v1/models/") && c.Param("model") != "" {
		return c.Param("model"), "path", nil
	}
	if model := ExtractGeminiPathModel(path); model != "" {
		return model, "gemini_path", nil
	}
	if strings.HasPrefix(path, "/v1/realtime") {
		if model := strings.TrimSpace(c.Query("model")); model != "" {
			return model, "query", nil
		}
	}

	if c.Request.Method == http.MethodGet {
		return "", "", nil
	}
	if err := normalizeImageJSONContentType(c); err != nil {
		return "", "", err
	}

	contentType := c.Request.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			return "", "", err
		}
		body, err := storage.Bytes()
		if err != nil {
			return "", "", err
		}
		if !gjson.ValidBytes(body) {
			return "", "", nil
		}
		modelResult := gjson.GetBytes(body, "model")
		if !modelResult.Exists() || modelResult.Type == gjson.Null {
			return "", "", nil
		}
		if modelResult.Type != gjson.String {
			return "", "", fmt.Errorf("field model must be a string")
		}
		return modelResult.String(), "json", nil
	}

	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			return "", "", err
		}
		body, err := storage.Bytes()
		if err != nil {
			return "", "", err
		}
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return "", "", err
		}
		return strings.TrimSpace(values.Get("model")), "form", nil
	}

	if strings.Contains(contentType, "multipart/form-data") {
		form, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return "", "", err
		}
		c.Request.MultipartForm = form
		c.Request.PostForm = url.Values(form.Value)
		if vals, ok := form.Value["model"]; ok && len(vals) > 0 {
			return strings.TrimSpace(vals[0]), "multipart", nil
		}
		return "", "", nil
	}

	return "", "", nil
}

func normalizeImageJSONContentType(c *gin.Context) error {
	if c == nil || c.Request == nil || c.Request.Method != http.MethodPost {
		return nil
	}
	path := c.Request.URL.Path
	if path != "/v1/images/generations" && path != "/v1/images/edits" && path != "/v1/edits" {
		return nil
	}
	if !common.IsMultipartContentTypeWithoutBoundary(c.Request.Header.Get("Content-Type")) {
		return nil
	}
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return err
	}
	body, err := storage.Bytes()
	if err != nil {
		return err
	}
	if gjson.ValidBytes(body) {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	return nil
}

func rewriteInboundModel(c *gin.Context, internalName string, source string) error {
	if source == "multipart" {
		form, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return err
		}
		if form.Value == nil {
			form.Value = make(map[string][]string)
		}
		form.Value["model"] = []string{internalName}
		c.Request.MultipartForm = form
		c.Request.PostForm = url.Values(form.Value)
		return nil
	}

	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return err
	}
	body, err := storage.Bytes()
	if err != nil {
		return err
	}

	var newBody []byte
	switch source {
	case "json":
		if !gjson.ValidBytes(body) {
			return nil
		}
		newBody, err = sjson.SetBytes(body, "model", internalName)
	case "form":
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return err
		}
		values.Set("model", internalName)
		newBody = []byte(values.Encode())
	default:
		return nil
	}
	if err != nil {
		return err
	}
	return replaceRequestBodyStorage(c, newBody)
}

func replaceRequestBodyStorage(c *gin.Context, newBody []byte) error {
	oldStorage, _ := common.GetBodyStorage(c)
	if oldStorage != nil {
		oldStorage.Close()
	}
	newStorage, err := common.CreateBodyStorage(newBody)
	if err != nil {
		return err
	}
	c.Set(common.KeyBodyStorage, newStorage)
	c.Request.Body = io.NopCloser(newStorage)
	c.Request.ContentLength = int64(len(newBody))
	if _, err := newStorage.Seek(0, io.SeekStart); err != nil {
		return err
	}
	return nil
}

// ExtractGeminiPathModel extracts the model segment from Gemini native paths such as
// /v1beta/models/gemini-banana-2.0:generateContent.
func ExtractGeminiPathModel(path string) string {
	modelsPrefix := "/models/"
	modelsIndex := strings.Index(path, modelsPrefix)
	if modelsIndex == -1 {
		return ""
	}

	startIndex := modelsIndex + len(modelsPrefix)
	if startIndex >= len(path) {
		return ""
	}

	colonIndex := strings.Index(path[startIndex:], ":")
	if colonIndex == -1 {
		return path[startIndex:]
	}
	return path[startIndex : startIndex+colonIndex]
}

func rewriteGeminiRequestPath(c *gin.Context, internalName string) error {
	path := c.Request.URL.Path
	oldModel := ExtractGeminiPathModel(path)
	if oldModel == "" || oldModel == internalName {
		return nil
	}
	marker := "/models/" + oldModel + ":"
	if !strings.Contains(path, marker) {
		return nil
	}
	c.Request.URL.Path = strings.Replace(path, marker, "/models/"+internalName+":", 1)
	if c.Request.URL.RawPath != "" {
		c.Request.URL.RawPath = strings.Replace(c.Request.URL.RawPath, marker, "/models/"+internalName+":", 1)
	}
	return nil
}
