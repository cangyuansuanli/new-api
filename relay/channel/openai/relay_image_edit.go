package openai

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
)

// CollectImageEditReferenceDataURIs 从 JSON 字段或 multipart 表单收集图生图参考图。
func CollectImageEditReferenceDataURIs(c *gin.Context, request dto.ImageRequest) (images []string, err error) {
	images = append(images, parseJSONStringList(request.Image)...)
	images = append(images, parseJSONStringList(request.Images)...)

	if c != nil && c.Request != nil && !isJSONRequest(c) {
		mf := c.Request.MultipartForm
		if mf == nil {
			mf, err = common.ParseMultipartFormReusable(c)
			if err != nil {
				return nil, err
			}
			c.Request.MultipartForm = mf
			c.Request.PostForm = mf.Value
		}
		if mf != nil {
			for _, key := range []string{"image", "image[]"} {
				for _, fh := range mf.File[key] {
					dataURI, convErr := multipartFileToDataURI(fh)
					if convErr != nil {
						return nil, convErr
					}
					images = append(images, dataURI)
				}
			}
			for fieldName, files := range mf.File {
				if strings.HasPrefix(fieldName, "image[") && len(files) > 0 {
					for _, fh := range files {
						dataURI, convErr := multipartFileToDataURI(fh)
						if convErr != nil {
							return nil, convErr
						}
						images = append(images, dataURI)
					}
				}
			}
		}
	}
	return images, nil
}
