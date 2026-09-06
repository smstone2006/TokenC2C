-- C2C provider-account ownership and double-entry settlement ledger.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS c2c_pending_balance NUMERIC(20,8) NOT NULL DEFAULT 0;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'users_c2c_pending_balance_nonnegative'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT users_c2c_pending_balance_nonnegative
            CHECK (c2c_pending_balance >= 0);
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS c2c_provider_accounts (
    account_id BIGINT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    owner_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    pool_status VARCHAR(20) NOT NULL DEFAULT 'verifying',
    verification_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    verification_error TEXT,
    identity_hash VARCHAR(64) NOT NULL UNIQUE,
    daily_limit_usd NUMERIC(20,8),
    daily_used_usd NUMERIC(20,8) NOT NULL DEFAULT 0,
    daily_window_started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT c2c_provider_accounts_pool_status_check
        CHECK (pool_status IN ('verifying', 'active', 'paused', 'rejected')),
    CONSTRAINT c2c_provider_accounts_verification_status_check
        CHECK (verification_status IN ('pending', 'verified', 'failed')),
    CONSTRAINT c2c_provider_accounts_daily_limit_check
        CHECK (daily_limit_usd IS NULL OR daily_limit_usd > 0),
    CONSTRAINT c2c_provider_accounts_daily_used_check
        CHECK (daily_used_usd >= 0)
);

CREATE INDEX IF NOT EXISTS idx_c2c_provider_accounts_owner
    ON c2c_provider_accounts(owner_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_c2c_provider_accounts_pool
    ON c2c_provider_accounts(pool_status, verification_status, account_id);

CREATE TABLE IF NOT EXISTS c2c_trades (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(255) NOT NULL,
    api_key_id BIGINT NOT NULL REFERENCES api_keys(id) ON DELETE RESTRICT,
    buyer_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    seller_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    model VARCHAR(255) NOT NULL DEFAULT '',
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    buyer_charge NUMERIC(20,8) NOT NULL,
    seller_earning NUMERIC(20,8) NOT NULL,
    platform_fee NUMERIC(20,8) NOT NULL,
    seller_share_bps INTEGER NOT NULL DEFAULT 9000,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    matures_at TIMESTAMPTZ NOT NULL,
    settled_at TIMESTAMPTZ,
    reversed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT c2c_trades_request_key UNIQUE (request_id, api_key_id),
    CONSTRAINT c2c_trades_amounts_check CHECK (
        buyer_charge >= 0 AND seller_earning >= 0 AND platform_fee >= 0
        AND buyer_charge = seller_earning + platform_fee
    ),
    CONSTRAINT c2c_trades_share_check CHECK (seller_share_bps BETWEEN 0 AND 10000),
    CONSTRAINT c2c_trades_status_check CHECK (status IN ('pending', 'settled', 'reversed'))
);

CREATE INDEX IF NOT EXISTS idx_c2c_trades_seller_status_matures
    ON c2c_trades(seller_user_id, status, matures_at);
CREATE INDEX IF NOT EXISTS idx_c2c_trades_buyer_created
    ON c2c_trades(buyer_user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS c2c_ledger_entries (
    id BIGSERIAL PRIMARY KEY,
    trade_id BIGINT REFERENCES c2c_trades(id) ON DELETE RESTRICT,
    user_id BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    wallet VARCHAR(20) NOT NULL,
    entry_type VARCHAR(40) NOT NULL,
    amount NUMERIC(20,8) NOT NULL,
    idempotency_key VARCHAR(320) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT c2c_ledger_entries_wallet_check
        CHECK (wallet IN ('available', 'pending', 'platform')),
    CONSTRAINT c2c_ledger_entries_nonzero_check CHECK (amount <> 0),
    CONSTRAINT c2c_ledger_entries_owner_check CHECK (
        (wallet = 'platform' AND user_id IS NULL)
        OR (wallet <> 'platform' AND user_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_c2c_ledger_entries_user_created
    ON c2c_ledger_entries(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_c2c_ledger_entries_trade
    ON c2c_ledger_entries(trade_id);

COMMENT ON COLUMN users.c2c_pending_balance IS
    'C2C seller earnings waiting for the 24-hour maturity window';
COMMENT ON TABLE c2c_provider_accounts IS
    'Ownership, verification, and pool state for user-contributed upstream accounts';
COMMENT ON TABLE c2c_trades IS
    'Immutable per-request buyer/seller/platform split, settled after maturity';
COMMENT ON TABLE c2c_ledger_entries IS
    'Append-only signed C2C wallet movements with idempotency keys';
