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
}

type storage struct {
    connection redis.Conn
}

func NewStorage(address string) (Storage, error) {
    c, err := redis.Dial("tcp", address)

    if err != nil {
        return nil, err
    }

    return &storage{
        c,
    }, nil
}

func (s *storage) Close() {
    s.connection.Close()
}

func (s *storage) Get(key string) (string, error) {
    log.Printf("Getting key '%s' from cache.", key)
    value, err := redis.String(s.connection.Do("GET", key))
    return value, err
}

func (s *storage) HasKey(key string) bool {
    exists, _ := redis.Bool(s.connection.Do("EXISTS", key))
    return exists
}

func (s *storage) Set(key, value string) {
    log.Printf("Setting key '%s' to cache.", key)
    s.connection.Do("SET", key, value)
}
