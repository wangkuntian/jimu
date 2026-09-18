package auth

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type fakeRedis struct {
	mu         sync.Mutex
	err        error
	limitCount map[string]int
	hashes     map[string]map[string]string
	sets       map[string]map[string]struct{}
}

func newFakeRedis() *fakeRedis {
	return &fakeRedis{
		limitCount: make(map[string]int),
		hashes:     make(map[string]map[string]string),
		sets:       make(map[string]map[string]struct{}),
	}
}

func (r *fakeRedis) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	return r.eval(keys, args...)
}

func (r *fakeRedis) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd {
	return r.eval(keys, args...)
}

func (r *fakeRedis) EvalRO(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	return r.eval(keys, args...)
}

func (r *fakeRedis) EvalShaRO(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd {
	return r.eval(keys, args...)
}

func (r *fakeRedis) ScriptExists(ctx context.Context, hashes ...string) *redis.BoolSliceCmd {
	exists := make([]bool, len(hashes))
	for i := range exists {
		exists[i] = true
	}
	return redis.NewBoolSliceResult(exists, nil)
}

func (r *fakeRedis) ScriptLoad(ctx context.Context, script string) *redis.StringCmd {
	return redis.NewStringResult("fake-script", nil)
}

func (r *fakeRedis) TxPipeline() sessionPipeline {
	return &fakePipeline{redis: r}
}

func (r *fakeRedis) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	r.mu.Lock()
	defer r.mu.Unlock()
	members := make([]string, 0, len(r.sets[key]))
	for member := range r.sets[key] {
		members = append(members, member)
	}
	sort.Strings(members)
	return redis.NewStringSliceResult(members, r.err)
}

func (r *fakeRedis) keys() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	keys := make([]string, 0, len(r.limitCount)+len(r.hashes)+len(r.sets))
	for key := range r.limitCount {
		keys = append(keys, key)
	}
	for key := range r.hashes {
		keys = append(keys, key)
	}
	for key := range r.sets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (r *fakeRedis) eval(keys []string, args ...interface{}) *redis.Cmd {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return redis.NewCmdResult(nil, r.err)
	}
	switch len(args) {
	case 2:
		return r.evalLimiter(keys, args...)
	case 4:
		return r.evalSessionRotate(keys, args...)
	default:
		return redis.NewCmdResult(nil, fmt.Errorf("unexpected script args: %d", len(args)))
	}
}

func (r *fakeRedis) evalLimiter(keys []string, args ...interface{}) *redis.Cmd {
	if len(keys) != 1 {
		return redis.NewCmdResult(nil, fmt.Errorf("unexpected limiter keys: %d", len(keys)))
	}
	limit, err := toFakeInt(args[1])
	if err != nil {
		return redis.NewCmdResult(nil, err)
	}
	r.limitCount[keys[0]]++
	if r.limitCount[keys[0]] > limit {
		return redis.NewCmdResult(int64(0), nil)
	}
	return redis.NewCmdResult(int64(1), nil)
}

func (r *fakeRedis) evalSessionRotate(keys []string, args ...interface{}) *redis.Cmd {
	if len(keys) != 2 {
		return redis.NewCmdResult(nil, fmt.Errorf("unexpected session keys: %d", len(keys)))
	}
	key := keys[0]
	expectedUserID := fmt.Sprint(args[0])
	oldTokenID := fmt.Sprint(args[1])
	newTokenID := fmt.Sprint(args[2])

	session := r.hashes[key]
	if session == nil {
		return redis.NewCmdResult(int64(0), nil)
	}
	if session["user_id"] != expectedUserID {
		return redis.NewCmdResult(int64(0), nil)
	}
	current := session["token_id"]
	if current == "" {
		return redis.NewCmdResult(int64(0), nil)
	}
	if current != oldTokenID {
		return redis.NewCmdResult(int64(-1), nil)
	}
	session["token_id"] = newTokenID
	return redis.NewCmdResult(int64(1), nil)
}

func toFakeInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return strconv.Atoi(fmt.Sprint(value))
	}
}

type fakePipeline struct {
	redis *fakeRedis
	ops   []func()
}

func (p *fakePipeline) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	entries := make(map[string]string, len(values)/2)
	for i := 0; i+1 < len(values); i += 2 {
		entries[fmt.Sprint(values[i])] = fmt.Sprint(values[i+1])
	}
	p.ops = append(p.ops, func() {
		if p.redis.hashes[key] == nil {
			p.redis.hashes[key] = make(map[string]string)
		}
		for field, value := range entries {
			p.redis.hashes[key][field] = value
		}
	})
	return redis.NewIntResult(int64(len(entries)), nil)
}

func (p *fakePipeline) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	return redis.NewBoolResult(true, nil)
}

func (p *fakePipeline) SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	values := make([]string, 0, len(members))
	for _, member := range members {
		values = append(values, fmt.Sprint(member))
	}
	p.ops = append(p.ops, func() {
		if p.redis.sets[key] == nil {
			p.redis.sets[key] = make(map[string]struct{})
		}
		for _, value := range values {
			p.redis.sets[key][value] = struct{}{}
		}
	})
	return redis.NewIntResult(int64(len(values)), nil)
}

func (p *fakePipeline) SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	values := make([]string, 0, len(members))
	for _, member := range members {
		values = append(values, fmt.Sprint(member))
	}
	p.ops = append(p.ops, func() {
		for _, value := range values {
			delete(p.redis.sets[key], value)
		}
	})
	return redis.NewIntResult(int64(len(values)), nil)
}

func (p *fakePipeline) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	copied := append([]string(nil), keys...)
	p.ops = append(p.ops, func() {
		for _, key := range copied {
			delete(p.redis.hashes, key)
			delete(p.redis.sets, key)
			delete(p.redis.limitCount, key)
		}
	})
	return redis.NewIntResult(int64(len(keys)), nil)
}

func (p *fakePipeline) Exec(ctx context.Context) ([]redis.Cmder, error) {
	p.redis.mu.Lock()
	defer p.redis.mu.Unlock()
	if p.redis.err != nil {
		return nil, p.redis.err
	}
	for _, op := range p.ops {
		op()
	}
	return nil, nil
}
