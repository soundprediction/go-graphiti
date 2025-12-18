package cache

import (
	"errors"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v4"
)

var (
	// ErrKeyNotFound is returned when a key is not found in the cache
	ErrKeyNotFound = errors.New("key not found in cache")
)

// Cache interface defines the standard caching operations
type Cache interface {
	// Set stores a value with a TTL
	Set(key string, value []byte, ttl time.Duration) error
	// Get retrieves a value
	Get(key string) ([]byte, error)
	// Delete removes a value
	Delete(key string) error
	// Close closes the cache
	Close() error
}

// BadgerCache implements Cache using BadgerDB
type BadgerCache struct {
	db *badger.DB
}

// NewBadgerCache creates a new BadgerDB-backed cache
func NewBadgerCache(path string) (*BadgerCache, error) {
	opts := badger.DefaultOptions(path)
	// Optimize for smaller memory usage in default config
	opts.Logger = nil // Disable default logger to reduce noise

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger db: %w", err)
	}

	return &BadgerCache{
		db: db,
	}, nil
}

// Set stores a value with a TTL
func (c *BadgerCache) Set(key string, value []byte, ttl time.Duration) error {
	return c.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), value).WithTTL(ttl)
		return txn.SetEntry(e)
	})
}

// Get retrieves a value
func (c *BadgerCache) Get(key string) ([]byte, error) {
	var val []byte
	err := c.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		val, err = item.ValueCopy(nil)
		return err
	})

	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, ErrKeyNotFound
		}
		return nil, err
	}

	return val, nil
}

// Delete removes a value
func (c *BadgerCache) Delete(key string) error {
	return c.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// Close closes the cache
func (c *BadgerCache) Close() error {
	return c.db.Close()
}
