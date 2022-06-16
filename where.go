package bun

import (
	"strings"
)

type WhereFields struct {
	Fields []string
}

func (wf *WhereFields) addWhereField(query string) {
	f := strings.Split(query, " ")
	wf.Fields = append(wf.Fields, f[0])
}

func (wf *WhereFields) GetWhereFields() []string {
	return wf.Fields
}
