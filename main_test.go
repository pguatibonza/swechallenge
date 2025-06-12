package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// --- fetchCurrentPrice tests ---
// rewriteTransport redirects requests to our httptest.Server
type rewriteTransport struct {
	orig   http.RoundTripper
	target *url.URL
}

func (r *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Copy original path+query
	req.URL.Scheme = r.target.Scheme
	req.URL.Host = r.target.Host
	return r.orig.RoundTrip(req)
}

// It implements http.RoundTripper
type RoundTripFunc func(req *http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// stubHTTPClient replaces http.DefaultClient.Transport to return the given body and status
func stubHTTPClient(body string, status int) (restore func()) {
	oldClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: status,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": {"application/json"}},
			}, nil
		}),
	}
	return func() { http.DefaultClient = oldClient }
}

// --- Tests for insertStockItem ---
func TestInsertStockItem_Success(t *testing.T) {
	// stub HTTP to return a fixed price
	body := `{"chart":{"result":[{"meta":{"regularMarketPrice":123.45}}],"error":null}}`
	restore := stubHTTPClient(body, http.StatusOK)
	defer restore()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Expect prepare and exec with correct args
	mock.ExpectPrepare("INSERT INTO stock_info").
		ExpectExec().
		WithArgs(
			"TCK", "Comp", "Brok", "Act", "RF", "RT",
			sqlmock.AnyArg(), // parsed target_from
			sqlmock.AnyArg(), // parsed target_to
			sqlmock.AnyArg(), // parsed time.Time
			sqlmock.AnyArg(), // fetched current_price
		).WillReturnResult(sqlmock.NewResult(1, 1))

	stmt, err := db.Prepare("INSERT INTO stock_info")
	assert.NoError(t, err)

	item := &StockItem{
		Ticker:     "TCK",
		Company:    "Comp",
		Brokerage:  "Brok",
		Action:     "Act",
		RatingFrom: "RF",
		RatingTo:   "RT",
		TargetFrom: "$100.00",
		TargetTo:   "$200.00",
		Time:       time.Now().Format(time.RFC3339Nano),
	}

	err = insertStockItem(stmt, item)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertStockItem_ParseError_TargetFrom(t *testing.T) {
	item := &StockItem{TargetFrom: "not-a-number", Time: time.Now().Format(time.RFC3339Nano)}
	err := insertStockItem(&sql.Stmt{}, item)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing TargetFrom")
}

func TestInsertStockItem_NoTargets(t *testing.T) {
	// stub HTTP client
	body := `{"chart":{"result":[{"meta":{"regularMarketPrice":50.0}}],"error":null}}`
	restore := stubHTTPClient(body, http.StatusOK)
	defer restore()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Expect nil pointers for target columns
	mock.ExpectPrepare("INSERT INTO stock_info").
		ExpectExec().
		WithArgs(
			"TCK", "Comp", "Brok", "Act", "RF", "RT",
			nil,              // no target_from
			nil,              // no target_to
			sqlmock.AnyArg(), // parsed time.Time
			sqlmock.AnyArg(), // fetched current_price
		).WillReturnResult(sqlmock.NewResult(1, 1))

	stmt, err := db.Prepare("INSERT INTO stock_info")
	assert.NoError(t, err)

	item := &StockItem{
		Ticker:     "TCK",
		Company:    "Comp",
		Brokerage:  "Brok",
		Action:     "Act",
		RatingFrom: "RF",
		RatingTo:   "RT",
		TargetFrom: "",
		TargetTo:   "",
		Time:       time.Now().Format(time.RFC3339Nano),
	}

	err = insertStockItem(stmt, item)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFetchCurrentPrice_Success(t *testing.T) {
	// Mock Yahoo Finance JSON
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"chart":{"result":[{"meta":{"regularMarketPrice":123.45}}]}}`)
	}))
	defer ts.Close()

	// Override transport
	orig := http.DefaultTransport
	defer func() { http.DefaultTransport = orig }()
	tu, _ := url.Parse(ts.URL)
	http.DefaultTransport = &rewriteTransport{orig: orig, target: tu}

	price, err := fetchCurrentPrice("ANY")
	assert.NoError(t, err)
	assert.Equal(t, 123.45, price)
}

func TestFetchCurrentPrice_HTTPError(t *testing.T) {
	// Server returns 500
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "internal error")
	}))
	defer ts.Close()

	orig := http.DefaultTransport
	defer func() { http.DefaultTransport = orig }()
	tu, _ := url.Parse(ts.URL)
	http.DefaultTransport = &rewriteTransport{orig: orig, target: tu}

	_, err := fetchCurrentPrice("ANY")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

// --- handleStock tests ---
func TestHandleStock_NotFound(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	// Expect QueryRow to return no rows
	mock.ExpectQuery("SELECT ticker, company, brokerage").
		WithArgs("ZZZ").
		WillReturnError(sql.ErrNoRows)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stocks/ZZZ", nil)
	handleStock(recorder, req, db)

	res := recorder.Result()
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}

func TestHandleStock_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	// Provide a valid row
	row := sqlmock.NewRows([]string{"ticker", "company", "brokerage", "action", "rating_from", "rating_to", "target_from", "target_to", "time"}).
		AddRow("XYZ", "X Co", "Brok", "reiterated", "Hold", "Hold", "$1", "$2", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

	mock.ExpectQuery("SELECT ticker, company, brokerage").
		WithArgs("XYZ").
		WillReturnRows(row)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stocks/XYZ", nil)
	handleStock(recorder, req, db)

	res := recorder.Result()
	assert.Equal(t, http.StatusOK, res.StatusCode)

	var item StockItem
	err := json.NewDecoder(res.Body).Decode(&item)
	assert.NoError(t, err)
	assert.Equal(t, "XYZ", item.Ticker)
	assert.Equal(t, "X Co", item.Company)
	assert.Equal(t, "Hold", item.RatingFrom)
}

// --- Tests for handleStock detail ---
func TestHandleStock_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("stock_info").
		WithArgs("TCK").
		WillReturnError(fmt.Errorf("detail error"))

	req := httptest.NewRequest("GET", "/stocks/TCK", nil)
	w := httptest.NewRecorder()
	handleStock(w, req, db)
	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	body, _ := io.ReadAll(res.Body)
	assert.Contains(t, string(body), "detail error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- Tests for handleStocks ---
func TestHandleStocks_Success_NoFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"ticker", "company", "brokerage", "action",
		"rating_from", "rating_to", "target_from", "target_to", "time",
	}).AddRow(
		"T1", "C1", "B1", "A1", "RF1", "RT1", "$1.00", "$2.00", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	)
	mock.ExpectQuery("stock_info").WillReturnRows(rows)

	req := httptest.NewRequest("GET", "/stocks", nil)
	w := httptest.NewRecorder()
	handleStocks(w, req, db)
	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	var resp struct {
		Items []StockItem `json:"items"`
	}
	err = json.NewDecoder(res.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "T1", resp.Items[0].Ticker)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- handleRecommend tests ---
func TestHandleRecommend(t *testing.T) {
	// Mock DB with sqlmock
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Prepare rows: two tickers with different composites
	t1 := sqlmock.NewRows([]string{"ticker", "company", "brokerage", "rating_from", "rating_to", "target_from", "target_to", "current_price"}).
		AddRow("A", "CoA", "B1", "Buy", "Buy", 10.0, 12.0, 5.0).
		AddRow("A", "CoB", "B2", "Sell", "Sell", 20.0, 22.0, 10.0)
	// Expect query
	mock.ExpectQuery(`SELECT DISTINCT ON \(ticker\)`).WillReturnRows(t1)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/recommend", nil)
	handleRecommend(recorder, req, db)

	res := recorder.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)

	var recs []RecResult
	err = json.NewDecoder(res.Body).Decode(&recs)
	assert.NoError(t, err)
	// Should have 2 entries sorted by composite descending (AAA has higher upside)
	assert.Len(t, recs, 2)
	assert.Equal(t, "A", recs[0].Ticker)
	assert.Equal(t, "A", recs[1].Ticker)

	// Ensure expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}
