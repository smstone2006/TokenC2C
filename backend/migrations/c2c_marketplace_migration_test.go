package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestC2CMarketplaceMigration(t *testing.T) {
	content, err := FS.ReadFile("234_c2c_marketplace.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS c2c_pending_balance NUMERIC(20,8)")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS c2c_provider_accounts")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS c2c_trades")
	require.Contains(t, sql, "UNIQUE (request_id, api_key_id)")
	require.Contains(t, sql, "buyer_charge = seller_earning + platform_fee")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS c2c_ledger_entries")
	require.Contains(t, sql, "idempotency_key VARCHAR(320) NOT NULL UNIQUE")
}
