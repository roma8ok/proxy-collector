package main

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4/pgxpool"
)

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

// SQLBuilder is a struct for sql statement.
type SQLBuilder struct {
	table string
}

// newSQLBuilder gets the table name and returns SQLBuilder.
func newSQLBuilder(table string) SQLBuilder {
	return SQLBuilder{table: table}
}

type WhereCondition map[string]interface{}

// count gets WhereCondition and returns SELECT COUNT sql statement, sql args and an error.
// count gets only "equal" (whereEq) and "greater or equal" (whereGtOrEq) conditions.
func (s *SQLBuilder) count(whereEq, whereGtOrEq WhereCondition) (string, []interface{}, error) {
	selectBuilder := sq.Select("COUNT(*)").From(s.table)

	if len(whereEq) > 0 && len(whereGtOrEq) > 0 {
		selectBuilder = selectBuilder.Where(sq.And{
			sq.Eq(whereEq),
			sq.GtOrEq(whereGtOrEq),
		})
	} else if len(whereEq) > 0 {
		selectBuilder = selectBuilder.Where(sq.Eq(whereEq))
	} else if len(whereGtOrEq) > 0 {
		selectBuilder = selectBuilder.Where(sq.GtOrEq(whereGtOrEq))
	}

	return selectBuilder.PlaceholderFormat(sq.Dollar).ToSql()
}

// insert gets data and returns INSERT sql statement, sql args and an error.
// Example data:
//  map[string]interface{}{"column_name": "value"}
func (s *SQLBuilder) insert(data map[string]interface{}) (string, []interface{}, error) {
	if len(data) == 0 {
		return "", nil, errDBInsertEmptyData
	}

	cols := make([]string, 0)
	values := make([]interface{}, 0)
	for k, v := range data {
		cols = append(cols, k)
		values = append(values, v)
	}

	return sq.
		Insert(s.table).
		Columns(cols...).
		Values(values...).
		PlaceholderFormat(sq.Dollar).
		ToSql()
}

func proxyExistInDB(pool *pgxpool.Pool, hostPort string) (bool, error) {
	sb := newSQLBuilder(tableProxies)
	s, args, err := sb.count(WhereCondition{"url": hostPort}, WhereCondition{})
	if err != nil {
		return false, err
	}

	var counter int
	if err = pool.QueryRow(context.Background(), s, args...).Scan(&counter); err != nil {
		return false, err
	}

	if counter != 0 {
		return true, nil
	}
	return false, nil
}

func insertProxyIntoDB(pool *pgxpool.Pool, p Proxy) error {
	sb := newSQLBuilder(tableProxies)
	s, args, err := sb.insert(map[string]interface{}{
		"url":        p.URL,
		"active":     p.Active,
		"http":       p.HTTP,
		"https":      p.HTTPS,
		"socks5":     p.SOCKS5,
		"anonymous":  p.Anonymous,
		"created":    p.Created,
		"last_check": p.LastCheck,
	})
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

func saveProxyToDB(pool *pgxpool.Pool, p Proxy) (successType string, err error) {
	exist, err := proxyExistInDB(pool, p.URL)
	if err != nil {
		return "", err
	}

	if exist {
		if err := updateProxyInDB(pool, p); err != nil {
			return "", err
		}
		successType = "update"
	} else {
		if err := insertProxyIntoDB(pool, p); err != nil {
			return "", err
		}
		successType = "insert"
	}

	return successType, nil
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
