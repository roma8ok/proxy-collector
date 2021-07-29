package main

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	tableBlacklistDomains = "blacklist_domains"
	tableProxies          = "proxies"
)

func blacklistDomains(pool *pgxpool.Pool) (domains []string, err error) {
	s, _, err := sq.Select("domain").From(tableBlacklistDomains).ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(context.Background(), s)
	if rows != nil {
		if err := rows.Err(); err != nil {
			return nil, err
		}

		for rows.Next() {
			var d string
			if err := rows.Scan(&d); err != nil {
				return nil, err
			}
			domains = append(domains, d)
		}
	}

	return domains, nil
}

type proxy struct {
	ID        int
	URL       string
	Type      string
	Anonymous bool
	Created   time.Time
	LastCheck time.Time
}

func proxyExistInDB(pool *pgxpool.Pool, u string) bool {
	s, _, err := sq.Select("COUNT(*)").
		From(tableProxies).
		Where(sq.Eq{"url": u}).
		ToSql()
	if err != nil {
		return false
	}

	var counter int
	if err := pool.QueryRow(context.Background(), s).Scan(&counter); err != nil {
		return false
	}

	if counter > 0 {
		return true
	}
	return false
}

func insertProxyIntoDB(pool *pgxpool.Pool, p proxy) error {
	s, args, err := sq.
		Insert(tableProxies).
		Columns("url", "type", "anonymous", "created", "last_check").
		Values(p.URL, p.Type, p.Anonymous, p.Created, p.LastCheck).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = pool.Exec(context.Background(), s, args...)
	if err != nil {
		return err
	}

	return nil
}

func updateProxyInDB(pool *pgxpool.Pool, p proxy) error {
	s, args, err := sq.
		Update("proxies").
		Where(sq.Eq{"url": p.URL}).
		Set("type", p.Type).
		Set("anonymous", p.Anonymous).
		Set("last_check", p.LastCheck).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = pool.Exec(context.Background(), s, args...)
	if err != nil {
		return err
	}

	return nil
}

func saveProxyToDB(pool *pgxpool.Pool, p proxy) error {
	exist := proxyExistInDB(pool, p.URL)

	if exist {
		if err := updateProxyInDB(pool, p); err != nil {
			return err
		}
	} else {
		if err := insertProxyIntoDB(pool, p); err != nil {
			return err
		}
	}

	return nil
}

func freshProxies(pool *pgxpool.Pool) (proxies []proxy, err error) {
	s, args, err := sq.Select("id", "url", "type", "anonymous", "created", "last_check").
		From(tableProxies).
		Where(sq.GtOrEq{"last_check": time.Now().Add(-1 * time.Minute * 60)}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(context.Background(), s, args...)
	if rows != nil {
		if err := rows.Err(); err != nil {
			return nil, err
		}

		for rows.Next() {
			var p proxy
			if err := rows.Scan(&p.ID, &p.URL, &p.Type, &p.Anonymous, &p.Created, &p.LastCheck); err != nil {
				return nil, err
			}
			proxies = append(proxies, p)
		}
	}

	return proxies, nil
}
