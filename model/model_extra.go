package model

func GetModelEnableGroups(modelName string) []string {
	if modelName == "" {
		return make([]string, 0)
	}

	snapshot := loadPricingSnapshot()
	if snapshot == nil {
		return make([]string, 0)
	}
	groups, ok := snapshot.modelEnableGroups[modelName]
	if !ok {
		return make([]string, 0)
	}
	return groups
}

// GetModelQuotaTypes 返回指定模型的计费类型集合（来自缓存）
func GetModelQuotaTypes(modelName string) []int {
	snapshot := loadPricingSnapshot()
	if snapshot == nil {
		return []int{}
	}
	quota, ok := snapshot.modelQuotaTypeMap[modelName]
	if !ok {
		return []int{}
	}
	return []int{quota}
}
