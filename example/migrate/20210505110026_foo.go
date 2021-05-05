package main

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	migrator.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] ")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [down migration] ")
		return nil
	})
}
