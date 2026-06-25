package service

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/tidwall/sjson"
	"github.com/tidwall/gjson"

	"github.com/gin-gonic/gin"
)

var channelRegistrationPrefixes = []string{
	"gz-",
	"yunwu-",
	"oairegbox-",
	"119337-",
	"ctlove-",
	"happyhorse-",
}

type modelPublicRegistry struct {
	internalSet       map[string]struct{}
	publicToInternals map[string][]string
	internalToPublic  map[string]string
	collisions        map[string][]string
}

var (
	modelPublicRegistryMu sync.RWMutex
	modelPublicRegistryData modelPublicRegistry
	modelPublicRegistryReady bool
)

func ModelPublicNameEnabled() bool {
	return true
}

func StripChannelRegistrationPrefix(modelName string) string {
	trimmed := strings.TrimSpace(modelName)
	for _, prefix := range channelRegistrationPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(trimmed[len(prefix):])
		}
	}
	return trimmed
}

func HasChannelRegistrationPrefix(modelName string) bool {
	trimmed := strings.TrimSpace(modelName)
	for _, prefix := range channelRegistrationPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

func RefreshModelPublicNameRegistry() error {
	models := model.GetEnabledModels()
	aliases, err := model.GetAllModelPublicAliases()
	if err != nil {
		return err
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
			public = StripChannelRegistrationPrefix(internal)
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
	}
	modelPublicRegistryReady = true
	return nil
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
	if strings.HasPrefix(path, "/v1/realtime") {
		if model := strings.TrimSpace(c.Query("model")); model != "" {
			return model, "query", nil
		}
	}

	if c.Request.Method == http.MethodGet {
		return "", "", nil
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
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			return "", "", err
		}
		body, err := storage.Bytes()
		if err != nil {
			return "", "", err
		}
		_, params, parseErr := mime.ParseMediaType(contentType)
		if parseErr != nil {
			return "", "", parseErr
		}
		boundary := params["boundary"]
		if boundary == "" {
			return "", "", nil
		}
		reader := multipart.NewReader(bytes.NewReader(body), boundary)
		form, err := reader.ReadForm(common.MultipartMemoryLimit())
		if err != nil {
			return "", "", err
		}
		defer form.RemoveAll()
		if vals, ok := form.Value["model"]; ok && len(vals) > 0 {
			return strings.TrimSpace(vals[0]), "multipart", nil
		}
		return "", "", nil
	}

	return "", "", nil
}

func rewriteInboundModel(c *gin.Context, internalName string, source string) error {
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
	case "multipart":
		newBody, err = replaceMultipartModelField(body, c.Request.Header.Get("Content-Type"), internalName)
	default:
		return nil
	}
	if err != nil {
		return err
	}
	return replaceRequestBodyStorage(c, newBody)
}

func replaceMultipartModelField(body []byte, contentType, internalName string) ([]byte, error) {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, err
	}
	boundary := params["boundary"]
	if boundary == "" {
		return body, nil
	}
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	form, err := reader.ReadForm(common.MultipartMemoryLimit())
	if err != nil {
		return nil, err
	}
	defer form.RemoveAll()
	form.Value["model"] = []string{internalName}

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	if err := writer.SetBoundary(boundary); err != nil {
		return nil, err
	}
	for key, vals := range form.Value {
		for _, val := range vals {
			if err := writer.WriteField(key, val); err != nil {
				return nil, err
			}
		}
		_ = key
	}
	for key, files := range form.File {
		for _, fileHeader := range files {
			part, err := writer.CreateFormFile(key, fileHeader.Filename)
			if err != nil {
				return nil, err
			}
			file, err := fileHeader.Open()
			if err != nil {
				return nil, err
			}
			_, copyErr := io.Copy(part, file)
			file.Close()
			if copyErr != nil {
				return nil, copyErr
			}
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
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
