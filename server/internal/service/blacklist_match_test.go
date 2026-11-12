package service

import (
	"amiya-eden/internal/model"
	"fmt"
	"testing"
)

func TestBlacklistSubjectNormalizesAndMerges(t *testing.T) {
	first := blacklistSubject{}
	first.addCharacter(90000002)
	first.addCharacter(0)
	first.addQQ(" 12345678 ")
	first.addQQ("1234")
	first.addDiscord("@Tester")
	first.addDiscord("  ")
	second := blacklistSubject{}
	second.addCharacter(90000001)
	second.addCharacter(90000002)
	second.addDiscord("tester")

	characters, qqs, discords := blacklistQueryValues([]blacklistSubject{first, second})
	got := fmt.Sprint(characters, qqs, discords)
	if want := "[90000001 90000002] [12345678] [tester]"; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	if characters, qqs, discords := blacklistQueryValues(nil); characters != nil || qqs != nil || discords != nil {
		t.Fatalf("empty subjects = %v %v %v", characters, qqs, discords)
	}
}

func TestMatchBlacklistAttributesEntriesToTheirSubject(t *testing.T) {
	entries := []model.BlacklistEntry{
		{Kind: model.BlacklistKindCharacter, Value: "90000001", Display: "Test Pilot", Reason: "scam"},
		{Kind: model.BlacklistKindDiscord, Value: "tester", Display: "Tester", Reason: "spy"},
		{Kind: model.BlacklistKindQQ, Value: "12345678", Display: "12345678", Reason: "awox"},
	}
	first := blacklistSubject{}
	first.addCharacter(90000001)
	first.addDiscord("@TESTER")
	second := blacklistSubject{}
	second.addQQ("12345678")
	// Same digits as the QQ entry, but as a character ID: kinds never cross.
	second.addCharacter(12345678)

	hits := matchBlacklist(entries, first)
	if got := fmt.Sprint(hits); got != "[{character 90000001 Test Pilot scam} {discord tester Tester spy}]" {
		t.Fatalf("first subject hits = %s", got)
	}
	if got := fmt.Sprint(matchBlacklist(entries, second)); got != "[{qq 12345678 12345678 awox}]" {
		t.Fatalf("second subject hits = %s", got)
	}
	if hits := matchBlacklist(entries, blacklistSubject{}); hits == nil || len(hits) != 0 {
		t.Fatalf("empty subject hits = %v", hits)
	}
}
