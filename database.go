package main

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4/pgxpool"
)

const tableProxies = "proxies"

type Proxy struct {
	ID        int
	Active    bool
	URL       string
	HTTP      bool
	HTTPS     bool
	SOCKS5    bool
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

func insertProxyIntoDB(pool *pgxpool.Pool, p Proxy) error {
	s, args, err := sq.
		Insert(tableProxies).
		Columns("url", "active", "http", "https", "socks5", "anonymous", "created", "last_check").
		Values(p.URL, p.Active, p.HTTP, p.HTTPS, p.SOCKS5, p.Anonymous, p.Created, p.LastCheck).
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

func updateProxyInDB(pool *pgxpool.Pool, p Proxy) error {
	s, args, err := sq.
		Update("proxies").
		Where(sq.Eq{"url": p.URL}).
		Set("active", p.Active).
		Set("http", p.HTTP).
		Set("https", p.HTTPS).
		Set("socks5", p.SOCKS5).
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

func saveProxyToDB(pool *pgxpool.Pool, p Proxy) error {
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

func changeProxyToInactive(pool *pgxpool.Pool, pURL string) error {
	s, args, err := sq.
		Update("proxies").
		Where(sq.Eq{"url": pURL}).
		Set("active", false).
		Set("last_check", time.Now()).
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

func freshProxies(pool *pgxpool.Pool, duration time.Duration) (proxies []Proxy, err error) {
	s, args, err := sq.Select("id", "url", "active", "http", "https", "socks5", "anonymous", "created", "last_check").
		From(tableProxies).
		Where(sq.GtOrEq{"last_check": time.Now().Add(-1 * duration)}).
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
			var p Proxy
			if err := rows.Scan(&p.ID, &p.URL, &p.Active, &p.HTTP, &p.HTTPS, &p.SOCKS5, &p.Anonymous, &p.Created, &p.LastCheck); err != nil {
				return nil, err
			}
			proxies = append(proxies, p)
		}
	}

	return proxies, nil
}

func proxiesFromDB(pool *pgxpool.Pool, active bool) (proxies []Proxy, err error) {
	s, args, err := sq.Select("id", "url", "active", "http", "https", "socks5", "anonymous", "created", "last_check").
		From(tableProxies).
		Where(sq.Eq{"active": active}).
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
			var p Proxy
			if err := rows.Scan(&p.ID, &p.URL, &p.Active, &p.HTTP, &p.HTTPS, &p.SOCKS5, &p.Anonymous, &p.Created, &p.LastCheck); err != nil {
				return nil, err
			}
			proxies = append(proxies, p)
		}
	}

	return proxies, nil
}
