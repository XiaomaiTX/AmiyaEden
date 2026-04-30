package repository

import "testing"

func TestNormalizeAuditListSize(t *testing.T) {
	if got := normalizeAuditListSize(0); got != defaultAuditListSize {
		t.Fatalf("normalizeAuditListSize(0) = %d, want %d", got, defaultAuditListSize)
	}
	if got := normalizeAuditListSize(maxAuditListSize + 1); got != maxAuditListSize {
		t.Fatalf("normalizeAuditListSize(overflow) = %d, want %d", got, maxAuditListSize)
	}
}

func TestNormalizeAuditExportLimit(t *testing.T) {
	if got := normalizeAuditExportLimit(0); got != defaultAuditExportLimit {
		t.Fatalf("normalizeAuditExportLimit(0) = %d, want %d", got, defaultAuditExportLimit)
	}
	if got := normalizeAuditExportLimit(maxAuditExportLimit + 1); got != maxAuditExportLimit {
		t.Fatalf("normalizeAuditExportLimit(overflow) = %d, want %d", got, maxAuditExportLimit)
	}
}

func TestNormalizeAuditArchiveBatchSize(t *testing.T) {
	if got := normalizeAuditArchiveBatchSize(0); got != defaultAuditArchiveBatch {
		t.Fatalf("normalizeAuditArchiveBatchSize(0) = %d, want %d", got, defaultAuditArchiveBatch)
	}
	if got := normalizeAuditArchiveBatchSize(maxAuditArchiveBatch + 1); got != maxAuditArchiveBatch {
		t.Fatalf("normalizeAuditArchiveBatchSize(overflow) = %d, want %d", got, maxAuditArchiveBatch)
	}
}

func TestNormalizeAuditExportTaskLimit(t *testing.T) {
	if got := normalizeAuditExportTaskLimit(0); got != defaultAuditExportTaskLimit {
		t.Fatalf("normalizeAuditExportTaskLimit(0) = %d, want %d", got, defaultAuditExportTaskLimit)
	}
	if got := normalizeAuditExportTaskLimit(maxAuditExportTaskLimit + 1); got != maxAuditExportTaskLimit {
		t.Fatalf("normalizeAuditExportTaskLimit(overflow) = %d, want %d", got, maxAuditExportTaskLimit)
	}
}
