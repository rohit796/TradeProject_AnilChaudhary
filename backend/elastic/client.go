package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"trade-reporting-backend/models"
)

type ESClient struct {
	baseURL    string
	indexName  string
	httpClient *http.Client
	isReady    bool
}

var GlobalESClient *ESClient

// InitElasticsearch connects and verifies Elasticsearch cluster
func InitElasticsearch() *ESClient {
	baseURL := os.Getenv("ELASTICSEARCH_URL")
	if baseURL == "" {
		baseURL = "http://localhost:9200"
	}
	indexName := os.Getenv("ELASTICSEARCH_INDEX")
	if indexName == "" {
		indexName = "trade_reports"
	}

	client := &ESClient{
		baseURL:   baseURL,
		indexName: indexName,
		httpClient: &http.Client{
			Timeout: 4 * time.Second,
		},
	}

	// Verify health
	resp, err := client.httpClient.Get(baseURL)
	if err != nil {
		log.Printf("[Elasticsearch] Note: Elasticsearch cluster at %s is not reachable (%v). Reports will query MSSQL directly.", baseURL, err)
		client.isReady = false
	} else {
		defer resp.Body.Close()
		log.Printf("[Elasticsearch] Connected successfully to %s! Index: %s", baseURL, indexName)
		client.isReady = true
		_ = client.ensureIndexMapping()
	}

	GlobalESClient = client
	return client
}

// IsReady returns whether Elasticsearch is reachable
func (es *ESClient) IsReady() bool {
	return es != nil && es.isReady
}

// ensureIndexMapping checks if the Elasticsearch index exists without embedding hardcoded schemas
func (es *ESClient) ensureIndexMapping() error {
	checkURL := fmt.Sprintf("%s/%s", es.baseURL, es.indexName)
	resp, err := es.httpClient.Head(checkURL)
	if err == nil && resp.StatusCode == http.StatusOK {
		return nil // Index already exists
	}

	// Read mapping schema from external JSON file if index needs one-time creation
	mappingFile := os.Getenv("ELASTICSEARCH_MAPPING_FILE")
	if mappingFile == "" {
		mappingFile = `elasticsearch\mappings\trade_reports_mapping.json`
	}

	mappingBytes, err := os.ReadFile(mappingFile)
	if err != nil {
		mappingBytes, err = os.ReadFile(`D:\TradeProject\elasticsearch\mappings\trade_reports_mapping.json`)
	}

	if err == nil && len(mappingBytes) > 0 {
		req, _ := http.NewRequest(http.MethodPut, checkURL, bytes.NewBuffer(mappingBytes))
		req.Header.Set("Content-Type", "application/json")
		createResp, err := es.httpClient.Do(req)
		if err == nil {
			defer createResp.Body.Close()
			log.Printf("[Elasticsearch] Initialized index '%s' using external schema: %s", es.indexName, mappingFile)
			return nil
		}
	}

	log.Printf("[Elasticsearch] Note: Index '%s' is not created yet. Run one-time setup from: elasticsearch/mappings/create_index.txt", es.indexName)
	return nil
}

