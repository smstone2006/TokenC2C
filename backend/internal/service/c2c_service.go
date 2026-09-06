package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/lib/pq"
)

const (
	C2CPoolStatusVerifying = "verifying"
	C2CPoolStatusActive    = "active"
	C2CPoolStatusPaused    = "paused"
	C2CPoolStatusRejected  = "rejected"

	C2CVerificationPending  = "pending"
	C2CVerificationVerified = "verified"
	C2CVerificationFailed   = "failed"

	C2CSellerShareBPS = 9000
	C2CMaturityWindow = 24 * time.Hour
)

var (
	ErrC2CAccountNotFound = infraerrors.NotFound("C2C_ACCOUNT_NOT_FOUND", "provider account not found")
	ErrC2CAccountConflict = infraerrors.Conflict("C2C_ACCOUNT_EXISTS", "this provider credential is already in the pool")
)

type c2cAccountManager interface {
	CreateAccount(context.Context, *CreateAccountInput) (*Account, error)
	DeleteAccount(context.Context, int64) error
	SetAccountSchedulable(context.Context, int64, bool) (*Account, error)
}

type c2cAccountTester interface {
	RunTestBackground(context.Context, int64, string) (*ScheduledTestResult, error)
}

type C2CCreateProviderAccountInput struct {
	Name          string         `json:"name"`
	Platform      string         `json:"platform"`
	Type          string         `json:"type"`
	Credentials   map[string]any `json:"credentials"`
	AccountMode   string         `json:"account_mode,omitempty"`
	APIProtocol   string         `json:"api_protocol,omitempty"`
	DailyLimitUSD *float64       `json:"daily_limit_usd,omitempty"`
}

type C2CProviderAccount struct {
	AccountID          int64      `json:"account_id"`
	Name               string     `json:"name"`
	Platform           string     `json:"platform"`
	Type               string     `json:"type"`
	Status             string     `json:"status"`
	Schedulable        bool       `json:"schedulable"`
	PoolStatus         string     `json:"pool_status"`
	VerificationStatus string     `json:"verification_status"`
	VerificationError  string     `json:"verification_error,omitempty"`
	DailyLimitUSD      *float64   `json:"daily_limit_usd,omitempty"`
	DailyUsedUSD       float64    `json:"daily_used_usd"`
	VerifiedAt         *time.Time `json:"verified_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

type C2CWallet struct {
	AvailableBalance float64 `json:"available_balance"`
	PendingBalance   float64 `json:"pending_balance"`
	LifetimeBought   float64 `json:"lifetime_bought"`
	LifetimeEarned   float64 `json:"lifetime_earned"`
	LifetimeFees     float64 `json:"lifetime_platform_fees"`
}

type C2CStats struct {
	BuyerRequestCount  int64   `json:"buyer_request_count"`
	BuyerTokenCount    int64   `json:"buyer_token_count"`
	BuyerSpend         float64 `json:"buyer_spend"`
	SellerRequestCount int64   `json:"seller_request_count"`
	SellerTokenCount   int64   `json:"seller_token_count"`
	SellerPending      float64 `json:"seller_pending"`
	SellerSettled      float64 `json:"seller_settled"`
	SellerEarnings     float64 `json:"seller_earnings"`
}

type C2CLedgerEntry struct {
	ID        int64     `json:"id"`
	TradeID   *int64    `json:"trade_id,omitempty"`
	Wallet    string    `json:"wallet"`
	EntryType string    `json:"entry_type"`
	Amount    float64   `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

type C2CService struct {
	db             *sql.DB
	accounts       c2cAccountManager
	tester         c2cAccountTester
	settleInterval time.Duration
	stopCh         chan struct{}
	stopOnce       sync.Once
	startOnce      sync.Once
}

func NewC2CService(db *sql.DB, accounts AdminService, tester *AccountTestService) *C2CService {
	svc := newC2CService(db, accounts, tester)
	svc.Start()
	return svc
}

func newC2CService(db *sql.DB, accounts c2cAccountManager, tester c2cAccountTester) *C2CService {
	return &C2CService{
		db:             db,
		accounts:       accounts,
		tester:         tester,
		settleInterval: time.Minute,
		stopCh:         make(chan struct{}),
	}
}

func (s *C2CService) Start() {
	if s == nil || s.db == nil {
		return
	}
	s.startOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(s.settleInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					_, _ = s.MaturePending(context.Background(), 100)
				case <-s.stopCh:
					return
				}
			}
		}()
	})
}

