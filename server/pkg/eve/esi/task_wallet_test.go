package esi

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWalletTaskExecuteUsesFromIDPaginationForTransactions(t *testing.T) {
	db := newWalletTaskTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	var transactionQueries []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch req.URL.Path {
		case "/characters/9001/wallet/":
			_, _ = w.Write([]byte(`100.5`))
		case "/characters/9001/wallet/journal":
			w.Header().Set("X-Pages", "1")
			_, _ = w.Write([]byte(`[]`))
		case "/characters/9001/wallet/transactions":
			transactionQueries = append(transactionQueries, req.URL.RawQuery)
			fromID := req.URL.Query().Get("from_id")
			switch fromID {
			case "":
				_, _ = w.Write([]byte(`[
					{"transaction_id":300,"date":"2026-04-10T00:00:00Z","location_id":6001,"type_id":34,"unit_price":5.5,"quantity":10,"client_id":7001,"is_buy":true,"is_personal":true,"journal_ref_id":8001},
					{"transaction_id":200,"date":"2026-04-09T00:00:00Z","location_id":6002,"type_id":35,"unit_price":6.5,"quantity":20,"client_id":7002,"is_buy":false,"is_personal":true,"journal_ref_id":8002}
				]`))
			case "200":
				_, _ = w.Write([]byte(`[
					{"transaction_id":150,"date":"2026-04-08T00:00:00Z","location_id":6003,"type_id":36,"unit_price":7.5,"quantity":30,"client_id":7003,"is_buy":true,"is_personal":false,"journal_ref_id":8003},
					{"transaction_id":100,"date":"2026-04-07T00:00:00Z","location_id":6004,"type_id":37,"unit_price":8.5,"quantity":40,"client_id":7004,"is_buy":false,"is_personal":false,"journal_ref_id":8004}
				]`))
			case "100":
				_, _ = w.Write([]byte(`[]`))
			default:
				t.Fatalf("unexpected from_id query: %q", fromID)
			}
		default:
			t.Fatalf("unexpected request path: %s", req.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	task := &WalletTask{}
	err := task.Execute(&TaskContext{
		CharacterID: 9001,
		AccessToken: "token",
		Client:      NewClientWithConfig(server.URL, ""),
	})
	if err != nil {
		t.Fatalf("execute wallet task: %v", err)
	}

	if !slices.Equal(transactionQueries, []string{"", "from_id=200", "from_id=100"}) {
		t.Fatalf("unexpected transaction query sequence: %v", transactionQueries)
	}

	var count int64
	if err := db.Model(&model.EVECharacterWalletTransaction{}).Count(&count).Error; err != nil {
		t.Fatalf("count wallet transactions: %v", err)
	}
	if count != 4 {
		t.Fatalf("wallet transaction row count = %d, want 4", count)
	}
}

func TestWalletTaskExecuteDeduplicatesDuplicateTransactionIDs(t *testing.T) {
	db := newWalletTaskTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch req.URL.Path {
		case "/characters/9001/wallet/":
			_, _ = w.Write([]byte(`100.5`))
		case "/characters/9001/wallet/journal":
			w.Header().Set("X-Pages", "1")
			_, _ = w.Write([]byte(`[]`))
		case "/characters/9001/wallet/transactions":
			fromID := req.URL.Query().Get("from_id")
			switch fromID {
			case "":
				_, _ = w.Write([]byte(`[
					{"transaction_id":300,"date":"2026-04-10T00:00:00Z","location_id":6001,"type_id":34,"unit_price":5.5,"quantity":10,"client_id":7001,"is_buy":true,"is_personal":true,"journal_ref_id":8001},
					{"transaction_id":200,"date":"2026-04-09T00:00:00Z","location_id":6002,"type_id":35,"unit_price":6.5,"quantity":20,"client_id":7002,"is_buy":false,"is_personal":true,"journal_ref_id":8002}
				]`))
			case "200":
				_, _ = w.Write([]byte(`[
					{"transaction_id":200,"date":"2026-04-09T00:00:00Z","location_id":6002,"type_id":35,"unit_price":6.5,"quantity":20,"client_id":7002,"is_buy":false,"is_personal":true,"journal_ref_id":8002},
					{"transaction_id":100,"date":"2026-04-07T00:00:00Z","location_id":6004,"type_id":37,"unit_price":8.5,"quantity":40,"client_id":7004,"is_buy":false,"is_personal":false,"journal_ref_id":8004}
				]`))
			case "100":
				_, _ = w.Write([]byte(`[]`))
			default:
				t.Fatalf("unexpected from_id query: %q", fromID)
			}
		default:
			t.Fatalf("unexpected request path: %s", req.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	task := &WalletTask{}
	if err := task.Execute(&TaskContext{
		CharacterID: 9001,
		AccessToken: "token",
		Client:      NewClientWithConfig(server.URL, ""),
	}); err != nil {
		t.Fatalf("execute wallet task: %v", err)
	}

	var ids []int64
	if err := db.Model(&model.EVECharacterWalletTransaction{}).Order("transaction_id desc").Pluck("transaction_id", &ids).Error; err != nil {
		t.Fatalf("list transaction ids: %v", err)
	}
	if !slices.Equal(ids, []int64{300, 200, 100}) {
		t.Fatalf("wallet transaction ids = %v, want [300 200 100]", ids)
	}
}

func TestWalletTaskExecuteStopsOnDuplicateOnlyTailPage(t *testing.T) {
	db := newWalletTaskTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	var transactionQueries []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch req.URL.Path {
		case "/characters/9001/wallet/":
			_, _ = w.Write([]byte(`100.5`))
		case "/characters/9001/wallet/journal":
			w.Header().Set("X-Pages", "1")
			_, _ = w.Write([]byte(`[]`))
		case "/characters/9001/wallet/transactions":
			transactionQueries = append(transactionQueries, req.URL.RawQuery)
			fromID := req.URL.Query().Get("from_id")
			switch fromID {
			case "":
				_, _ = w.Write([]byte(`[
					{"transaction_id":300,"date":"2026-04-10T00:00:00Z","location_id":6001,"type_id":34,"unit_price":5.5,"quantity":10,"client_id":7001,"is_buy":true,"is_personal":true,"journal_ref_id":8001},
					{"transaction_id":200,"date":"2026-04-09T00:00:00Z","location_id":6002,"type_id":35,"unit_price":6.5,"quantity":20,"client_id":7002,"is_buy":false,"is_personal":true,"journal_ref_id":8002}
				]`))
			case "200":
				_, _ = w.Write([]byte(`[
					{"transaction_id":200,"date":"2026-04-09T00:00:00Z","location_id":6002,"type_id":35,"unit_price":6.5,"quantity":20,"client_id":7002,"is_buy":false,"is_personal":true,"journal_ref_id":8002}
				]`))
			default:
				t.Fatalf("unexpected from_id query: %q", fromID)
			}
		default:
			t.Fatalf("unexpected request path: %s", req.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	task := &WalletTask{}
	if err := task.Execute(&TaskContext{
		CharacterID: 9001,
		AccessToken: "token",
		Client:      NewClientWithConfig(server.URL, ""),
	}); err != nil {
		t.Fatalf("execute wallet task: %v", err)
	}

	if !slices.Equal(transactionQueries, []string{"", "from_id=200"}) {
		t.Fatalf("unexpected transaction query sequence: %v", transactionQueries)
	}

	var ids []int64
	if err := db.Model(&model.EVECharacterWalletTransaction{}).Order("transaction_id desc").Pluck("transaction_id", &ids).Error; err != nil {
		t.Fatalf("list transaction ids: %v", err)
	}
	if !slices.Equal(ids, []int64{300, 200}) {
		t.Fatalf("wallet transaction ids = %v, want [300 200]", ids)
	}
}

func newWalletTaskTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:wallet_task_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.EVECharacterWallet{},
		&model.EVECharacterWalletJournal{},
		&model.EVECharacterWalletTransaction{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}
