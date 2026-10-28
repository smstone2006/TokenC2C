//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestC2CBillingAndMaturityEndToEnd(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	buyer := mustCreateUser(t, client, &service.User{
		Email: fmt.Sprintf("c2c-buyer-%d@example.com", time.Now().UnixNano()), PasswordHash: "hash", Balance: 100,
	})
	seller := mustCreateUser(t, client, &service.User{
		Email: fmt.Sprintf("c2c-seller-%d@example.com", time.Now().UnixNano()), PasswordHash: "hash", Balance: 0,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: buyer.ID, Key: "sk-c2c-" + uuid.NewString(), Name: "c2c buyer",
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "c2c-supply-" + uuid.NewString(), Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
	})
	cleanupC2CAccountAfterTest(t, account.ID)
	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO c2c_provider_accounts
			(account_id, owner_user_id, pool_status, verification_status, identity_hash, verified_at)
		VALUES ($1, $2, 'active', 'verified', $3, NOW())
	`, account.ID, seller.ID, uuid.NewString())
	require.NoError(t, err)

	requestID := uuid.NewString()
	cmd := &service.UsageBillingCommand{
		RequestID: requestID, APIKeyID: apiKey.ID, UserID: buyer.ID, AccountID: account.ID,
		AccountType: service.AccountTypeAPIKey, Model: "gpt-test",
		InputTokens: 120, OutputTokens: 30, BalanceCost: 2,
	}
	first, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.True(t, first.Applied)
	second, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.False(t, second.Applied)

	var buyerBalance, sellerBalance, sellerPending float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT balance FROM users WHERE id = $1`, buyer.ID).Scan(&buyerBalance))
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT balance, c2c_pending_balance FROM users WHERE id = $1`, seller.ID).Scan(&sellerBalance, &sellerPending))
	require.InDelta(t, 98, buyerBalance, 0.00000001)
	require.InDelta(t, 0, sellerBalance, 0.00000001)
	require.InDelta(t, 1.8, sellerPending, 0.00000001)

	var tradeID int64
	var sellerEarning, platformFee float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT id, seller_earning, platform_fee FROM c2c_trades
		WHERE request_id = $1 AND api_key_id = $2
	`, requestID, apiKey.ID).Scan(&tradeID, &sellerEarning, &platformFee))
	require.InDelta(t, 1.8, sellerEarning, 0.00000001)
	require.InDelta(t, 0.2, platformFee, 0.00000001)

	var ledgerCount int
	var ledgerSum float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*), SUM(amount) FROM c2c_ledger_entries WHERE trade_id = $1`, tradeID).Scan(&ledgerCount, &ledgerSum))
	require.Equal(t, 3, ledgerCount)
	require.InDelta(t, 0, ledgerSum, 0.00000001)

	_, err = integrationDB.ExecContext(ctx, `UPDATE c2c_trades SET matures_at = NOW() - INTERVAL '1 second' WHERE id = $1`, tradeID)
	require.NoError(t, err)
	settler := service.NewC2CService(integrationDB, nil, nil)
	t.Cleanup(settler.Stop)
	count, err := settler.MaturePending(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT balance, c2c_pending_balance FROM users WHERE id = $1`, seller.ID).Scan(&sellerBalance, &sellerPending))
	require.InDelta(t, 1.8, sellerBalance, 0.00000001)
	require.InDelta(t, 0, sellerPending, 0.00000001)
	var status string
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT status FROM c2c_trades WHERE id = $1`, tradeID).Scan(&status))
	require.Equal(t, "settled", status)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*), SUM(amount) FROM c2c_ledger_entries WHERE trade_id = $1`, tradeID).Scan(&ledgerCount, &ledgerSum))
	require.Equal(t, 5, ledgerCount)
	require.InDelta(t, 0, ledgerSum, 0.00000001)
}

func TestC2CBillingSelfMatchKeepsDoubleEntryAndPlatformFee(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)
	user := mustCreateUser(t, client, &service.User{
		Email: fmt.Sprintf("c2c-self-%d@example.com", time.Now().UnixNano()), PasswordHash: "hash", Balance: 10,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID, Key: "sk-c2c-self-" + uuid.NewString(), Name: "self buyer",
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "c2c-self-supply-" + uuid.NewString(), Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
	})
	cleanupC2CAccountAfterTest(t, account.ID)
	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO c2c_provider_accounts
			(account_id, owner_user_id, pool_status, verification_status, identity_hash, verified_at)
		VALUES ($1, $2, 'active', 'verified', $3, NOW())
	`, account.ID, user.ID, uuid.NewString())
	require.NoError(t, err)

	requestID := uuid.NewString()
	result, err := repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID: requestID, APIKeyID: apiKey.ID, UserID: user.ID, AccountID: account.ID,
		AccountType: service.AccountTypeAPIKey, Model: "gpt-test", BalanceCost: 1,
	})
	require.NoError(t, err)
	require.True(t, result.Applied)

	var available, pending float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT balance, c2c_pending_balance FROM users WHERE id = $1`, user.ID).Scan(&available, &pending))
	require.InDelta(t, 9, available, 0.00000001)
	require.InDelta(t, 0.9, pending, 0.00000001)

	var buyerID, sellerID int64
	var charge, earning, fee float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT buyer_user_id, seller_user_id, buyer_charge, seller_earning, platform_fee
		FROM c2c_trades WHERE request_id = $1 AND api_key_id = $2
	`, requestID, apiKey.ID).Scan(&buyerID, &sellerID, &charge, &earning, &fee))
	require.Equal(t, user.ID, buyerID)
	require.Equal(t, user.ID, sellerID)
	require.InDelta(t, charge, earning+fee, 0.00000001)
	require.InDelta(t, 0.1, fee, 0.00000001)
}

// These fixtures commit real transactions. Remove their C2C references in FK
// order so later repository suites can delete users and API keys normally.
func cleanupC2CAccountAfterTest(t *testing.T, accountID int64) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		tx, err := integrationDB.BeginTx(ctx, nil)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback() }()
		for _, query := range []string{
			`DELETE FROM c2c_ledger_entries WHERE trade_id IN (SELECT id FROM c2c_trades WHERE account_id = $1)`,
			`DELETE FROM c2c_trades WHERE account_id = $1`,
			`DELETE FROM c2c_provider_accounts WHERE account_id = $1`,
		} {
			_, err := tx.ExecContext(ctx, query, accountID)
			require.NoError(t, err, "clean C2C fixture references")
		}
		require.NoError(t, tx.Commit())
	})
}
