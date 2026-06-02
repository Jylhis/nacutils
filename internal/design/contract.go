package design

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed contract.v0.4.0.json
var contractData []byte

type Tone struct {
	Light string `json:"light"`
	Dark  string `json:"dark"`
}

type Typography struct {
	TUIFallback string `json:"tuiFallback"`
}

type ANSIColor struct {
	Name  string `json:"name"`
	Light string `json:"light"`
	Dark  string `json:"dark"`
}

type Contract struct {
	Source     string `json:"source"`
	Tag        string `json:"tag"`
	TokensMeta struct {
		Version string `json:"version"`
	} `json:"tokensMeta"`
	Typography Typography      `json:"typography"`
	Palette    map[string]Tone `json:"palette"`
	Status     map[string]Tone `json:"status"`
	ANSI       []ANSIColor     `json:"ansi"`
}

var (
	loadOnce       sync.Once
	cachedContract Contract
	cachedErr      error
)

func Load() (Contract, error) {
	loadOnce.Do(func() {
		if err := json.Unmarshal(contractData, &cachedContract); err != nil {
			cachedErr = fmt.Errorf("parse design contract: %w", err)
			return
		}
		if cachedContract.Tag == "" {
			cachedErr = fmt.Errorf("parse design contract: missing tag")
		}
	})
	return cachedContract, cachedErr
}

func NormalizeHex(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
