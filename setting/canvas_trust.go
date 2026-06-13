package setting

import (
	"github.com/QuantumNous/new-api/common"
)

var (
	CanvasTrustEnabled   = false
	CanvasTrustSecret    = ""
	CanvasBaseURL        = "https://canvas.cangyuansuanli.cn"
	CanvasTrustTokenTTL  = 300
)

func InitCanvasTrustSetting() {
	CanvasTrustEnabled = common.GetEnvOrDefaultBool("CANVAS_TRUST_ENABLED", false)
	CanvasTrustSecret = common.GetEnvOrDefaultString("CANVAS_TRUST_SECRET", "")
	CanvasBaseURL = common.GetEnvOrDefaultString("CANVAS_BASE_URL", CanvasBaseURL)
	CanvasTrustTokenTTL = common.GetEnvOrDefault("CANVAS_TRUST_TOKEN_TTL", 300)
	if CanvasTrustTokenTTL <= 0 {
		CanvasTrustTokenTTL = 300
	}
	if CanvasTrustSecret != "" {
		CanvasTrustEnabled = true
	}
}

func CanvasTrustConfigured() bool {
	return CanvasTrustEnabled && CanvasTrustSecret != ""
}
