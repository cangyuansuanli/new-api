package model

import (
	"fmt"
	"math"

	"github.com/QuantumNous/new-api/common"

	"github.com/shopspring/decimal"
)

const (
	epayModernTopUpQuotaMaxRatio = 1.15
	epayModernTopUpQuotaMinRatio = 0.85
)

// EpayModernTopUpQuotaCeiling is the maximum quota allowed for a modern (CNY face) epay order.
// Caps at 1.15× face value at QuotaPerUnit to block rate-inversion / double-credit bugs.
func EpayModernTopUpQuotaCeiling(cnyAmount int64) int {
	if cnyAmount <= 0 {
		return 0
	}
	return int(decimal.NewFromInt(cnyAmount).
		Mul(decimal.NewFromFloat(common.QuotaPerUnit)).
		Mul(decimal.NewFromFloat(epayModernTopUpQuotaMaxRatio)).
		Round(0).
		IntPart())
}

// ValidateEpayModernTopUpQuota rejects epay credits that deviate from the configured 1:1 policy.
// Legacy USD-face orders are skipped.
func ValidateEpayModernTopUpQuota(topUp *TopUp, quotaToAdd int) error {
	if topUp == nil || topUp.PaymentProvider != PaymentProviderEpay {
		return nil
	}
	if IsLegacyEpayUsdTopUpAmount(topUp) {
		return nil
	}
	if topUp.Amount <= 0 || quotaToAdd <= 0 {
		return fmt.Errorf("invalid epay top-up amount=%d quota=%d", topUp.Amount, quotaToAdd)
	}

	ceiling := EpayModernTopUpQuotaCeiling(topUp.Amount)
	if quotaToAdd > ceiling {
		return fmt.Errorf(
			"epay top-up quota %d exceeds ceiling %d for cny amount %d (rate=%.4f)",
			quotaToAdd, ceiling, topUp.Amount, EpayDisplayExchangeRate(),
		)
	}

	expected := EpayTopUpQuota(topUp)
	if expected <= 0 {
		return fmt.Errorf("invalid expected epay quota for amount %d", topUp.Amount)
	}
	ratio := float64(quotaToAdd) / float64(expected)
	if ratio < epayModernTopUpQuotaMinRatio || ratio > epayModernTopUpQuotaMaxRatio {
		return fmt.Errorf(
			"epay top-up quota ratio %.4f out of bounds [%.2f, %.2f] (amount=%d quota=%d expected=%d rate=%.4f)",
			ratio, epayModernTopUpQuotaMinRatio, epayModernTopUpQuotaMaxRatio,
			topUp.Amount, quotaToAdd, expected, EpayDisplayExchangeRate(),
		)
	}

	// Sanity: exchange rate used for epay should stay in a reasonable CNY display range.
	rate := EpayDisplayExchangeRate()
	if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) || rate < 0.5 || rate > 15 {
		return fmt.Errorf("epay display exchange rate out of safe range: %.4f", rate)
	}
	return nil
}
