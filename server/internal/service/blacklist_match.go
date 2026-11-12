package service

import (
	"amiya-eden/internal/model"
	"maps"
	"slices"
	"strconv"
)

// BlacklistHit is one blacklist entry matching a screened person.
type BlacklistHit struct {
	Kind    string `json:"kind"`
	Value   string `json:"value"`
	Display string `json:"display"`
	Reason  string `json:"reason"`
}

// blacklistSubject is the set of check values of one screened person, keyed by
// blacklist kind. Values are normalized on the way in, exactly like the stored
// entries, so matching is plain equality.
type blacklistSubject map[string]map[string]struct{}

func (s blacklistSubject) add(kind, value string) {
	if value == "" {
		return
	}
	if s[kind] == nil {
		s[kind] = map[string]struct{}{}
	}
	s[kind][value] = struct{}{}
}

func (s blacklistSubject) addCharacter(id int64) {
	if id > 0 {
		s.add(model.BlacklistKindCharacter, strconv.FormatInt(id, 10))
	}
}

func (s blacklistSubject) addQQ(raw string) { s.add(model.BlacklistKindQQ, normalizeBlacklistQQ(raw)) }

func (s blacklistSubject) addDiscord(raw string) {
	s.add(model.BlacklistKindDiscord, normalizeBlacklistDiscord(raw))
}

// blacklistQueryValues merges several subjects into one sorted value list per
// kind, so all of them are looked up with a single query.
func blacklistQueryValues(subjects []blacklistSubject) (characters, qqs, discords []string) {
	merged := blacklistSubject{}
	for _, subject := range subjects {
		for kind, values := range subject {
			for value := range values {
				merged.add(kind, value)
			}
		}
	}
	list := func(kind string) []string { return slices.Sorted(maps.Keys(merged[kind])) }
	return list(model.BlacklistKindCharacter), list(model.BlacklistKindQQ), list(model.BlacklistKindDiscord)
}

// matchBlacklist returns the entries that belong to the subject, in entry order.
func matchBlacklist(entries []model.BlacklistEntry, subject blacklistSubject) []BlacklistHit {
	hits := make([]BlacklistHit, 0)
	for _, entry := range entries {
		if _, ok := subject[entry.Kind][entry.Value]; ok {
			hits = append(hits, BlacklistHit{Kind: entry.Kind, Value: entry.Value, Display: entry.Display, Reason: entry.Reason})
		}
	}
	return hits
}
