package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// The alliance blacklist lives in one public Google Sheet. Its column layout is
// hard-wired into the parser, so the source is a code constant, not a setting.
const (
	blacklistSheetID       = "1nKtgFowqPEI7I1qzuXnKIFtNZurpKw7K5mZmCY4AHFg"
	blacklistSheetGID      = "0" // "The List" tab; Grey List, Revoked List and the CN sheet are out of scope.
	blacklistSheetURL      = "https://docs.google.com/spreadsheets/d/" + blacklistSheetID + "/export?format=csv&gid=" + blacklistSheetGID
	blacklistSheetTimeout  = 30 * time.Second
	blacklistSheetMaxBytes = 10 << 20
	blacklistHeaderScan    = 5
)

// Zero-based columns of "The List".
const (
	blacklistColMain    = 0
	blacklistColReason  = 3
	blacklistColAlt1    = 5
	blacklistColAlt2    = 6
	blacklistColAlt3    = 7
	blacklistColDiscord = 8
	blacklistColQQ      = 9
)

var (
	blacklistNameColumns = []int{blacklistColMain, blacklistColAlt1, blacklistColAlt2, blacklistColAlt3}
	blacklistQQPattern   = regexp.MustCompile(`^[1-9][0-9]{4,11}$`)
	// Deliberately looser than EVE's real rules: a doubtful name goes to ESI
	// rather than being dropped, so legitimate old character names survive.
	blacklistNamePattern = regexp.MustCompile(`^[A-Za-z0-9 .'_-]{3,37}$`)
)

type blacklistDiscordName struct {
	Value   string // lowercased match key
	Display string // original spelling
}

// blacklistSheetRow is one data row. Names, QQ numbers and Discord names are
// independent check values; the row only contributes its reason to each.
type blacklistSheetRow struct {
	Names    []string
	QQs      []string
	Discords []blacklistDiscordName
	Reason   string
}

func downloadBlacklistSheet(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build request: %v", ErrBlacklistSourceUnavailable, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBlacklistSourceUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrBlacklistSourceUnavailable, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, blacklistSheetMaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrBlacklistSourceUnavailable, err)
	}
	if len(data) > blacklistSheetMaxBytes {
		return nil, fmt.Errorf("%w: body exceeds %d bytes", ErrBlacklistSourceUnavailable, blacklistSheetMaxBytes)
	}
	return data, nil
}

// parseBlacklistSheet turns the CSV export into data rows. Rows without any
// name, QQ number or Discord name are dropped; every other row counts, whether
// or not its names later resolve to characters.
func parseBlacklistSheet(data []byte) ([]blacklistSheetRow, error) {
	reader := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})))
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%w: parse csv: %v", ErrBlacklistSourceUnavailable, err)
	}
	header := findBlacklistHeader(records)
	if header < 0 {
		return nil, ErrBlacklistSourceLayoutChanged
	}
	rows := make([]blacklistSheetRow, 0, len(records)-header-1)
	for _, record := range records[header+1:] {
		row := blacklistSheetRow{
			Reason:   strings.TrimSpace(blacklistCell(record, blacklistColReason)),
			QQs:      splitBlacklistQQs(blacklistCell(record, blacklistColQQ)),
			Discords: splitBlacklistDiscords(blacklistCell(record, blacklistColDiscord)),
		}
		for _, col := range blacklistNameColumns {
			row.Names = append(row.Names, splitBlacklistNames(blacklistCell(record, col))...)
		}
		if len(row.Names)+len(row.QQs)+len(row.Discords) > 0 {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func findBlacklistHeader(records [][]string) int {
	for i := 0; i < min(blacklistHeaderScan, len(records)); i++ {
		record := records[i]
		has := func(col int, word string) bool {
			return strings.Contains(strings.ToLower(blacklistCell(record, col)), word)
		}
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(blacklistCell(record, blacklistColMain))), "main") &&
			has(blacklistColReason, "reason") &&
			has(blacklistColAlt1, "alt") && has(blacklistColAlt2, "alt") && has(blacklistColAlt3, "alt") &&
			has(blacklistColDiscord, "discord") && has(blacklistColQQ, "qq") {
			return i
		}
	}
	return -1
}

func blacklistCell(record []string, col int) string {
	if col < len(record) {
		return record[col]
	}
	return ""
}

func splitBlacklistNames(cell string) []string {
	var names []string
	for _, part := range strings.FieldsFunc(cell, func(r rune) bool { return r == '\n' || r == ',' }) {
		if name := strings.TrimSpace(part); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// validBlacklistCharacterName reports whether a trimmed name could be an EVE
// character name; names that fail are recorded as invalid without asking ESI.
func validBlacklistCharacterName(name string) bool {
	return blacklistNamePattern.MatchString(name)
}

func splitBlacklistQQs(cell string) []string {
	var qqs []string
	for _, token := range strings.FieldsFunc(cell, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune(",;/，；、", r)
	}) {
		if qq := normalizeBlacklistQQ(token); qq != "" {
			qqs = append(qqs, qq)
		}
	}
	return qqs
}

// Discord cells are split on line breaks only: legacy usernames may contain
// spaces, and splitting on them would invent short names that match strangers.
func splitBlacklistDiscords(cell string) []blacklistDiscordName {
	var names []blacklistDiscordName
	for _, line := range strings.Split(cell, "\n") {
		if display := blacklistDiscordDisplay(line); display != "" {
			names = append(names, blacklistDiscordName{Value: strings.ToLower(display), Display: display})
		}
	}
	return names
}

// The normalizers below define the stored blacklist values; anything compared
// against the blacklist must go through the same functions.

// normalizeBlacklistQQ returns the trimmed QQ number, or "" when it is not a
// plausible QQ number.
func normalizeBlacklistQQ(s string) string {
	if qq := strings.TrimSpace(s); blacklistQQPattern.MatchString(qq) {
		return qq
	}
	return ""
}

// normalizeBlacklistDiscord returns the match key of a Discord name, or "".
func normalizeBlacklistDiscord(s string) string {
	return strings.ToLower(blacklistDiscordDisplay(s))
}

// blacklistDiscordDisplay trims a Discord name and drops one leading at-sign.
func blacklistDiscordDisplay(s string) string {
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "@"))
}
