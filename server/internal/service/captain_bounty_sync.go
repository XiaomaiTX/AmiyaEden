package service

import (
	"amiya-eden/internal/model"
	"sort"
	"time"
)

var supportedPlayerAttributionRefTypes = map[string]struct{}{
	"bounty_prizes": {},
}

func supportedPlayerAttributionRefTypeList() []string {
	result := make([]string, 0, len(supportedPlayerAttributionRefTypes))
	for refType := range supportedPlayerAttributionRefTypes {
		result = append(result, refType)
	}
	sort.Strings(result)
	return result
}

func refTypeSupportsPlayerAttribution(refType string) bool {
	_, ok := supportedPlayerAttributionRefTypes[refType]
	return ok
}

func selectCaptainWalletJournalMatch(
	playerJournal model.EVECharacterWalletJournal,
	candidates []model.EVECharacterWalletJournal,
) *model.EVECharacterWalletJournal {
	filtered := make([]model.EVECharacterWalletJournal, 0, len(candidates))
	for _, candidate := range candidates {
		if !captainJournalCandidateMatches(playerJournal, candidate) {
			continue
		}
		filtered = append(filtered, candidate)
	}
	if len(filtered) == 0 {
		return nil
	}

	best := filtered[0]
	bestDelta := absDuration(best.Date.Sub(playerJournal.Date))

	for _, candidate := range filtered[1:] {
		candidateDelta := absDuration(candidate.Date.Sub(playerJournal.Date))
		if candidateDelta < bestDelta {
			best = candidate
			bestDelta = candidateDelta
			continue
		}
		if candidateDelta > bestDelta {
			continue
		}
		if candidate.Date.Before(best.Date) {
			best = candidate
			bestDelta = candidateDelta
			continue
		}
		if candidate.Date.Equal(best.Date) && candidate.ID < best.ID {
			best = candidate
			bestDelta = candidateDelta
		}
	}

	return &best
}

func shouldConsiderAttributionJournal(
	journal model.EVECharacterWalletJournal,
	now time.Time,
	lookbackDays int,
) bool {
	if !refTypeSupportsPlayerAttribution(journal.RefType) {
		return false
	}

	lookbackStart := now.AddDate(0, 0, -lookbackDays)
	return !journal.Date.Before(lookbackStart)
}

func captainJournalCandidateMatches(
	playerJournal model.EVECharacterWalletJournal,
	candidate model.EVECharacterWalletJournal,
) bool {
	if !refTypeSupportsPlayerAttribution(playerJournal.RefType) {
		return false
	}
	if candidate.RefType != playerJournal.RefType {
		return false
	}
	return bountyReasonsAgree(playerJournal.Reason, candidate.Reason)
}

func bountyReasonsAgree(playerReason, candidateReason string) bool {
	playerNPCs := parseReason(playerReason)
	candidateNPCs := parseReason(candidateReason)
	if len(playerNPCs) == 0 || len(candidateNPCs) == 0 {
		return false
	}
	if len(playerNPCs) != len(candidateNPCs) {
		return false
	}
	for npcID, count := range playerNPCs {
		if candidateNPCs[npcID] != count {
			return false
		}
	}
	return true
}

func absDuration(v time.Duration) time.Duration {
	if v < 0 {
		return -v
	}
	return v
}
