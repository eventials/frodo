// Package storage implements a Redis based storage.
package storage

import (
    "log"
    "time"

    "github.com/garyburd/redigo/redis"
)

type Settings struct {
    Url string
    KeyTTL int
}

type Storage struct {
    settings Settings
    pool redis.Pool
}

func NewStorage(settings Settings) (*Storage, error) {
    pool := redis.Pool{
        MaxIdle: 3,
        MaxActive: 100,
        IdleTimeout: 120 * time.Second,
        Dial: func () (redis.Conn, error) {
            c, err := redis.DialURL(settings.Url)

            if err != nil {
                return nil, err
            }

            return c, err
        },
        TestOnBorrow: func(c redis.Conn, t time.Time) error {
            _, err := c.Do("PING")
            return err
        },
    }

    return &Storage{
        settings,
        pool,
    }, nil
}

func (s *Storage) Close() {
    s.pool.Close()
}

func (s *Storage) Get(key string) (string, error) {
    log.Printf("Getting key '%s' from cache.\n", key)

    conn := s.pool.Get()
    defer conn.Close()

    value, err := redis.String(conn.Do("GET", key))

    return value, err
}

func (s *Storage) HasKey(key string) bool {
    conn := s.pool.Get()
    defer conn.Close()

    exists, _ := redis.Bool(conn.Do("EXISTS", key))

    return exists
}

func (s *Storage) Set(key, value string) {
    log.Printf("Setting key '%s' to cache.\n", key)

    conn := s.pool.Get()
    defer conn.Close()

    if s.settings.KeyTTL == 0 {
        conn.Do("SET", key, value)
    } else {
        conn.Do("SETEX", key, s.settings.KeyTTL, value)
    }
}

func (s *Storage) Ping() bool {
    conn := s.pool.Get()
    defer conn.Close()

    if err := conn.Err(); err != nil {
        log.Printf("Ping error: %s\n", err)
        return false
    }

    value, err := redis.String(conn.Do("PING"))

    if err != nil {
        log.Printf("Ping error: %s\n", err)
    }

    return err == nil && value == "PONG"
}

func (s *Storage) ConnectionCount() int {
    return s.pool.ActiveCount()
}
