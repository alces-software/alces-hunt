// Copyright (C) 2026 Alces Software Ltd.
//
// This program and the accompanying materials are made available under
// the terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0
//
// SPDX-License-Identifier: EPL-2.0

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const envPrefix = "ALCES_HUNT_"

// Config holds all runtime settings. Every scalar key may be overridden
// by an environment variable of the form ALCES_HUNT_<key>.
type Config struct {
	Root             string
	Port             string
	TargetHost       string
	AutorunMode      string
	IncludeSelf      bool
	Broadcast        bool
	BroadcastAddress string
	ContentCommand   string
	AllowExisting    bool
	AuthKey          string
	AutoParse        string
	DefaultLabel     string
	DefaultStart     string
	PrefixStarts     map[string]string
	SkipUsedIndex    bool
	RetryInterval    string
	AutoApply        []ApplyRule
	Presets          Presets
	ProfileCommand   []string
	profileSet       bool
}

// ApplyRule is one auto_apply regex → identity pair, in file order.
type ApplyRule struct {
	Regex    string
	Identity string
}

// Presets are client-side defaults for label, prefix and groups.
type Presets struct {
	Label  string   `yaml:"label"`
	Prefix string   `yaml:"prefix"`
	Groups []string `yaml:"groups"`
}

type fileConfig struct {
	Port             interface{}       `yaml:"port"`
	TargetHost       string            `yaml:"target_host"`
	AutorunMode      string            `yaml:"autorun_mode"`
	IncludeSelf      interface{}       `yaml:"include_self"`
	Broadcast        interface{}       `yaml:"broadcast"`
	BroadcastAddress string            `yaml:"broadcast_address"`
	ContentCommand   string            `yaml:"content_command"`
	AllowExisting    interface{}       `yaml:"allow_existing"`
	AuthKey          string            `yaml:"auth_key"`
	AutoParse        string            `yaml:"auto_parse"`
	DefaultLabel     string            `yaml:"default_label"`
	DefaultStart     interface{}       `yaml:"default_start"`
	PrefixStarts     map[string]string `yaml:"prefix_starts"`
	SkipUsedIndex    interface{}       `yaml:"skip_used_index"`
	RetryInterval    interface{}       `yaml:"retry_interval"`
	AutoApply        yaml.Node         `yaml:"auto_apply"`
	Presets          Presets           `yaml:"presets"`
	ProfileCommand   interface{}       `yaml:"profile_command"`
}

// Load reads YAML config from the installation tree and XDG paths,
// then applies ALCES_HUNT_* environment overrides.
func Load() (*Config, error) {
	root, err := ResolveRoot()
	if err != nil {
		return nil, err
	}
	cfg := &Config{
		Root:         root,
		DefaultLabel: "long",
		DefaultStart: "01",
		AutoParse:    ".^",
		PrefixStarts: map[string]string{},
	}

	for _, path := range configPaths(root) {
		if err := mergeFile(cfg, path); err != nil {
			return nil, err
		}
	}
	applyEnv(cfg)
	if cfg.DefaultLabel == "" {
		cfg.DefaultLabel = "long"
	}
	if cfg.DefaultStart == "" {
		cfg.DefaultStart = "01"
	}
	if cfg.AutoParse == "" {
		cfg.AutoParse = ".^"
	}
	if cfg.PrefixStarts == nil {
		cfg.PrefixStarts = map[string]string{}
	}
	if err := validateAutoApply(cfg.AutoApply); err != nil {
		return nil, err
	}
	if !cfg.profileSet {
		cfg.ProfileCommand = []string{filepath.Join(root, "bin", "alces-hunt"), "profile"}
	}
	return cfg, nil
}

