//go:build unit

package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestApplyC2CTradeSplitsChargeAndCreditsPending(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)

	mock.ExpectQuery("SELECT owner_user_id").WithArgs(int64(33)).
		WillReturnRows(sqlmock.NewRows([]string{"owner_user_id"}).AddRow(8))
	mock.ExpectQuery("INSERT INTO c2c_trades").
		WithArgs("req-1", int64(44), int64(7), int64(8), int64(33), "gpt-test", 100, 50, 10, 5, 1.25, 1.125, 0.125, service.C2CSellerShareBPS).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(91))
	mock.ExpectExec("UPDATE users").WithArgs(1.125, int64(8)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO c2c_ledger_entries").
		WithArgs(int64(91), int64(7), 1.25, "trade:91:buyer:charge", int64(8), 1.125, "trade:91:seller:pending", 0.125, "trade:91:platform:fee").
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectQuery("UPDATE c2c_provider_accounts").WithArgs(int64(33), 1.25).
		WillReturnRows(sqlmock.NewRows([]string{"limit_reached"}).AddRow(false))
	mock.ExpectCommit()

	err = applyC2CTrade(context.Background(), tx, &service.UsageBillingCommand{
		RequestID: "req-1", APIKeyID: 44, UserID: 7, AccountID: 33, Model: "gpt-test",
		InputTokens: 100, OutputTokens: 50, CacheCreationTokens: 10, CacheReadTokens: 5,
		BalanceCost: 1.25,
	})
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestApplyC2CTradeIgnoresAdminOwnedAccount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	mock.ExpectQuery("SELECT owner_user_id").WithArgs(int64(33)).WillReturnError(sql.ErrNoRows)
	mock.ExpectCommit()

	err = applyC2CTrade(context.Background(), tx, &service.UsageBillingCommand{AccountID: 33, BalanceCost: 1})
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}
