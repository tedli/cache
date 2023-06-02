package cache

import (
	"context"
	"net/http"
	"testing"
	"time"
)

/*
   cache_test.go:12: get
   cache_test.go:31: status: 200
   cache_test.go:36: cached status: 200
   cache_test.go:12: get
   cache_test.go:42: 2nd time status: 200
*/

func TestE2E(t *testing.T) {
	cc := BuildCache[string, int](WithGetter(func(ctx context.Context, key string) (int, error) {
		t.Logf("get")
		if resp, err := http.Get(key); err != nil {
			return 0, err
		} else {
			return resp.StatusCode, nil
		}
	}), WithTTL[string, int](10*time.Second), WithBehaviour[string, int](RefreshBehaviourRemoveOnly),
		WithRefreshInterval[string, int](20*time.Second), WithOnRefreshError[string, int](
			func(key string, err error, remove func()) {
				t.Logf("refresh failed, key: %s, error: %#v", key, err)
				remove()
			}))
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