// ResolveRoot returns the installation root directory.
func ResolveRoot() (string, error) {
	if v := os.Getenv("ALCES_HUNT_ROOT"); v != "" {
		return filepath.Abs(v)
	}
	exe, err := os.Executable()
	if err == nil {
		resolved, err := filepath.EvalSymlinks(exe)
		if err == nil {
			exe = resolved
		}
		dir := filepath.Dir(exe)
		// Installed as $ROOT/bin/alces-hunt
		if filepath.Base(dir) == "bin" {
			return filepath.Abs(filepath.Dir(dir))
		}
		// Development: binary next to repo or in bin/
		if st, err := os.Stat(filepath.Join(filepath.Dir(dir), "etc")); err == nil && st.IsDir() {
			return filepath.Abs(filepath.Dir(dir))
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Abs(cwd)
}

// BufferDir is var/buffer under the root.
func (c *Config) BufferDir() string {
	return filepath.Join(c.Root, "var", "buffer")
}

// ParsedDir is var/parsed under the root.
func (c *Config) ParsedDir() string {
	return filepath.Join(c.Root, "var", "parsed")
}

// EnsureDirs creates the buffer and parsed directories.
func (c *Config) EnsureDirs() error {
	for _, d := range []string{c.BufferDir(), c.ParsedDir(), filepath.Join(c.Root, "var", "log")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// ProfileCommandPath returns the executable path of the profile command.
func (c *Config) ProfileCommandPath() string {
	if len(c.ProfileCommand) == 0 {
		return ""
	}
	return c.ProfileCommand[0]
}

func configPaths(root string) []string {
	var paths []string
	paths = append(paths, filepath.Join(root, "etc", "config.yml"))
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		if home, err := os.UserHomeDir(); err == nil {
			xdg = filepath.Join(home, ".config")
		}
	}
	if xdg != "" {
		paths = append(paths, filepath.Join(xdg, "alces-hunt", "config.yml"))
	}
	return paths
}

func mergeFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var fc fileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	if s := stringify(fc.Port); s != "" {
		cfg.Port = s
	}
	if fc.TargetHost != "" {
		cfg.TargetHost = fc.TargetHost
	}
	if fc.AutorunMode != "" {
		cfg.AutorunMode = fc.AutorunMode
	}
	if fc.IncludeSelf != nil {
		cfg.IncludeSelf = asBool(fc.IncludeSelf)
	}
	if fc.Broadcast != nil {
		cfg.Broadcast = asBool(fc.Broadcast)
	}
	if fc.BroadcastAddress != "" {
		cfg.BroadcastAddress = fc.BroadcastAddress
	}
	if fc.ContentCommand != "" {
		cfg.ContentCommand = fc.ContentCommand
	}
	if fc.AllowExisting != nil {
		cfg.AllowExisting = asBool(fc.AllowExisting)
	}
	if fc.AuthKey != "" {
		cfg.AuthKey = fc.AuthKey
	}
	if fc.AutoParse != "" {
		cfg.AutoParse = fc.AutoParse
	}
	if fc.DefaultLabel != "" {
		cfg.DefaultLabel = fc.DefaultLabel
	}
	if s := stringify(fc.DefaultStart); s != "" {
		cfg.DefaultStart = s
	}
	if fc.PrefixStarts != nil {
		cfg.PrefixStarts = fc.PrefixStarts
	}
	if fc.SkipUsedIndex != nil {
		cfg.SkipUsedIndex = asBool(fc.SkipUsedIndex)
	}
	if s := stringify(fc.RetryInterval); s != "" {
		cfg.RetryInterval = s
	}
	if rules, err := parseApplyRules(fc.AutoApply); err != nil {
		return err
	} else if rules != nil {
		cfg.AutoApply = rules
	}
	cfg.Presets = fc.Presets
	if fc.ProfileCommand != nil {
		cmd, err := parseProfileCommand(fc.ProfileCommand)
		if err != nil {
			return err
		}
		cfg.ProfileCommand = cmd
		cfg.profileSet = true
	}
	return nil
}

func applyEnv(cfg *Config) {
	if v, ok := lookupEnv("port"); ok {
		cfg.Port = v
	}
	if v, ok := lookupEnv("target_host"); ok {
		cfg.TargetHost = v
	}
	if v, ok := lookupEnv("autorun_mode"); ok {
		cfg.AutorunMode = v
	}
	if v, ok := lookupEnv("include_self"); ok {
		cfg.IncludeSelf = asBool(v)
	}
	if v, ok := lookupEnv("broadcast"); ok {
		cfg.Broadcast = asBool(v)
	}
	if v, ok := lookupEnv("broadcast_address"); ok {
		cfg.BroadcastAddress = v
	}
	if v, ok := lookupEnv("content_command"); ok {
		cfg.ContentCommand = v
	}
	if v, ok := lookupEnv("allow_existing"); ok {
		cfg.AllowExisting = asBool(v)
	}
	if v, ok := lookupEnv("auth_key"); ok {
		cfg.AuthKey = v
	}
	if v, ok := lookupEnv("auto_parse"); ok {
		cfg.AutoParse = v
	}
	if v, ok := lookupEnv("default_label"); ok {
		cfg.DefaultLabel = v
	}
	if v, ok := lookupEnv("default_start"); ok {
		cfg.DefaultStart = v
	}
	if v, ok := lookupEnv("skip_used_index"); ok {
		cfg.SkipUsedIndex = asBool(v)
	}
	if v, ok := lookupEnv("retry_interval"); ok {
		cfg.RetryInterval = v
	}
	if v, ok := lookupEnv("prefix_starts"); ok {
		m := map[string]string{}
		if err := json.Unmarshal([]byte(v), &m); err == nil {
			cfg.PrefixStarts = m
		} else if err := yaml.Unmarshal([]byte(v), &m); err == nil {
			cfg.PrefixStarts = m
		}
	}
	if v, ok := lookupEnv("auto_apply"); ok {
		if rules, err := parseApplyRulesEnv(v); err == nil {
			cfg.AutoApply = rules
		}
	}
	if v, ok := lookupEnv("profile_command"); ok {
		cfg.ProfileCommand = strings.Fields(v)
		cfg.profileSet = true
	}
}

func lookupEnv(key string) (string, bool) {
	return os.LookupEnv(envPrefix + key)
}

func stringify(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprint(t)
	}
}

func asBool(v interface{}) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "1", "true", "yes", "on":
			return true
		default:
			return false
		}
	case int:
		return t != 0
	case int64:
		return t != 0
	case float64:
		return t != 0
	default:
		return false
	}
}

