package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

const blacklistTestHeader = "Main/主角色,Date Added,Added By,Reason/理由,Verification,known alt1,known alt2,known alt3,Discord Tag,QQ (for CN),EvE Who Link\n"

func TestParseBlacklistSheetFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/blacklist_sheet.csv")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseBlacklistSheet(data)
	if err != nil {
		t.Fatal(err)
	}
	want := []blacklistSheetRow{
		{
			Names:  []string{"Test Main One", "Alt Alpha", "Alt Beta"},
			QQs:    []string{"10000001", "10000002"},
			Reason: "Scam in hauling",
		},
		{
			Names: []string{"Test Main Two", "Alt Gamma", "Alt Delta"},
			Discords: []blacklistDiscordName{
				{Value: "mixed_case", Display: "Mixed_Case"},
				{Value: "legacy name#1234", Display: "Legacy Name#1234"},
			},
			Reason: "Spy",
		},
		{
			Names:  []string{"Corp Placeholder Industries"},
			QQs:    []string{"20000001", "20000002", "20000003", "20000004"},
			Reason: "Corp awox",
		},
		{
			Names:  []string{"Bad/Name Here"},
			QQs:    []string{"30000001"},
			Reason: "Theft",
		},
		{
			QQs:      []string{"40000001", "40000002"},
			Discords: []blacklistDiscordName{{Value: "contact person", Display: "Contact Person"}},
			Reason:   "Contact only",
		},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows mismatch\n got: %+v\nwant: %+v", rows, want)
	}
}

func TestParseBlacklistSheetLayoutChanged(t *testing.T) {
	cases := map[string]string{
		"no header": "Test Main,2024-01-01,tester,Spy,,,,,,10000001,\n",
		"QQ and Discord swapped": "Main/主角色,Date,By,Reason,Verification,known alt1,known alt2,known alt3,QQ (for CN),Discord Tag,Link\n" +
			"Test Main,2024-01-01,tester,Spy,,,,,,10000001,\n",
		"header after the scanned rows": strings.Repeat("note,,,,,,,,,,\n", blacklistHeaderScan) + blacklistTestHeader,
		"HTML page":                     "<!DOCTYPE html><html><head><title>Sign in</title></head><body>Sign in to continue</body></html>\n",
		"empty body":                    "",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := parseBlacklistSheet([]byte(body))
			if !errors.Is(err, ErrBlacklistSourceLayoutChanged) {
				t.Fatalf("err = %v, want layout changed", err)
			}
		})
	}
}

func TestParseBlacklistSheetAcceptsBOMAndShortRows(t *testing.T) {
	body := string([]byte{0xEF, 0xBB, 0xBF}) + blacklistTestHeader + "Short Row Pilot,2024-01-01,tester,Spy\n"
	rows, err := parseBlacklistSheet([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || !reflect.DeepEqual(rows[0].Names, []string{"Short Row Pilot"}) || rows[0].Reason != "Spy" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestSplitBlacklistQQs(t *testing.T) {
	got := splitBlacklistQQs("824084149\r\n1984278640 / 12345,99999,999999999999;1234567890123 sirbalu 123 01234567 1234")
	want := []string{"824084149", "1984278640", "12345", "99999", "999999999999"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSplitBlacklistDiscords(t *testing.T) {
	got := splitBlacklistDiscords("@Akina Suzuki#7576\n\n  plain.name  \r\n@")
	want := []blacklistDiscordName{
		{Value: "akina suzuki#7576", Display: "Akina Suzuki#7576"},
		{Value: "plain.name", Display: "plain.name"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestNormalizeBlacklistValues(t *testing.T) {
	for in, want := range map[string]string{" 12345678 ": "12345678", "12a": "", "": "", "01234567": "", "1234": ""} {
		if got := normalizeBlacklistQQ(in); got != want {
			t.Errorf("normalizeBlacklistQQ(%q) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[string]string{" @Tester ": "tester", "Tester": "tester", "@ Old Name#0001": "old name#0001", "": "", "@": ""} {
		if got := normalizeBlacklistDiscord(in); got != want {
			t.Errorf("normalizeBlacklistDiscord(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidBlacklistCharacterName(t *testing.T) {
	for _, name := range []string{"Abc", "Test Main One", "O'Neil", "a.b_c-d", strings.Repeat("x", 37)} {
		if !validBlacklistCharacterName(name) {
			t.Errorf("%q should be valid", name)
		}
	}
	for _, name := range []string{"ab", strings.Repeat("x", 38), "Bad/Name Here", "#ERROR!", "名字测试", "(retired) Pilot"} {
		if validBlacklistCharacterName(name) {
			t.Errorf("%q should be invalid", name)
		}
	}
}

func TestDownloadBlacklistSheet(t *testing.T) {
	csvBody := blacklistTestHeader + "Test Main,2024-01-01,tester,Spy,,,,,,10000001,\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/ok", http.StatusFound)
		case "/ok":
			_, _ = fmt.Fprint(w, csvBody)
		case "/html":
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(w, "<!DOCTYPE html><html><body>Sign in</body></html>")
		case "/huge":
			_, _ = w.Write(make([]byte, blacklistSheetMaxBytes+1))
		case "/exact":
			_, _ = w.Write(make([]byte, blacklistSheetMaxBytes))
		default:
			http.Error(w, "nope", http.StatusServiceUnavailable)
		}
	}))
	t.Cleanup(server.Close)
	client := &http.Client{Timeout: blacklistSheetTimeout}
	ctx := context.Background()

	data, err := downloadBlacklistSheet(ctx, client, server.URL+"/redirect")
	if err != nil || string(data) != csvBody {
		t.Fatalf("redirect download = %q, %v", data, err)
	}
	if _, err := downloadBlacklistSheet(ctx, client, server.URL+"/down"); !errors.Is(err, ErrBlacklistSourceUnavailable) {
		t.Fatalf("non-200 err = %v", err)
	}
	if _, err := downloadBlacklistSheet(ctx, client, server.URL+"/huge"); !errors.Is(err, ErrBlacklistSourceUnavailable) {
		t.Fatalf("oversized err = %v", err)
	}
	if data, err := downloadBlacklistSheet(ctx, client, server.URL+"/exact"); err != nil || len(data) != blacklistSheetMaxBytes {
		t.Fatalf("size at the limit = %d bytes, %v", len(data), err)
	}
	data, err = downloadBlacklistSheet(ctx, client, server.URL+"/html")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseBlacklistSheet(data); !errors.Is(err, ErrBlacklistSourceLayoutChanged) {
		t.Fatalf("HTML page err = %v, want layout changed", err)
	}
	if err := ctxCanceledDownload(client, server.URL+"/ok"); !errors.Is(err, ErrBlacklistSourceUnavailable) {
		t.Fatalf("canceled download err = %v", err)
	}
}

func ctxCanceledDownload(client *http.Client, url string) error {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := downloadBlacklistSheet(ctx, client, url)
	return err
}
