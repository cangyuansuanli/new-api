package vendorpatch

// ImageTransformResult carries vendor-specific metadata for consume logging after
// the upstream request body has been patched.
type ImageTransformResult struct {
	// LogSize overrides request.Size in consume logs when non-empty (user-facing size).
	LogSize string
	// SuppressQualityLog omits the quality field from consume logs when the patcher
	// strips quality from the upstream request.
	SuppressQualityLog bool
}