func parseProfileCommand(v interface{}) ([]string, error) {
	switch t := v.(type) {
	case string:
		return strings.Fields(t), nil
	case []interface{}:
		out := make([]string, 0, len(t))
		for _, item := range t {
			out = append(out, fmt.Sprint(item))
		}
		return out, nil
	case []string:
		return t, nil
	default:
		return nil, fmt.Errorf("profile_command must be a string or array")
	}
}

func parseApplyRules(n yaml.Node) ([]ApplyRule, error) {
	if n.Kind == 0 {
		return nil, nil
	}
	if n.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("Malformed hash passed to `auto_apply`")
	}
	var rules []ApplyRule
	for i := 0; i+1 < len(n.Content); i += 2 {
		rules = append(rules, ApplyRule{Regex: n.Content[i].Value, Identity: n.Content[i+1].Value})
	}
	return rules, nil
}

func parseApplyRulesEnv(v string) ([]ApplyRule, error) {
	var raw yaml.Node
	if err := yaml.Unmarshal([]byte(v), &raw); err != nil {
		// JSON object
		var m map[string]string
		if jerr := json.Unmarshal([]byte(v), &m); jerr != nil {
			return nil, fmt.Errorf("Malformed hash passed to `auto_apply`")
		}
		var rules []ApplyRule
		for k, id := range m {
			rules = append(rules, ApplyRule{Regex: k, Identity: id})
		}
		return rules, nil
	}
	doc := raw
	if raw.Kind == yaml.DocumentNode && len(raw.Content) > 0 {
		doc = *raw.Content[0]
	}
	if doc.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("Malformed hash passed to `auto_apply`")
	}
	return parseApplyRules(doc)
}

func validateAutoApply(rules []ApplyRule) error {
	var bad []string
	for _, r := range rules {
		if _, err := regexp.Compile(NormalizeRegex(r.Regex)); err != nil {
			bad = append(bad, r.Regex)
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("The following regular expressions passed to `auto_apply` are invalid:\n%s", strings.Join(bad, "\n"))
	}
	return nil
}

// NormalizeRegex strips optional surrounding slashes used in Ruby-style literals.
func NormalizeRegex(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '/' && s[len(s)-1] == '/' {
		return s[1 : len(s)-1]
	}
	return s
}

// CompileRegex compiles a user-supplied pattern after slash-normalization.
func CompileRegex(s string) (*regexp.Regexp, error) {
	return regexp.Compile(NormalizeRegex(s))
}

// HasAutoApply reports whether any auto-apply rules are configured.
func (c *Config) HasAutoApply() bool {
	return len(c.AutoApply) > 0
}
