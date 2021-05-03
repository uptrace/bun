package bun

import (
	"github.com/uptrace/bun/schema"
	"github.com/uptrace/bun/sqlfmt"
)

type (
	Safe  = sqlfmt.Safe
	Ident = sqlfmt.Ident
)

type (
	BeforeScanHook   = schema.BeforeScanHook
	AfterScanHook    = schema.AfterScanHook
	AfterSelectHook  = schema.AfterSelectHook
	BeforeInsertHook = schema.BeforeInsertHook
	AfterInsertHook  = schema.AfterInsertHook
	BeforeUpdateHook = schema.BeforeUpdateHook
	AfterUpdateHook  = schema.AfterUpdateHook
	BeforeDeleteHook = schema.BeforeDeleteHook
	AfterDeleteHook  = schema.AfterDeleteHook
)

type BaseTable struct{}
