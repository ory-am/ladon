package ladon

import (
	"encoding/json"

	"github.com/pkg/errors"
	"gopkg.in/redis.v5"
)

// RedisManager is a redis implementation of Manager to store policies persistently.
type RedisManager struct {
	db        *redis.Client
	keyPrefix string
}

// NewRedisManager initializes a new RedisManager with no policies
func NewRedisManager(db *redis.Client, keyPrefix string) *RedisManager {
	return &RedisManager{
		db:        db,
		keyPrefix: keyPrefix,
	}
}

const redisPolicies = "ladon:policies"

var redisNotFound = errors.New("Not found")

func (m *RedisManager) redisPoliciesKey() string {
	return m.keyPrefix + redisPolicies
}

// Create a new policy to RedisManager
func (m *RedisManager) Create(policy Policy) error {
	payload, err := json.Marshal(policy)
	if err != nil {
		return err
	}

	wasKeySet, err := m.db.HSetNX(m.redisPoliciesKey(), policy.GetID(), string(payload)).Result()
	if !wasKeySet {
		return errors.New("Policy exists")
	} else if err != nil {
		return err
	}

	return nil
}

// Get retrieves a policy.
func (m *RedisManager) Get(id string) (Policy, error) {
	resp, err := m.db.HGet(m.redisPoliciesKey(), id).Bytes()
	if err == redis.Nil {
		return nil, redisNotFound
	} else if err != nil {
		return nil, errors.Wrap(err, "")
	}

	return redisUnmarshalPolicy(resp)
}

// Delete removes a policy.
func (m *RedisManager) Delete(id string) error {
	return m.db.HDel(m.redisPoliciesKey(), id).Err()
}

// FindPoliciesForSubject finds all policies associated with the subject.
func (m *RedisManager) FindPoliciesForSubject(subject string) (Policies, error) {
	var ps Policies

	iter := m.db.HScan(m.redisPoliciesKey(), 0, "", 0).Iterator()
	for iter.Next() {
		if !iter.Next() {
			break
		}
		resp := []byte(iter.Val())

		p, err := redisUnmarshalPolicy(resp)
		if err != nil {
			return nil, err
		}

		if ok, err := Match(p, p.GetSubjects(), subject); err != nil {
			return nil, err
		} else if !ok {
			continue
		}

		ps = append(ps, p)
	}
	if err := iter.Err(); err != nil {
		return nil, errors.Wrap(err, "")
	}

	return ps, nil
}

func redisUnmarshalPolicy(policy []byte) (Policy, error) {
	var p *DefaultPolicy
	if err := json.Unmarshal(policy, &p); err != nil {
		return nil, errors.Wrap(err, "")
	}

	return p, nil
}
