package utils

// LoadSymbolByScan is the scan tier (Tier 2, D-01): a scoped lexical
// candidate search over d.ImportPaths, each candidate confirmed by a real
// ParseOne plus a MessageRegistry re-check before it is trusted (D-05,
// T-13-02) -- a lexical match alone is never an answer. Returns false
// immediately when the registry has no ImportPaths (every eager registry,
// D-03).
//
// TODO(13-01 RED): stub -- always returns false. GREEN phase wires the real
// scoped scan.
func (d *DescriptorRegistry) LoadSymbolByScan(fullName string) bool {
	return false
}

// ScanResolutionCount reports how many times the scan tier (LoadSymbolByScan)
// has answered a lookup by confirming a candidate. Test-only observable, no
// log line, no CLI surface. Safe to call concurrently with ParseOne.
func (d *DescriptorRegistry) ScanResolutionCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.scanResolutions
}