func (s *C2CService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() { close(s.stopCh) })
}

func (s *C2CService) CreateProviderAccount(ctx context.Context, ownerUserID int64, input C2CCreateProviderAccountInput) (*C2CProviderAccount, error) {
	if ownerUserID <= 0 {
		return nil, infraerrors.New(http.StatusUnauthorized, "C2C_AUTH_REQUIRED", "user authentication required")
	}
	if err := validateC2CProviderInput(&input); err != nil {
		return nil, err
	}
	credentials := cloneC2CJSONMap(input.Credentials)
	applyC2COfficialEndpointDefaults(input.Platform, input.AccountMode, input.APIProtocol, credentials)
	identityHash, err := c2cCredentialIdentityHash(input.Platform, input.Type, credentials)
	if err != nil {
		return nil, err
	}

	account, err := s.accounts.CreateAccount(ctx, &CreateAccountInput{
		Name:                 strings.TrimSpace(input.Name),
		Platform:             strings.ToLower(strings.TrimSpace(input.Platform)),
		Type:                 strings.ToLower(strings.TrimSpace(input.Type)),
		Credentials:          credentials,
		Extra:                map[string]any{"c2c_owned": true},
		Concurrency:          1,
		Priority:             50,
		SkipDefaultGroupBind: false,
	})
	if err != nil {
		return nil, err
	}
	if _, err := s.accounts.SetAccountSchedulable(ctx, account.ID, false); err != nil {
		_ = s.accounts.DeleteAccount(ctx, account.ID)
		return nil, err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO c2c_provider_accounts
			(account_id, owner_user_id, identity_hash, daily_limit_usd)
		VALUES ($1, $2, $3, $4)
	`, account.ID, ownerUserID, identityHash, input.DailyLimitUSD)
	if err != nil {
		_ = s.accounts.DeleteAccount(ctx, account.ID)
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrC2CAccountConflict
		}
		return nil, err
	}

	item, err := s.GetProviderAccount(ctx, ownerUserID, account.ID)
	if err != nil {
		return nil, err
	}
	// Verification can involve a real upstream request, so creation returns the
	// safe unschedulable record immediately and verification continues in the
	// background. A successful check automatically joins the account to the pool.
	if s.tester != nil {
		go func() {
			_, _ = s.VerifyProviderAccount(context.Background(), ownerUserID, account.ID, "")
		}()
	}
	return item, nil
}

func (s *C2CService) ListProviderAccounts(ctx context.Context, ownerUserID int64) ([]C2CProviderAccount, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, a.name, a.platform, a.type, a.status, a.schedulable,
		       c.pool_status, c.verification_status, COALESCE(c.verification_error, ''),
		       c.daily_limit_usd, c.daily_used_usd, c.verified_at, c.created_at
		FROM c2c_provider_accounts c
		JOIN accounts a ON a.id = c.account_id AND a.deleted_at IS NULL
		WHERE c.owner_user_id = $1
		ORDER BY c.created_at DESC
	`, ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]C2CProviderAccount, 0)
	for rows.Next() {
		item, err := scanC2CProviderAccount(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (s *C2CService) GetProviderAccount(ctx context.Context, ownerUserID, accountID int64) (*C2CProviderAccount, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT a.id, a.name, a.platform, a.type, a.status, a.schedulable,
		       c.pool_status, c.verification_status, COALESCE(c.verification_error, ''),
		       c.daily_limit_usd, c.daily_used_usd, c.verified_at, c.created_at
		FROM c2c_provider_accounts c
		JOIN accounts a ON a.id = c.account_id AND a.deleted_at IS NULL
		WHERE c.owner_user_id = $1 AND c.account_id = $2
	`, ownerUserID, accountID)
	item, err := scanC2CProviderAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrC2CAccountNotFound
	}
	return item, err
}

type c2cScanner interface{ Scan(...any) error }

func scanC2CProviderAccount(row c2cScanner) (*C2CProviderAccount, error) {
	var item C2CProviderAccount
	var dailyLimit sql.NullFloat64
	var verifiedAt sql.NullTime
	if err := row.Scan(
		&item.AccountID, &item.Name, &item.Platform, &item.Type, &item.Status, &item.Schedulable,
		&item.PoolStatus, &item.VerificationStatus, &item.VerificationError,
		&dailyLimit, &item.DailyUsedUSD, &verifiedAt, &item.CreatedAt,
	); err != nil {
		return nil, err
	}
	if dailyLimit.Valid {
		item.DailyLimitUSD = &dailyLimit.Float64
	}
	if verifiedAt.Valid {
		item.VerifiedAt = &verifiedAt.Time
	}
	return &item, nil
}

func (s *C2CService) VerifyProviderAccount(ctx context.Context, ownerUserID, accountID int64, model string) (*C2CProviderAccount, error) {
	if _, err := s.GetProviderAccount(ctx, ownerUserID, accountID); err != nil {
		return nil, err
	}
	if s.tester == nil {
		return nil, infraerrors.New(http.StatusServiceUnavailable, "C2C_VERIFIER_UNAVAILABLE", "account verifier unavailable")
	}
	result, err := s.tester.RunTestBackground(ctx, accountID, strings.TrimSpace(model))
	if err != nil {
		return nil, err
	}
	if result == nil || result.Status != "success" {
		message := "upstream verification failed"
		if result != nil && strings.TrimSpace(result.ErrorMessage) != "" {
			message = result.ErrorMessage
		}
		_, updateErr := s.db.ExecContext(ctx, `
			UPDATE c2c_provider_accounts
			SET pool_status = 'rejected', verification_status = 'failed',
			    verification_error = $3, updated_at = NOW()
			WHERE owner_user_id = $1 AND account_id = $2
		`, ownerUserID, accountID, message)
		if updateErr != nil {
			return nil, updateErr
		}
		_, _ = s.accounts.SetAccountSchedulable(ctx, accountID, false)
		return s.GetProviderAccount(ctx, ownerUserID, accountID)
	}

	res, err := s.db.ExecContext(ctx, `
		UPDATE c2c_provider_accounts
		SET pool_status = 'active', verification_status = 'verified',
		    verification_error = NULL, verified_at = NOW(), updated_at = NOW()
		WHERE owner_user_id = $1 AND account_id = $2
	`, ownerUserID, accountID)
	if err != nil {
		return nil, err
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return nil, ErrC2CAccountNotFound
	}
	if _, err := s.accounts.SetAccountSchedulable(ctx, accountID, true); err != nil {
		_, _ = s.db.ExecContext(ctx, `UPDATE c2c_provider_accounts SET pool_status = 'paused', updated_at = NOW() WHERE account_id = $1`, accountID)
		return nil, err
	}
	return s.GetProviderAccount(ctx, ownerUserID, accountID)
}

func (s *C2CService) SetPoolState(ctx context.Context, ownerUserID, accountID int64, active bool) (*C2CProviderAccount, error) {
	item, err := s.GetProviderAccount(ctx, ownerUserID, accountID)
	if err != nil {
		return nil, err
	}
	if active && item.VerificationStatus != C2CVerificationVerified {
		return nil, infraerrors.BadRequest("C2C_ACCOUNT_NOT_VERIFIED", "account must pass verification before joining the pool")
	}
	target := C2CPoolStatusPaused
	if active {
		target = C2CPoolStatusActive
	}
	if !active {
		if _, err := s.accounts.SetAccountSchedulable(ctx, accountID, false); err != nil {
			return nil, err
		}
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE c2c_provider_accounts SET pool_status = $3, updated_at = NOW()
		WHERE owner_user_id = $1 AND account_id = $2
	`, ownerUserID, accountID, target)
	if err != nil {
		return nil, err
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return nil, ErrC2CAccountNotFound
	}
	if active {
		if _, err := s.accounts.SetAccountSchedulable(ctx, accountID, true); err != nil {
			_, _ = s.db.ExecContext(ctx, `UPDATE c2c_provider_accounts SET pool_status = 'paused', updated_at = NOW() WHERE account_id = $1`, accountID)
			return nil, err
		}
	}
	return s.GetProviderAccount(ctx, ownerUserID, accountID)
}

func (s *C2CService) DeleteProviderAccount(ctx context.Context, ownerUserID, accountID int64) error {
	if _, err := s.GetProviderAccount(ctx, ownerUserID, accountID); err != nil {
		return err
	}
	if _, err := s.accounts.SetAccountSchedulable(ctx, accountID, false); err != nil {
		return err
	}
	if err := s.accounts.DeleteAccount(ctx, accountID); err != nil {
		return err
	}
	// Accounts are soft-deleted by the upstream account service, so remove the
	// pool metadata explicitly. Historical trades and ledger entries retain the
	// seller/account snapshot they need and do not depend on this row.
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM c2c_provider_accounts
		WHERE owner_user_id = $1 AND account_id = $2
	`, ownerUserID, accountID)
	return err
}

func (s *C2CService) Wallet(ctx context.Context, userID int64) (*C2CWallet, error) {
	var wallet C2CWallet
	err := s.db.QueryRowContext(ctx, `
		SELECT u.balance, u.c2c_pending_balance,
		       COALESCE((SELECT SUM(t.buyer_charge) FROM c2c_trades t WHERE t.buyer_user_id = u.id AND t.status <> 'reversed'), 0),
		       COALESCE((SELECT SUM(t.seller_earning) FROM c2c_trades t WHERE t.seller_user_id = u.id AND t.status <> 'reversed'), 0),
		       COALESCE((SELECT SUM(t.platform_fee) FROM c2c_trades t WHERE t.seller_user_id = u.id AND t.status <> 'reversed'), 0)
		FROM users u WHERE u.id = $1 AND u.deleted_at IS NULL
	`, userID).Scan(&wallet.AvailableBalance, &wallet.PendingBalance, &wallet.LifetimeBought, &wallet.LifetimeEarned, &wallet.LifetimeFees)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return &wallet, err
}

func (s *C2CService) Stats(ctx context.Context, userID int64) (*C2CStats, error) {
	var stats C2CStats
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COALESCE((SELECT COUNT(*) FROM c2c_trades WHERE buyer_user_id = $1 AND status <> 'reversed'), 0),
			COALESCE((SELECT SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens) FROM c2c_trades WHERE buyer_user_id = $1 AND status <> 'reversed'), 0),
			COALESCE((SELECT SUM(buyer_charge) FROM c2c_trades WHERE buyer_user_id = $1 AND status <> 'reversed'), 0),
			COALESCE((SELECT COUNT(*) FROM c2c_trades WHERE seller_user_id = $1 AND status <> 'reversed'), 0),
			COALESCE((SELECT SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens) FROM c2c_trades WHERE seller_user_id = $1 AND status <> 'reversed'), 0),
			COALESCE((SELECT SUM(seller_earning) FROM c2c_trades WHERE seller_user_id = $1 AND status = 'pending'), 0),
			COALESCE((SELECT SUM(seller_earning) FROM c2c_trades WHERE seller_user_id = $1 AND status = 'settled'), 0),
			COALESCE((SELECT SUM(seller_earning) FROM c2c_trades WHERE seller_user_id = $1 AND status <> 'reversed'), 0)
	`, userID).Scan(
		&stats.BuyerRequestCount, &stats.BuyerTokenCount, &stats.BuyerSpend,
		&stats.SellerRequestCount, &stats.SellerTokenCount, &stats.SellerPending,
		&stats.SellerSettled, &stats.SellerEarnings,
	)
	return &stats, err
}

