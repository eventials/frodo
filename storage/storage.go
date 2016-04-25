// Package storage implements a Redis based storage.
package storage

import (
    "log"
    "time"

    "github.com/garyburd/redigo/redis"
)

type Storage interface {
    Close()
    Get(key string) (string, error)
    HasKey(key string) bool
    Set(key, value string)
    Ping() bool
    ConnectionCount() int
}

type storage struct {
    pool redis.Pool
    keyTTL int
}

func NewStorage(address string, keyTTL int) (Storage, error) {
    pool := redis.Pool{
        MaxIdle: 3,
        MaxActive: 100,
        IdleTimeout: 120 * time.Second,
        Dial: func () (redis.Conn, error) {
            c, err := redis.DialURL(address)

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

    return &storage{
        pool,
        keyTTL,
    }, nil
}

func (s *storage) Close() {
    s.pool.Close()
}

func (s *storage) Get(key string) (string, error) {
    log.Printf("Getting key '%s' from cache.\n", key)

    conn := s.pool.Get()
    defer conn.Close()

    value, err := redis.String(conn.Do("GET", key))

    return value, err
}

func (s *storage) HasKey(key string) bool {
    conn := s.pool.Get()
    defer conn.Close()

    exists, _ := redis.Bool(conn.Do("EXISTS", key))

    return exists
}

func (s *storage) Set(key, value string) {
    log.Printf("Setting key '%s' to cache.\n", key)

    conn := s.pool.Get()
    defer conn.Close()

    if s.keyTTL == 0 {
        conn.Do("SET", key, value)
    } else {
        conn.Do("SETEX", key, s.keyTTL, value)
    }
}

func (s *storage) Ping() bool {
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

func (s *storage) ConnectionCount() int {
    return s.pool.ActiveCount()
}
