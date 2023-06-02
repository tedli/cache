package main

import (
	"context"
	"fmt"
	"github.com/tedli/cache"
	"net/http"
	"os"
)

func main() {
	cc := cache.BuildCache[string, int](cache.WithGetter(func(ctx context.Context, key string) (int, error) {
		if resp, err := http.Get(key); err != nil {
			return 0, err
		} else {
			return resp.StatusCode, nil
		}
	}))
	ctx := context.Background()
	url := "https://www.github.com"
	if status, err := cc.Get(ctx, url); err != nil {
		fmt.Fprintf(os.Stderr, "url: %s, error: %#v\n", url, err)
	} else {
		fmt.Fprintf(os.Stdout, "url: %s, status: %d\n", url, status)
	}
}