func (s *C2CService) Ledger(ctx context.Context, userID int64, limit int) ([]C2CLedgerEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, trade_id, wallet, entry_type, amount, created_at
		FROM c2c_ledger_entries
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]C2CLedgerEntry, 0)
	for rows.Next() {
		var item C2CLedgerEntry
		var tradeID sql.NullInt64
		if err := rows.Scan(&item.ID, &tradeID, &item.Wallet, &item.EntryType, &item.Amount, &item.CreatedAt); err != nil {
			return nil, err
		}
		if tradeID.Valid {
			item.TradeID = &tradeID.Int64
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *C2CService) MaturePending(ctx context.Context, limit int) (_ int, err error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()
	// Re-open active, verified accounts whose rolling contribution window has
	// elapsed. The outbox makes the scheduler snapshot observe the transition.
	if _, err := tx.ExecContext(ctx, `
		WITH reset AS (
			UPDATE c2c_provider_accounts c
			SET daily_used_usd = 0, daily_window_started_at = NOW(), updated_at = NOW()
			WHERE c.pool_status = 'active'
			  AND c.verification_status = 'verified'
			  AND c.daily_limit_usd IS NOT NULL
			  AND c.daily_window_started_at <= NOW() - INTERVAL '24 hours'
			RETURNING c.account_id
		), enabled AS (
			UPDATE accounts a SET schedulable = TRUE, updated_at = NOW()
			FROM reset r WHERE a.id = r.account_id AND a.deleted_at IS NULL
			RETURNING a.id
		)
		INSERT INTO scheduler_outbox (event_type, account_id)
		SELECT $1, id FROM enabled
	`, SchedulerOutboxEventAccountChanged); err != nil {
		return 0, err
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT id, seller_user_id, seller_earning
		FROM c2c_trades
		WHERE status = 'pending' AND matures_at <= NOW()
		ORDER BY matures_at, id
		FOR UPDATE SKIP LOCKED
		LIMIT $1
	`, limit)
	if err != nil {
		return 0, err
	}
	type matureTrade struct {
		id, sellerID int64
		earning      float64
	}
	trades := make([]matureTrade, 0)
	for rows.Next() {
		var trade matureTrade
		if err := rows.Scan(&trade.id, &trade.sellerID, &trade.earning); err != nil {
			_ = rows.Close()
			return 0, err
		}
		trades = append(trades, trade)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	for _, trade := range trades {
		res, err := tx.ExecContext(ctx, `
			UPDATE users
			SET c2c_pending_balance = c2c_pending_balance - $1,
			    balance = balance + $1, updated_at = NOW()
			WHERE id = $2 AND deleted_at IS NULL AND c2c_pending_balance >= $1
		`, trade.earning, trade.sellerID)
		if err != nil {
			return 0, err
		}
		if affected, _ := res.RowsAffected(); affected != 1 {
			return 0, fmt.Errorf("c2c settlement balance mismatch for trade %d", trade.id)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO c2c_ledger_entries (trade_id, user_id, wallet, entry_type, amount, idempotency_key)
			VALUES
				($1, $2, 'pending', 'seller_mature_out', (-$3::numeric), $4),
				($1, $2, 'available', 'seller_mature_in', $3::numeric, $5)
		`, trade.id, trade.sellerID, trade.earning,
			fmt.Sprintf("trade:%d:seller:mature:pending", trade.id),
			fmt.Sprintf("trade:%d:seller:mature:available", trade.id)); err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE c2c_trades SET status = 'settled', settled_at = NOW(), updated_at = NOW()
			WHERE id = $1 AND status = 'pending'
		`, trade.id); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	tx = nil
	return len(trades), nil
}

func validateC2CProviderInput(input *C2CCreateProviderAccountInput) error {
	input.Name = strings.TrimSpace(input.Name)
	input.Platform = strings.ToLower(strings.TrimSpace(input.Platform))
	input.Type = strings.ToLower(strings.TrimSpace(input.Type))
	if input.Name == "" || len(input.Name) > 100 {
		return infraerrors.BadRequest("C2C_INVALID_NAME", "name is required and must not exceed 100 characters")
	}
	allowedPlatform := false
	for _, platform := range AllowedQuotaPlatforms {
		if input.Platform == platform {
			allowedPlatform = true
			break
		}
	}
	if !allowedPlatform {
		return infraerrors.BadRequest("C2C_INVALID_PLATFORM", "unsupported provider platform")
	}
	if !c2cAccountTypeAllowed(input.Platform, input.Type) {
		return infraerrors.BadRequest("C2C_INVALID_ACCOUNT_TYPE", "unsupported account type for provider")
	}
	if len(input.Credentials) == 0 {
		return infraerrors.BadRequest("C2C_CREDENTIALS_REQUIRED", "credentials are required")
	}
	for key := range input.Credentials {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "base_url", "baseurl", "api_base", "endpoint", "url":
			return infraerrors.BadRequest("C2C_CUSTOM_ENDPOINT_FORBIDDEN", "custom upstream endpoints are not allowed")
		}
	}
	if input.DailyLimitUSD != nil && *input.DailyLimitUSD <= 0 {
		return infraerrors.BadRequest("C2C_INVALID_DAILY_LIMIT", "daily_limit_usd must be greater than zero")
	}
	return nil
}

func c2cAccountTypeAllowed(platform, accountType string) bool {
	switch platform {
	case PlatformKimi, PlatformZhipu, PlatformDeepseek:
		return accountType == AccountTypeAPIKey
	case PlatformAntigravity:
		return accountType == AccountTypeOAuth
	case PlatformOpenAI, PlatformGemini, PlatformGrok:
		return accountType == AccountTypeOAuth || accountType == AccountTypeAPIKey
	case PlatformAnthropic:
		return accountType == AccountTypeOAuth || accountType == AccountTypeSetupToken ||
			accountType == AccountTypeAPIKey || accountType == AccountTypeBedrock ||
			accountType == AccountTypeServiceAccount
	default:
		return false
	}
}

func applyC2COfficialEndpointDefaults(platform, accountMode, protocol string, credentials map[string]any) {
	if !IsCNProvider(platform) {
		return
	}
	mode := strings.ToLower(strings.TrimSpace(accountMode))
	if mode != AccountModeCoding {
		mode = AccountModePayG
	}
	proto := strings.ToLower(strings.TrimSpace(protocol))
	if proto == "" {
		proto = APIProtocolChatCompletions
	}
	credentials["account_mode"] = mode
	credentials["api_protocol"] = proto
	if proto == APIProtocolAnthropic {
		switch platform {
		case PlatformKimi:
			if mode == AccountModeCoding {
				credentials["base_url"] = DefaultKimiCodingAnthropicBaseURL
			} else {
				credentials["base_url"] = DefaultKimiPayGAnthropicBaseURL
			}
		case PlatformZhipu:
			credentials["base_url"] = DefaultZhipuAnthropicBaseURL
		case PlatformDeepseek:
			credentials["base_url"] = DefaultDeepseekAnthropicBaseURL
		}
		return
	}
	switch platform {
	case PlatformKimi:
		if mode == AccountModeCoding {
			credentials["base_url"] = DefaultKimiCodingBaseURL
		} else {
			credentials["base_url"] = DefaultKimiPayGBaseURL
		}
	case PlatformZhipu:
		if mode == AccountModeCoding {
			credentials["base_url"] = DefaultZhipuCodingBaseURL
		} else {
			credentials["base_url"] = DefaultZhipuPayGBaseURL
		}
	case PlatformDeepseek:
		credentials["base_url"] = DefaultDeepseekBaseURL
	}
}

func c2cCredentialIdentityHash(platform, accountType string, credentials map[string]any) (string, error) {
	payload, err := json.Marshal(credentials)
	if err != nil {
		return "", infraerrors.BadRequest("C2C_INVALID_CREDENTIALS", "credentials must be valid JSON")
	}
	sum := sha256.Sum256(append([]byte(strings.ToLower(platform)+"\x00"+strings.ToLower(accountType)+"\x00"), payload...))
	return hex.EncodeToString(sum[:]), nil
}

func cloneC2CJSONMap(source map[string]any) map[string]any {
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
