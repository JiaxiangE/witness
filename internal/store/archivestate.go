package store

// ArchiveState is a cheap read-side probe. RawCount("") counts the literal empty
// session, NOT the archive, so it cannot diagnose whether capture has started.
type ArchiveState struct {
	HasRaw, HasObservations, HasDerived bool
}

// ArchiveState treats an empty lens as the cross-lens view. Derived rows survive
// cleanup, so their presence must not be mistaken for an unconfigured capture path.
func (r *rawIO) ArchiveState(lens string) (s ArchiveState, err error) {
	err = r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM raw),
		EXISTS(SELECT 1 FROM observations WHERE ? = '' OR lens = ?),
		EXISTS(SELECT 1 FROM observations) OR EXISTS(SELECT 1 FROM facets)`, lens, lens).
		Scan(&s.HasRaw, &s.HasObservations, &s.HasDerived)
	return
}
