package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

const (
	APIEndpoint  = "https://8j5baasof2.execute-api.us-west-2.amazonaws.com/production/swechallenge/list"
	BearerToken  = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdHRlbXB0cyI6MTYsImVtYWlsIjoicGFibG9hZ3VhdGlAZ21haWwuY29tIiwiZXhwIjoxNzQ5MzE4NTg4LCJpZCI6IjAiLCJwYXNzd29yZCI6IicgT1IgJzEnPScxIn0.Aw-Wss9QuY_SrgUEkEuxA9OfJ6NdEccHzqujmx7gbZ8"
	DBConnString = "postgresql://root@localhost:26257/stock_data?sslmode=disable"
)

// enableCors wraps an http.Handler to add CORS headers
func enableCors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			// Preflight request, no further handling
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

type StockItem struct {
	Ticker     string `json:"ticker"`
	Company    string `json:"company"`
	Brokerage  string `json:"brokerage"`
	Action     string `json:"action"`
	RatingFrom string `json:"rating_from"`
	RatingTo   string `json:"rating_to"`
	TargetFrom string `json:"target_from"` // e.g. "$4.20"
	TargetTo   string `json:"target_to"`   // e.g. "$4.70"
	Time       string `json:"time"`        // e.g. "2025-01-13T00:30:05.813548892Z"
}

type APIResponse struct {
	Items    []StockItem `json:"items"`
	NextPage string      `json:"next_page"`
}

