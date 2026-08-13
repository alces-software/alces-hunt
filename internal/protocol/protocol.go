// Copyright (C) 2026 Alces Software Ltd.
//
// This program and the accompanying materials are made available under
// the terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0
//
// SPDX-License-Identifier: EPL-2.0

package protocol

// Payload is the JSON body sent by the client.
type Payload struct {
	HostID   string   `json:"hostid"`
	Hostname string   `json:"hostname"`
	Content  string   `json:"content"`
	Label    string   `json:"label,omitempty"`
	Prefix   string   `json:"prefix,omitempty"`
	Groups   []string `json:"groups,omitempty"`
	AuthKey  string   `json:"auth_key"`
}
