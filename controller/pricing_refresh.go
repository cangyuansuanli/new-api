package controller

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

func refreshPricingAfterMutation() {
	if err := model.RefreshPricing(); err != nil {
		common.SysError(fmt.Sprintf("refresh pricing after mutation: %v", err))
	}
}