var insertStmt = `
		INSERT INTO stock_info (
		ticker, company, brokerage, action,
		rating_from, rating_to, target_from, target_to,
		time
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

func main() {
	mode := flag.String("mode", "serve", "Mode to run: 'fetch' to load data, 'serve' to start HTTP API")
	flag.Parse()

	// Open DB connection
	db, err := sql.Open("postgres", DBConnString)
	if err != nil {
		log.Fatalf("DB open error: %v", err)
	}
	defer db.Close()

	switch *mode {
	case "fetch":
		executeFetch(db)
	case "serve":
		startServer(db)
	default:
		log.Fatalf("Unknown mode '%s'; use 'fetch' or 'serve'", *mode)
	}
}
func executeFetch(db *sql.DB) {
	log.Println("Starting data fetch...")
	// Prepare statement
	prep, err := db.Prepare(insertStmt)
	if err != nil {
		log.Fatalf("Prepare insert error: %v", err)
	}
	defer prep.Close()

	// Load all pages
	if err := fetchAndStoreAllPages(db, prep); err != nil {
		log.Fatalf("Fetch/store error: %v", err)
	}
	log.Println("Data fetch complete.")
}
func startServer(db *sql.DB) {
	mux := http.NewServeMux()
	mux.HandleFunc("/stocks", func(w http.ResponseWriter, r *http.Request) {
		handleStocks(w, r, db)
	})
	mux.HandleFunc("/stocks/", func(w http.ResponseWriter, r *http.Request) {
		handleStock(w, r, db)
	})

	addr := ":8081"

	log.Printf("Starting HTTP API on %s...", addr)
	handler := enableCors(mux)
	log.Fatal(http.ListenAndServe(addr, handler))
}
func fetchAndStoreAllPages(db *sql.DB, prep *sql.Stmt) error {

	nextKey := ""
	for {
		// Build URL (if nextKey is empty, call without query param)
		url := APIEndpoint
		if nextKey != "" {
			url = APIEndpoint + "?next_page=" + nextKey
		}

		// Perform HTTP GET
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+BearerToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("http request error: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return fmt.Errorf("unexpected HTTP status: %s", resp.Status)
		}

		var apiResp APIResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			resp.Body.Close()
			return fmt.Errorf("decoding JSON: %w", err)
		}
		resp.Body.Close()

		// If no items, we’re done
		if len(apiResp.Items) == 0 {
			break
		}

		// Insert each item
		for _, item := range apiResp.Items {
			if err := insertStockItem(prep, &item); err != nil {
				log.Printf("warning: failed to insert ticker %s: %v", item.Ticker, err)
			}
		}

		// If next_page is empty, break; otherwise, loop again
		if apiResp.NextPage == "" {
			break
		}
		nextKey = apiResp.NextPage
	}

	return nil
}

// insertStockItem parses fields and executes the prepared INSERT statement.
func insertStockItem(prep *sql.Stmt, item *StockItem) error {
	// 1. Parse the target_from string (strip "$")
	var tf *float64
	if item.TargetFrom != "" {
		cleaned := strings.ReplaceAll(item.TargetFrom, ",", "")
		cleaned = strings.TrimPrefix(cleaned, "$")
		parsed, err := strconv.ParseFloat(cleaned, 64)
		if err != nil {
			return fmt.Errorf("parsing TargetFrom %q: %w", item.TargetFrom, err)
		}
		tf = &parsed
	}
	// 2. Parse the target_to string
	var tt *float64
	if item.TargetTo != "" {
		cleaned := strings.ReplaceAll(item.TargetTo, ",", "")
		cleaned = strings.TrimPrefix(cleaned, "$")
		parsed, err := strconv.ParseFloat(cleaned, 64)
		if err != nil {
			return fmt.Errorf("parsing TargetTo %q: %w", item.TargetTo, err)
		}
		tt = &parsed
	}

	// 3. Parse the raw time
	parsedTime, err := time.Parse(time.RFC3339Nano, item.Time)
	if err != nil {
		return fmt.Errorf("parsing Time %q: %w", item.Time, err)
	}

	// 4. Execute the INSERT (using nil for DECIMAL columns if parsing failed or was empty)
	_, err = prep.Exec(
		item.Ticker,
		item.Company,
		item.Brokerage,
		item.Action,
		item.RatingFrom,
		item.RatingTo,
		tf,
		tt,
		parsedTime,
	)
	if err != nil {
		return fmt.Errorf("exec insert: %w", err)
	}
	return nil
}

// handleStocks returns a list of stocks, supports search, sort, pagination.
func handleStocks(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	q := r.URL.Query()

	// Search across multiple text fields
	search := q.Get("search")

	// Faceted filters (comma-separated lists)
	actions := splitParam(q.Get("action"))
	brokerages := splitParam(q.Get("brokerage"))
	ratingFrom := splitParam(q.Get("rating_from"))
	ratingTo := splitParam(q.Get("rating_to"))

	// Numeric range filters for target_from/to
	minTFStr := q.Get("min_target_from")
	maxTFStr := q.Get("max_target_from")
	minTTStr := q.Get("min_target_to")
	maxTTStr := q.Get("max_target_to")

	// Date range filters (ISO8601 format)
	dateFromStr := q.Get("date_from")
	dateToStr := q.Get("date_to")

	// Sorting and pagination parameters
	sortBy := q.Get("sort")
	if sortBy == "" {
		sortBy = "ticker"
	}
	order := strings.ToUpper(q.Get("order"))
	if order != "DESC" {
		order = "ASC"
	}

	limit := 100
	if v, err := strconv.Atoi(q.Get("limit")); err == nil && v > 0 {
		limit = v
	}
	offset := 0
	if v, err := strconv.Atoi(q.Get("offset")); err == nil && v >= 0 {
		offset = v
	}

	// Build WHERE clauses dynamically
	filters := []string{}
	args := []interface{}{}
	argPos := 1

	if search != "" {
		pattern := "%" + search + "%"
		// Search multiple fields with case-insensitive match
		filters = append(filters, fmt.Sprintf("(ticker ILIKE $%d OR company ILIKE $%d OR brokerage ILIKE $%d OR action ILIKE $%d OR rating_from ILIKE $%d OR rating_to ILIKE $%d)",
			argPos, argPos+1, argPos+2, argPos+3, argPos+4, argPos+5))
		for i := 0; i < 6; i++ {
			args = append(args, pattern)
		}
		argPos += 6
	}

	// Helper to add IN(...) filters
	addInFilter := func(field string, vals []string) {
		if len(vals) == 0 {
			return
		}
		var ph []string
		for _, v := range vals {
			args = append(args, v)
			ph = append(ph, fmt.Sprintf("$%d", len(args)))
		}
		filters = append(filters, fmt.Sprintf("%s IN (%s)", field, strings.Join(ph, ",")))
	}
	addInFilter("action", actions)
	addInFilter("brokerage", brokerages)
	addInFilter("rating_from", ratingFrom)
	addInFilter("rating_to", ratingTo)

	// Numeric filters
	if minTFStr != "" {
		if v, err := strconv.ParseFloat(minTFStr, 64); err == nil {
			filters = append(filters, fmt.Sprintf("target_from >= $%d", argPos))
			args = append(args, v)
			argPos++
		}
	}
	if maxTFStr != "" {
		if v, err := strconv.ParseFloat(maxTFStr, 64); err == nil {
			filters = append(filters, fmt.Sprintf("target_from <= $%d", argPos))
			args = append(args, v)
			argPos++
		}
	}
	if minTTStr != "" {
		if v, err := strconv.ParseFloat(minTTStr, 64); err == nil {
			filters = append(filters, fmt.Sprintf("target_to >= $%d", argPos))
			args = append(args, v)
			argPos++
		}
	}
	if maxTTStr != "" {
		if v, err := strconv.ParseFloat(maxTTStr, 64); err == nil {
			filters = append(filters, fmt.Sprintf("target_to <= $%d", argPos))
			args = append(args, v)
			argPos++
		}
	}

	// Date filters
	if dateFromStr != "" {
		if t, err := time.Parse(time.RFC3339, dateFromStr); err == nil {
			filters = append(filters, fmt.Sprintf("time >= $%d", argPos))
			args = append(args, t)
			argPos++
		}
	}
	if dateToStr != "" {
		if t, err := time.Parse(time.RFC3339, dateToStr); err == nil {
			filters = append(filters, fmt.Sprintf("time <= $%d", argPos))
			args = append(args, t)
			argPos++
		}
	}

	where := ""
	if len(filters) > 0 {
		where = "WHERE " + strings.Join(filters, " AND ")
	}

	// Final SQL
	sqlQuery := fmt.Sprintf(
		"SELECT ticker, company, brokerage, action, rating_from, rating_to, target_from, target_to, time FROM stock_info %s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		where, sortBy, order, len(args)+1, len(args)+2,
	)
	args = append(args, limit, offset)

	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	results := []StockItem{}
	for rows.Next() {
		var s StockItem
		var tf, tt sql.NullString
		var t time.Time
		if err := rows.Scan(
			&s.Ticker, &s.Company, &s.Brokerage, &s.Action,
			&s.RatingFrom, &s.RatingTo, &tf, &tt, &t,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if tf.Valid {
			s.TargetFrom = tf.String
		}
		if tt.Valid {
			s.TargetTo = tt.String
		}
		s.Time = t.Format(time.RFC3339Nano)
		results = append(results, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"items": results})
}

// handleStock returns the latest record for a given ticker.
func handleStock(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	ticker := strings.TrimPrefix(r.URL.Path, "/stocks/")
	if ticker == "" {
		http.Error(w, "ticker required", http.StatusBadRequest)
		return
	}
	var s StockItem
	var tf, tt sql.NullString
	var t time.Time
	err := db.QueryRow(
		"SELECT ticker, company, brokerage, action, rating_from, rating_to, target_from, target_to, time FROM stock_info WHERE ticker=$1 ORDER BY time DESC LIMIT 1",
		ticker,
	).Scan(&s.Ticker, &s.Company, &s.Brokerage, &s.Action, &s.RatingFrom, &s.RatingTo, &tf, &tt, &t)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tf.Valid {
		s.TargetFrom = tf.String
	}
	if tt.Valid {
		s.TargetTo = tt.String
	}
	s.Time = t.Format(time.RFC3339Nano)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}

func splitParam(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	var out []string
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
