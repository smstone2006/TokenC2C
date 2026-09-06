package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// applyC2CTrade records the seller/platform split in the same transaction as
// the buyer charge. An account without verified C2C ownership is a normal
// administrator account and is intentionally ignored.
func applyC2CTrade(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand) error {
	if cmd == nil || cmd.AccountID <= 0 || cmd.BalanceCost <= 0 {
		return nil
	}

	var sellerUserID int64
	err := tx.QueryRowContext(ctx, `
		SELECT owner_user_id
		FROM c2c_provider_accounts
		WHERE account_id = $1 AND verification_status = 'verified'
	`, cmd.AccountID).Scan(&sellerUserID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}

	charge := service.QuantizeUsageBillingAmount(cmd.BalanceCost)
	sellerEarning := service.QuantizeUsageBillingAmount(charge * float64(service.C2CSellerShareBPS) / 10000)
	platformFee := service.QuantizeUsageBillingAmount(charge - sellerEarning)
	// Quantize the earning from the final fee so the persisted NUMERIC values
	// satisfy buyer_charge = seller_earning + platform_fee exactly.
	sellerEarning = service.QuantizeUsageBillingAmount(charge - platformFee)

	var tradeID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO c2c_trades (
			request_id, api_key_id, buyer_user_id, seller_user_id, account_id,
			model, input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens,
			buyer_charge, seller_earning, platform_fee, seller_share_bps, matures_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, NOW() + INTERVAL '24 hours'
		)
		RETURNING id
	`,
		cmd.RequestID, cmd.APIKeyID, cmd.UserID, sellerUserID, cmd.AccountID,
		cmd.Model, cmd.InputTokens, cmd.OutputTokens, cmd.CacheCreationTokens, cmd.CacheReadTokens,
		charge, sellerEarning, platformFee, service.C2CSellerShareBPS,
	).Scan(&tradeID)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE users
		SET c2c_pending_balance = c2c_pending_balance + $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`, sellerEarning, sellerUserID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO c2c_ledger_entries
			(trade_id, user_id, wallet, entry_type, amount, idempotency_key)
		VALUES
			($1, $2, 'available', 'buyer_charge', (-$3::numeric), $4),
			($1, $5, 'pending', 'seller_earning', $6::numeric, $7),
			($1, NULL, 'platform', 'platform_fee', $8::numeric, $9)
	`,
		tradeID, cmd.UserID, charge, fmt.Sprintf("trade:%d:buyer:charge", tradeID),
		sellerUserID, sellerEarning, fmt.Sprintf("trade:%d:seller:pending", tradeID),
		platformFee, fmt.Sprintf("trade:%d:platform:fee", tradeID),
	); err != nil {
		return err
	}

	var limitReached bool
	err = tx.QueryRowContext(ctx, `
		UPDATE c2c_provider_accounts
		SET daily_used_usd = CASE
				WHEN daily_window_started_at <= NOW() - INTERVAL '24 hours' THEN $2
				ELSE daily_used_usd + $2
			END,
			daily_window_started_at = CASE
				WHEN daily_window_started_at <= NOW() - INTERVAL '24 hours' THEN NOW()
				ELSE daily_window_started_at
			END,
			updated_at = NOW()
		WHERE account_id = $1
		RETURNING daily_limit_usd IS NOT NULL AND daily_used_usd >= daily_limit_usd
	`, cmd.AccountID, charge).Scan(&limitReached)
	if err != nil {
		return err
	}
	if limitReached {
		if _, err := tx.ExecContext(ctx, `UPDATE accounts SET schedulable = FALSE, updated_at = NOW() WHERE id = $1`, cmd.AccountID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO scheduler_outbox (event_type, account_id)
			VALUES ($1, $2)
		`, service.SchedulerOutboxEventAccountChanged, cmd.AccountID); err != nil {
			return err
		}
	}
	return nil
}
