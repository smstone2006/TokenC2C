//go:build unit

package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

type c2cFakeAccountManager struct {
	created       *CreateAccountInput
	deletedID     int64
	schedulableID int64
	schedulable   bool
}

func (f *c2cFakeAccountManager) CreateAccount(_ context.Context, input *CreateAccountInput) (*Account, error) {
	f.created = input
	return &Account{ID: 71, Name: input.Name, Platform: input.Platform, Type: input.Type}, nil
}
func (f *c2cFakeAccountManager) DeleteAccount(_ context.Context, id int64) error {
	f.deletedID = id
	return nil
}
func (f *c2cFakeAccountManager) SetAccountSchedulable(_ context.Context, id int64, value bool) (*Account, error) {
	f.schedulableID, f.schedulable = id, value
	return &Account{ID: id, Schedulable: value}, nil
}

func TestValidateC2CProviderInput(t *testing.T) {
	tests := []struct {
		name    string
		input   C2CCreateProviderAccountInput
		wantErr string
	}{
		{
			name:  "official openai api key",
			input: C2CCreateProviderAccountInput{Name: "mine", Platform: "OPENAI", Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "sk-test"}},
		},
		{
			name:    "composite is routing only",
			input:   C2CCreateProviderAccountInput{Name: "mine", Platform: PlatformComposite, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "x"}},
			wantErr: "unsupported provider platform",
		},
		{
			name:    "custom endpoint rejected",
			input:   C2CCreateProviderAccountInput{Name: "mine", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "x", "base_url": "https://example.test"}},
			wantErr: "custom upstream endpoints are not allowed",
		},
		{
			name:    "upstream relay rejected",
			input:   C2CCreateProviderAccountInput{Name: "mine", Platform: PlatformAntigravity, Type: AccountTypeUpstream, Credentials: map[string]any{"api_key": "x"}},
			wantErr: "unsupported account type for provider",
		},
		{
			name:    "non-positive limit rejected",
			input:   C2CCreateProviderAccountInput{Name: "mine", Platform: PlatformGemini, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "x"}, DailyLimitUSD: c2cFloatPtr(0)},
			wantErr: "daily_limit_usd must be greater than zero",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateC2CProviderInput(&tt.input)
			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, stringsLower(tt.input.Platform), tt.input.Platform)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestApplyC2COfficialEndpointDefaults(t *testing.T) {
	credentials := map[string]any{"api_key": "secret"}
	applyC2COfficialEndpointDefaults(PlatformKimi, AccountModeCoding, APIProtocolAnthropic, credentials)
	require.Equal(t, DefaultKimiCodingAnthropicBaseURL, credentials["base_url"])
	require.Equal(t, AccountModeCoding, credentials["account_mode"])
	require.Equal(t, APIProtocolAnthropic, credentials["api_protocol"])

	deepseek := map[string]any{"api_key": "secret"}
	applyC2COfficialEndpointDefaults(PlatformDeepseek, "", "", deepseek)
	require.Equal(t, DefaultDeepseekBaseURL, deepseek["base_url"])
}

func TestC2CCredentialIdentityHashIsStableAcrossMapOrder(t *testing.T) {
	one, err := c2cCredentialIdentityHash(PlatformOpenAI, AccountTypeAPIKey, map[string]any{"api_key": "secret", "org": "o"})
	require.NoError(t, err)
	two, err := c2cCredentialIdentityHash(PlatformOpenAI, AccountTypeAPIKey, map[string]any{"org": "o", "api_key": "secret"})
	require.NoError(t, err)
	require.Equal(t, one, two)
	require.Len(t, one, 64)
}

func TestCreateProviderAccountStartsUnschedulableAndHidesCredentials(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	manager := &c2cFakeAccountManager{}
	svc := newC2CService(db, manager, nil)

	mock.ExpectExec("INSERT INTO c2c_provider_accounts").
		WithArgs(int64(71), int64(9), sqlmock.AnyArg(), nil).
		WillReturnResult(sqlmock.NewResult(1, 1))
	now := time.Now()
	mock.ExpectQuery("SELECT a.id, a.name, a.platform").
		WithArgs(int64(9), int64(71)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "platform", "type", "status", "schedulable", "pool_status", "verification_status", "verification_error", "daily_limit_usd", "daily_used_usd", "verified_at", "created_at",
		}).AddRow(71, "mine", PlatformOpenAI, AccountTypeAPIKey, StatusActive, false, C2CPoolStatusVerifying, C2CVerificationPending, "", nil, 0, nil, now))

	item, err := svc.CreateProviderAccount(context.Background(), 9, C2CCreateProviderAccountInput{
		Name: "mine", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "secret"},
	})
	require.NoError(t, err)
	require.Equal(t, int64(71), item.AccountID)
	require.False(t, item.Schedulable)
	require.Equal(t, int64(71), manager.schedulableID)
	require.False(t, manager.schedulable)
	require.Equal(t, map[string]any{"api_key": "secret"}, manager.created.Credentials)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSetPoolStateCannotAccessAnotherUsersAccount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	svc := newC2CService(db, &c2cFakeAccountManager{}, nil)
	mock.ExpectQuery("SELECT a.id, a.name, a.platform").
		WithArgs(int64(10), int64(71)).
		WillReturnError(sql.ErrNoRows)
	_, err = svc.SetPoolState(context.Background(), 10, 71, false)
	require.ErrorIs(t, err, ErrC2CAccountNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteProviderAccountRemovesPoolMetadataAfterSoftDelete(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	manager := &c2cFakeAccountManager{}
	svc := newC2CService(db, manager, nil)
	now := time.Now()

	mock.ExpectQuery("SELECT a.id, a.name, a.platform").
		WithArgs(int64(9), int64(71)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "platform", "type", "status", "schedulable", "pool_status", "verification_status", "verification_error", "daily_limit_usd", "daily_used_usd", "verified_at", "created_at",
		}).AddRow(71, "mine", PlatformOpenAI, AccountTypeAPIKey, StatusActive, true, C2CPoolStatusActive, C2CVerificationVerified, "", nil, 0, now, now))
	mock.ExpectExec("DELETE FROM c2c_provider_accounts").
		WithArgs(int64(9), int64(71)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, svc.DeleteProviderAccount(context.Background(), 9, 71))
	require.Equal(t, int64(71), manager.schedulableID)
	require.False(t, manager.schedulable)
	require.Equal(t, int64(71), manager.deletedID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func c2cFloatPtr(v float64) *float64 { return &v }
func stringsLower(v string) string {
	if v == "OPENAI" {
		return "openai"
	}
	return v
}