// IndexTradeReport indexes a single trade into Elasticsearch
func (es *ESClient) IndexTradeReport(ctx context.Context, report *models.TradeReport) error {
	if !es.isReady {
		return fmt.Errorf("elasticsearch not ready")
	}

	docURL := fmt.Sprintf("%s/%s/_doc/%s", es.baseURL, es.indexName, report.TradeID)
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, docURL, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("indexing error (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// SearchTradeReports queries Elasticsearch with boolean filters and sort
func (es *ESClient) SearchTradeReports(ctx context.Context, filter models.TradeFilter) ([]models.TradeReport, int64, error) {
	if !es.isReady {
		return nil, 0, fmt.Errorf("elasticsearch not ready")
	}

	var mustClauses []map[string]interface{}

	if filter.ClientID != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{"client_id": filter.ClientID},
		})
	}
	if filter.Symbol != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{"symbol": filter.Symbol},
		})
	}
	if filter.TradeType != "" && filter.TradeType != "ALL" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{"trade_type": filter.TradeType},
		})
	}

	// Universal partial / substring / case-insensitive search across all keywords and text fields
	if filter.Search != "" {
		searchTerm := strings.TrimSpace(filter.Search)
		upperTerm := strings.ToUpper(searchTerm)
		lowerTerm := strings.ToLower(searchTerm)

		shouldClauses := []map[string]interface{}{
			// Substring wildcard on Trade ID (keyword)
			{"wildcard": map[string]interface{}{"trade_id": map[string]interface{}{"value": "*" + upperTerm + "*"}}},
			{"wildcard": map[string]interface{}{"trade_id": map[string]interface{}{"value": "*" + lowerTerm + "*", "case_insensitive": true}}},
			// Substring wildcard on Order ID (keyword)
			{"wildcard": map[string]interface{}{"order_id": map[string]interface{}{"value": "*" + upperTerm + "*"}}},
			{"wildcard": map[string]interface{}{"order_id": map[string]interface{}{"value": "*" + lowerTerm + "*", "case_insensitive": true}}},
			// Substring wildcard on Stock Symbol (keyword)
			{"wildcard": map[string]interface{}{"symbol": map[string]interface{}{"value": "*" + upperTerm + "*"}}},
			{"wildcard": map[string]interface{}{"symbol": map[string]interface{}{"value": "*" + lowerTerm + "*", "case_insensitive": true}}},
			// Substring wildcard on Client ID and Code (keyword)
			{"wildcard": map[string]interface{}{"client_id": map[string]interface{}{"value": "*" + upperTerm + "*"}}},
			{"wildcard": map[string]interface{}{"client_id": map[string]interface{}{"value": "*" + lowerTerm + "*", "case_insensitive": true}}},
			{"wildcard": map[string]interface{}{"client_code": map[string]interface{}{"value": "*" + upperTerm + "*"}}},
			{"wildcard": map[string]interface{}{"client_code": map[string]interface{}{"value": "*" + lowerTerm + "*", "case_insensitive": true}}},
			// Side & Sector & Status (keyword)
			{"wildcard": map[string]interface{}{"trade_type": map[string]interface{}{"value": "*" + upperTerm + "*"}}},
			{"wildcard": map[string]interface{}{"sector": map[string]interface{}{"value": "*" + lowerTerm + "*", "case_insensitive": true}}},
			{"wildcard": map[string]interface{}{"trade_status": map[string]interface{}{"value": "*" + lowerTerm + "*", "case_insensitive": true}}},
			// Text fields (first_name, last_name, company_name)
			{"match_phrase_prefix": map[string]interface{}{"first_name": searchTerm}},
			{"match": map[string]interface{}{"first_name": searchTerm}},
			{"match_phrase_prefix": map[string]interface{}{"last_name": searchTerm}},
			{"match": map[string]interface{}{"last_name": searchTerm}},
			{"match_phrase_prefix": map[string]interface{}{"company_name": searchTerm}},
			{"match": map[string]interface{}{"company_name": searchTerm}},
			{"query_string": map[string]interface{}{
				"query":            "*" + lowerTerm + "*",
				"fields":           []string{"first_name", "last_name", "company_name"},
				"analyze_wildcard": true,
			}},
		}

		mustClauses = append(mustClauses, map[string]interface{}{
			"bool": map[string]interface{}{
				"should":               shouldClauses,
				"minimum_should_match": 1,
			},
		})
	}

	size := filter.Limit
	if size <= 0 {
		size = 100
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	from := (page - 1) * size

	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "trade_time"
	}
	order := strings.ToLower(filter.Order)
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	queryMap := map[string]interface{}{
		"from": from,
		"size": size,
		"sort": []map[string]interface{}{
			{sortBy: map[string]string{"order": order}},
		},
	}

	if len(mustClauses) > 0 {
		queryMap["query"] = map[string]interface{}{
			"bool": map[string]interface{}{
				"must": mustClauses,
			},
		}
	} else {
		queryMap["query"] = map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
	}

	reqBody, _ := json.Marshal(queryMap)
	searchURL := fmt.Sprintf("%s/%s/_search", es.baseURL, es.indexName)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, searchURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("search failed with code %d", resp.StatusCode)
	}

	var searchResp struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source models.TradeReport `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, 0, err
	}

	var reports []models.TradeReport
	for _, h := range searchResp.Hits.Hits {
		reports = append(reports, h.Source)
	}

	return reports, searchResp.Hits.Total.Value, nil
}

// UpdateClientInTradeReports updates client details across all trade reports in Elasticsearch for a given client_id
func (es *ESClient) UpdateClientInTradeReports(ctx context.Context, client *models.Client) (int64, error) {
	if !es.isReady {
		return 0, fmt.Errorf("elasticsearch not ready")
	}

	updateURL := fmt.Sprintf("%s/%s/_update_by_query?conflicts=proceed&refresh=true", es.baseURL, es.indexName)
	payload := map[string]interface{}{
		"script": map[string]interface{}{
			"source": "ctx._source.first_name = params.first_name; ctx._source.last_name = params.last_name; ctx._source.client_code = params.client_code; ctx._source.account_type = params.account_type",
			"params": map[string]interface{}{
				"first_name":   client.FirstName,
				"last_name":    client.LastName,
				"client_code":  client.ClientCode,
				"account_type": client.AccountType,
			},
		},
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"client_id": client.ClientID,
			},
		},
	}

	reqBody, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, updateURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Updated int64 `json:"updated"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return result.Updated, nil
}

// UpdateStockInTradeReports updates stock details across all trade reports in Elasticsearch for a given symbol
func (es *ESClient) UpdateStockInTradeReports(ctx context.Context, stock *models.Stock) (int64, error) {
	if !es.isReady {
		return 0, fmt.Errorf("elasticsearch not ready")
	}

	updateURL := fmt.Sprintf("%s/%s/_update_by_query?conflicts=proceed&refresh=true", es.baseURL, es.indexName)
	payload := map[string]interface{}{
		"script": map[string]interface{}{
			"source": "ctx._source.company_name = params.company_name; ctx._source.sector = params.sector; ctx._source.exchange = params.exchange",
			"params": map[string]interface{}{
				"company_name": stock.CompanyName,
				"sector":       stock.Sector,
				"exchange":     stock.Exchange,
			},
		},
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"symbol": stock.Symbol,
			},
		},
	}

	reqBody, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, updateURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Updated int64 `json:"updated"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return result.Updated, nil
}
