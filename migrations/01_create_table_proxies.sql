CREATE TYPE proxy_type as ENUM ('http', 'https', 'socks');

CREATE TABLE IF NOT EXISTS proxies
(
    id         SERIAL PRIMARY KEY,
    url        VARCHAR(256) NOT NULL UNIQUE,
    active     BOOLEAN      NOT NULL,
    http       BOOLEAN      NOT NULL,
    https      BOOLEAN      NOT NULL,
    socks5     BOOLEAN      NOT NULL,
    anonymous  BOOLEAN      NOT NULL,
    created    TIMESTAMPTZ  NOT NULL,
    last_check TIMESTAMPTZ  NOT NULL
)
