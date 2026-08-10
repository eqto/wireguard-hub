package wireguard

import "strings"

// wgConfigLine is a single line in a WireGuard config section. Either it is a
// key=value pair (isKV=true, key is lowercased for matching, displayKey is the
// original casing for serialization) or a raw line (comment, blank, or
// unparseable) preserved as-is.
type wgConfigLine struct {
	raw        string
	isKV       bool
	key        string // lowercased key for matching
	displayKey string // original casing for serialization
	value      string
}

// wgConfigSection is a single [Interface] or [Peer] section of a WireGuard
// config file. Lines are preserved in order so serialized output closely
// matches the input format.
type wgConfigSection struct {
	header string
	lines  []wgConfigLine
}

// wgConfigFile is a parsed WireGuard config file: one [Interface] section and
// zero or more ordered [Peer] sections.
type wgConfigFile struct {
	interfaceSection *wgConfigSection
	peers            []*wgConfigSection
}

// metadataCommentKeys are comment prefixes treated as key=value pairs (# Name,
// # Description) rather than plain comments to skip during parsing.
var metadataCommentKeys = map[string]bool{
	"# name":        true,
	"# description": true,
}

// parseWGConfig parses a WireGuard config file into structured sections.
func parseWGConfig(text string) *wgConfigFile {
	cfg := &wgConfigFile{}
	var current *wgConfigSection
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case "[Interface]":
			if current != nil {
				cfg.addSection(current)
			}
			current = &wgConfigSection{header: "[Interface]"}
			continue
		case "[Peer]":
			if current != nil {
				cfg.addSection(current)
			}
			current = &wgConfigSection{header: "[Peer]"}
			continue
		}
		if current == nil {
			continue
		}
		current.lines = append(current.lines, parseWGLine(trimmed))
	}
	if current != nil {
		cfg.addSection(current)
	}
	return cfg
}

func (c *wgConfigFile) addSection(s *wgConfigSection) {
	if s.header == "[Interface]" {
		c.interfaceSection = s
	} else {
		c.peers = append(c.peers, s)
	}
}

func parseWGLine(trimmed string) wgConfigLine {
	if trimmed == "" {
		return wgConfigLine{raw: ""}
	}
	if strings.HasPrefix(trimmed, "#") {
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "# name") || strings.HasPrefix(lower, "# description") {
			if parts := strings.SplitN(trimmed, "=", 2); len(parts) == 2 {
				return wgConfigLine{
					raw:        trimmed,
					isKV:       true,
					key:        strings.ToLower(strings.TrimSpace(parts[0])),
					displayKey: strings.TrimSpace(parts[0]),
					value:      strings.TrimSpace(parts[1]),
				}
			}
		}
		return wgConfigLine{raw: trimmed}
	}
	parts := strings.SplitN(trimmed, "=", 2)
	if len(parts) != 2 {
		return wgConfigLine{raw: trimmed}
	}
	return wgConfigLine{
		raw:        trimmed,
		isKV:       true,
		key:        strings.ToLower(strings.TrimSpace(parts[0])),
		displayKey: strings.TrimSpace(parts[0]),
		value:      strings.TrimSpace(parts[1]),
	}
}

// get returns the value for the given (lowercased) key, or "" if not found.
func (s *wgConfigSection) get(key string) string {
	for _, l := range s.lines {
		if l.isKV && l.key == key {
			return l.value
		}
	}
	return ""
}

// set updates the value of an existing key or appends a new key=value line.
func (s *wgConfigSection) set(displayKey, value string) {
	key := strings.ToLower(displayKey)
	for i, l := range s.lines {
		if l.isKV && l.key == key {
			s.lines[i].value = value
			s.lines[i].displayKey = displayKey
			s.lines[i].raw = formatWGLine(displayKey, value)
			return
		}
	}
	s.lines = append(s.lines, wgConfigLine{
		raw: formatWGLine(displayKey, value), isKV: true,
		key: key, displayKey: displayKey, value: value,
	})
}

// removeKey removes all lines matching the given (lowercased) key.
func (s *wgConfigSection) removeKey(key string) {
	for i := len(s.lines) - 1; i >= 0; i-- {
		if s.lines[i].isKV && s.lines[i].key == key {
			s.lines = append(s.lines[:i], s.lines[i+1:]...)
		}
	}
}

// setMetadata inserts or updates # Name and # Description comments immediately
// after the PublicKey line in this peer section.
func (s *wgConfigSection) setMetadata(name, description string) {
	s.removeKey("# name")
	s.removeKey("# description")
	var newLines []wgConfigLine
	if name != "" {
		newLines = append(newLines, wgConfigLine{
			raw: formatWGLine("# Name", name), isKV: true,
			key: "# name", displayKey: "# Name", value: name,
		})
	}
	if description != "" {
		newLines = append(newLines, wgConfigLine{
			raw: formatWGLine("# Description", description), isKV: true,
			key: "# description", displayKey: "# Description", value: description,
		})
	}
	if len(newLines) == 0 {
		return
	}
	idx := s.indexOfKey("publickey")
	s.insertAfter(idx, newLines)
}

// indexOfKey returns the index of the first line matching key, or -1.
func (s *wgConfigSection) indexOfKey(key string) int {
	for i, l := range s.lines {
		if l.isKV && l.key == key {
			return i
		}
	}
	return -1
}

// insertAfter inserts newLines immediately after position idx (or at the end
// if idx is -1).
func (s *wgConfigSection) insertAfter(idx int, newLines []wgConfigLine) {
	if idx < 0 || idx >= len(s.lines) {
		s.lines = append(s.lines, newLines...)
		return
	}
	s.lines = append(s.lines, newLines...) // grow
	copy(s.lines[idx+1+len(newLines):], s.lines[idx+1:])
	copy(s.lines[idx+1:], newLines)
}

// findPeer returns the [Peer] section matching publicKey, or nil.
func (c *wgConfigFile) findPeer(publicKey string) *wgConfigSection {
	for _, p := range c.peers {
		if p.get("publickey") == publicKey {
			return p
		}
	}
	return nil
}

// removePeer removes the [Peer] section matching publicKey. Returns true if a
// section was removed.
func (c *wgConfigFile) removePeer(publicKey string) bool {
	for i, p := range c.peers {
		if p.get("publickey") == publicKey {
			c.peers = append(c.peers[:i], c.peers[i+1:]...)
			return true
		}
	}
	return false
}

// serialize renders the config file back to text.
func (c *wgConfigFile) serialize() string {
	var b strings.Builder
	if c.interfaceSection != nil {
		b.WriteString(c.interfaceSection.serialize())
	}
	for _, p := range c.peers {
		b.WriteString(p.serialize())
	}
	return b.String()
}

func (s *wgConfigSection) serialize() string {
	var b strings.Builder
	b.WriteString(s.header)
	b.WriteByte('\n')
	for _, l := range s.lines {
		b.WriteString(l.raw)
		b.WriteByte('\n')
	}
	return b.String()
}

func formatWGLine(key, value string) string {
	return key + " = " + value
}
