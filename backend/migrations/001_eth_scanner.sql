CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    google_sub TEXT,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL DEFAULT '',
    avatar_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    gmail_enabled BOOLEAN NOT NULL DEFAULT true,
    min_severity TEXT NOT NULL DEFAULT 'info',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS watchlists (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    address TEXT NOT NULL,
    alias TEXT,
    min_interaction_eth NUMERIC(38, 18) NOT NULL DEFAULT 500,
    notification_on BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, address)
);

CREATE TABLE IF NOT EXISTS whale_imports (
    id TEXT PRIMARY KEY,
    filename TEXT NOT NULL,
    source TEXT NOT NULL,
    row_count INTEGER NOT NULL,
    skipped_count INTEGER NOT NULL DEFAULT 0,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS address_balances (
    address TEXT PRIMARY KEY,
    rank INTEGER,
    balance_wei NUMERIC(78, 0),
    balance_eth NUMERIC(38, 18) NOT NULL,
    percentage TEXT,
    txn_count INTEGER NOT NULL DEFAULT 0,
    source TEXT NOT NULL,
    evidence_ref TEXT,
    observed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_address_balances_balance_eth
    ON address_balances (balance_eth DESC);

CREATE TABLE IF NOT EXISTS address_labels (
    address TEXT NOT NULL,
    category TEXT NOT NULL,
    name TEXT NOT NULL,
    source TEXT NOT NULL,
    confidence NUMERIC(5, 4) NOT NULL,
    evidence_ref TEXT,
    heuristic BOOLEAN NOT NULL DEFAULT false,
    last_checked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (address, category, source)
);

CREATE TABLE IF NOT EXISTS transactions (
    hash TEXT NOT NULL,
    block_number BIGINT,
    ts TIMESTAMPTZ,
    from_address TEXT,
    to_address TEXT,
    value_wei NUMERIC(78, 0),
    value_eth NUMERIC(38, 18),
    asset TEXT NOT NULL DEFAULT 'ETH',
    category TEXT NOT NULL,
    source TEXT NOT NULL,
    PRIMARY KEY (hash, category)
);

CREATE INDEX IF NOT EXISTS idx_transactions_from_ts ON transactions (from_address, ts DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_to_ts ON transactions (to_address, ts DESC);

CREATE TABLE IF NOT EXISTS price_ticks (
    asset TEXT NOT NULL,
    interval TEXT NOT NULL,
    ts TIMESTAMPTZ NOT NULL,
    open NUMERIC(20, 8) NOT NULL,
    high NUMERIC(20, 8) NOT NULL,
    low NUMERIC(20, 8) NOT NULL,
    close NUMERIC(20, 8) NOT NULL,
    source TEXT NOT NULL,
    PRIMARY KEY (asset, interval, ts)
);

CREATE TABLE IF NOT EXISTS ml_scores (
    address TEXT PRIMARY KEY,
    score INTEGER NOT NULL,
    model_version TEXT NOT NULL,
    features_hash TEXT NOT NULL,
    explanation_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    scored_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS alerts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    address TEXT NOT NULL,
    type TEXT NOT NULL,
    severity TEXT NOT NULL,
    threshold_eth NUMERIC(38, 18),
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    evidence_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    labels_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    confidence NUMERIC(5, 4) NOT NULL,
    heuristic BOOLEAN NOT NULL DEFAULT true,
    status TEXT NOT NULL DEFAULT 'new',
    dedupe_key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS notification_logs (
    id TEXT PRIMARY KEY,
    alert_id TEXT NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel TEXT NOT NULL,
    provider_message_id TEXT,
    status TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMPTZ,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'queued',
    run_after TIMESTAMPTZ NOT NULL DEFAULT now(),
    locked_at TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    actor_user_id TEXT,
    action TEXT NOT NULL,
    target TEXT,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS source_health_checks (
    source TEXT PRIMARY KEY,
    status TEXT NOT NULL,
    message TEXT,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
