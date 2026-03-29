package service

const (
	ledgerDefaultPageSize = 200
	ledgerMaxPageSize     = 1000
)

func normalizeLedgerPageSize(pageSize int) int {
	if pageSize < 1 {
		return ledgerDefaultPageSize
	}
	if pageSize > ledgerMaxPageSize {
		return ledgerMaxPageSize
	}
	return pageSize
}
