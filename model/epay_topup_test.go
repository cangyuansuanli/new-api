package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/require"
)

func TestEpayQuotaFromCny_oneToOne(t *testing.T) {
	oldRate := operation_setting.USDExchangeRate
	oldQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		operation_setting.USDExchangeRate = oldRate
		common.QuotaPerUnit = oldQuotaPerUnit
	})

	operation_setting.USDExchangeRate = 7.2
	common.QuotaPerUnit = 500_000

	quota := EpayQuotaFromCny(20)
	require.Equal(t, int((20.0/7.2)*500_000), quota)
}

func TestIsLegacyEpayUsdTopUpAmount(t *testing.T) {
	oldRate := operation_setting.USDExchangeRate
	t.Cleanup(func() {
		operation_setting.USDExchangeRate = oldRate
	})
	operation_setting.USDExchangeRate = 7.2

	require.True(t, IsLegacyEpayUsdTopUpAmount(&TopUp{
		PaymentProvider: PaymentProviderEpay,
		Amount:          2,
		Money:           19.6,
	}))
	require.False(t, IsLegacyEpayUsdTopUpAmount(&TopUp{
		PaymentProvider: PaymentProviderEpay,
		Amount:          20,
		Money:           19.6,
	}))
}

func TestValidateEpayModernTopUpQuota_blocksOverCredit(t *testing.T) {
	oldRate := operation_setting.USDExchangeRate
	t.Cleanup(func() {
		operation_setting.USDExchangeRate = oldRate
	})
	operation_setting.USDExchangeRate = 1

	topUp := &TopUp{
		PaymentProvider: PaymentProviderEpay,
		Amount:          20,
		Money:           19.6,
	}
	okQuota := EpayTopUpQuota(topUp)
	require.NoError(t, ValidateEpayModernTopUpQuota(topUp, okQuota))

	// Simulates the cc22pp bug: ~4.5× over-credit
	require.Error(t, ValidateEpayModernTopUpQuota(topUp, okQuota*5))
}

func TestEpayModernTopUpQuotaCeiling(t *testing.T) {
	require.Equal(t, int(20*500_000*1.15), EpayModernTopUpQuotaCeiling(20))
}
