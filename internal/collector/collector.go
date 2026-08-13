// Copyright (C) 2026 Alces Software Ltd.
//
// This program and the accompanying materials are made available under
// the terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0
//
// SPDX-License-Identifier: EPL-2.0

package collector

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Collect gathers default hardware identity content as YAML.
// sysroot is typically "" meaning the live system; tests may pass a fake root.
func Collect(sysroot string) (string, error) {
	payload := map[string]interface{}{}

	cmdlinePath := joinRoot(sysroot, "/proc/cmdline")
	if data, err := os.ReadFile(cmdlinePath); err == nil {
		kv := parseCmdline(string(data))
		if v := kv["SYSUUID"]; v != "" {
			payload["sysuuid"] = v
		}
		if v := kv["BOOTIF"]; v != "" {
			payload["bootif"] = v
		}
	}

	nets := collectNets(sysroot)
	if len(nets) > 0 {
		payload["nets"] = nets
	}

	disks := collectDisks(sysroot)
	if len(disks) > 0 {
		payload["disks"] = disks
	}

	if bmcip, bmcmac := collectBMC(); bmcip != "" || bmcmac != "" {
		if bmcip != "" {
			payload["bmcip"] = bmcip
		}
		if bmcmac != "" {
			payload["bmcmac"] = bmcmac
		}
	}

	return marshalYAML(payload)
}

// HostID returns SYSUUID from /proc/cmdline, falling back to `hostid`.
func HostID(sysroot string) string {
	syshostid := runTrim("hostid")
	cmdlinePath := joinRoot(sysroot, "/proc/cmdline")
	data, err := os.ReadFile(cmdlinePath)
	if err != nil {
		return syshostid
	}
	kv := parseCmdline(string(data))
	if v := kv["SYSUUID"]; v != "" {
		return v
	}
	if syshostid != "" {
		return syshostid
	}
	h, _ := os.Hostname()
	return h
}

// DefaultSerialLabel returns the trimmed stdout of
// `dmidecode -s system-serial-number`. It fails if dmidecode cannot be executed.
func DefaultSerialLabel() (string, error) {
	path, err := exec.LookPath("dmidecode")
	if err != nil {
		return "", fmt.Errorf("dmidecode is required for the default send label and is not available")
	}
	cmd := exec.Command(path, "-s", "system-serial-number")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("dmidecode is required for the default send label and is not available")
	}
	return strings.TrimSpace(stdout.String()), nil
}

// RunCommand executes a shell command and returns chomped stdout.
func RunCommand(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}

func collectNets(sysroot string) map[string]string {
	dir := joinRoot(sysroot, "/sys/class/net")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	nets := map[string]string{}
	var names []string
	for _, e := range entries {
		name := e.Name()
		if name == "lo" || strings.HasPrefix(name, ".") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		addr, err := os.ReadFile(filepath.Join(dir, name, "address"))
		if err != nil {
			nets[name] = "unknown"
			continue
		}
		nets[name] = strings.TrimSpace(string(addr))
	}
	return nets
}

func collectDisks(sysroot string) map[string]string {
	dir := joinRoot(sysroot, "/sys/class/block")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	disks := map[string]string{}
	var names []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if st, err := os.Stat(filepath.Join(dir, name, "device")); err != nil || !st.IsDir() {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		size, err := os.ReadFile(filepath.Join(dir, name, "size"))
		if err != nil {
			continue
		}
		disks[name] = strings.TrimSpace(string(size))
	}
	return disks
}

func collectBMC() (ip, mac string) {
	ip = ipmitoolField("IP Address", true)
	mac = ipmitoolField("MAC Address", false)
	return ip, mac
}

func ipmitoolField(label string, skipSource bool) string {
	script := fmt.Sprintf(`ipmitool lan print 1 2>/dev/null | grep -e %q`, label)
	if skipSource {
		script += ` | grep -vi Source`
	}
	script += ` | awk '{ print $4 }'`
	out, err := exec.Command("sh", "-c", script).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func parseCmdline(s string) map[string]string {
	out := map[string]string{}
	for _, tok := range strings.Fields(s) {
		k, v, ok := strings.Cut(tok, "=")
		if !ok || k == "" {
			continue
		}
		out[k] = v
	}
	return out
}

func joinRoot(sysroot, path string) string {
	if sysroot == "" {
		return path
	}
	return filepath.Join(sysroot, path)
}

func runTrim(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func marshalYAML(v interface{}) (string, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		enc.Close()
		return "", err
	}
	enc.Close()
	return buf.String(), nil
}
