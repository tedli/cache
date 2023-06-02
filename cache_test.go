package cache

import (
	"context"
	"net/http"
	"testing"
	"time"
)

/*
=== RUN   TestE2E
    cache_test.go:20: get
    cache_test.go:37: status: 200
    cache_test.go:42: cached status: 200
    cache_test.go:20: get
    cache_test.go:48: 2nd time status: 200
--- PASS: TestE2E (30.00s)
PASS
*/

func TestE2E(t *testing.T) {
	cc := BuildCache[string, int](WithGetter(func(ctx context.Context, key string) (int, error) {
		t.Logf("get")
		if resp, err := http.Get(key); err != nil {
			return 0, err
		} else {
			return resp.StatusCode, nil
		}
	}), WithOnRefreshError(func(key string, prevStatus int, err error, remove func()) {
		t.Logf("refresh failed, key: %s, status: %d, error: %#v", key, prevStatus, err)
		remove()
	}))(WithTTL(10*time.Second), WithBehaviour(RefreshBehaviourRemoveOnly), WithRefreshInterval(20*time.Second))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	go cc.Start(ctx)
	url := "https://www.github.com"
	if status, err := cc.Get(ctx, url); err != nil {
		t.Errorf("get failed")
	} else {
		t.Logf("status: %d", status)
	}
	if status, err := cc.Get(ctx, url); err != nil {
		t.Errorf("get from cache failed")
	} else {
		t.Logf("cached status: %d", status)
	}
	<-time.After(21 * time.Second)
	if status, err := cc.Get(ctx, url); err != nil {
		t.Errorf("get again failed")
	} else {
		t.Logf("2nd time status: %d", status)
	}
	<-ctx.Done()
}
