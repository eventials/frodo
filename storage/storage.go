// Package storage implements a Redis based storage.
package storage

import (
    "log"

    "github.com/garyburd/redigo/redis"
)

type Storage interface {
    Close()
    Get(key string) (string, error)
    HasKey(key string) bool
    Set(key, value string)
    Ping() bool
}

type storage struct {
    connection redis.Conn
    keyTTL int
}

func NewStorage(address string, keyTTL int) (Storage, error) {
    c, err := redis.DialURL(address)

    if err != nil {
        return nil, err
    }

    return &storage{
        c,
        keyTTL,
    }, nil
}

func (s *storage) Close() {
    s.connection.Close()
}

func (s *storage) Get(key string) (string, error) {
    log.Printf("Getting key '%s' from cache.\n", key)
    value, err := redis.String(s.connection.Do("GET", key))
    return value, err
}

func (s *storage) HasKey(key string) bool {
    exists, _ := redis.Bool(s.connection.Do("EXISTS", key))
    return exists
}

func (s *storage) Set(key, value string) {
    log.Printf("Setting key '%s' to cache.\n", key)

    if s.keyTTL == 0 {
        s.connection.Do("SET", key, value)
    } else {
        s.connection.Do("SETEX", key, s.keyTTL, value)
    }
}

func (s *storage) Ping() bool {
    if err := s.connection.Err(); err != nil {
        log.Printf("Ping error: %s\n", err)
        return false
    }

    value, err := redis.String(s.connection.Do("PING"))

    if err != nil {
        log.Printf("Ping error: %s\n", err)
    }

    return err == nil && value == "PONG"
}
