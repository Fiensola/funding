CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS funding_rates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    exchange VARCHAR(50) NOT NULL,
    symbol VARCHAR(50) NOT NULL,
    rate DOUBLE PRECISION NOT NULL,
    price DOUBLE PRECISION NULL,
    timestamp TIMESTAMP NOT NULL,
    next_funding TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_exchange_symbol ON funding_rates (exchange, symbol);
CREATE INDEX IF NOT EXISTS idx_timestamp ON funding_rates (timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_created_at ON funding_rates (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_latest_rates ON funding_rates (exchange, symbol, timestamp DESC);