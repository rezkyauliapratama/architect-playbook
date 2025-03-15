CREATE TABLE IF NOT EXISTS transactions (
    id String ,
    amount Decimal(18, 2),
    currency String,
    status Enum('pending' = 1, 'completed' = 2, 'failed' = 3),
    user_id String,
    merchant_id String,
    payment_method String,
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY user_id
SETTINGS index_granularity=8192;
