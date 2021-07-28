CREATE TABLE IF NOT EXISTS blacklist_domains
(
    id     SERIAL PRIMARY KEY,
    domain VARCHAR(256) NOT NULL
)
