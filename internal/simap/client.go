package simap

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

var topCPVCodes = []string{
	"72000000", // IT services
	"72200000", // Software programming and consultancy
	"72500000", // Computer-related services
	"72600000", // Computer support and consultancy
	"48000000", // Software packages and information systems
}

// MultiLang is a localised string as returned by the simap API.
type MultiLang struct {
	De *string `json:"de"`
	En *string `json:"en"`
	Fr *string `json:"fr"`
	It *string `json:"it"`
}

// Best returns the best available translation (DE > EN > FR > IT).
func (m MultiLang) Best() string {
	for _, s := range []*string{m.De, m.En, m.Fr, m.It} {
		if s != nil && *s != "" {
			return *s
		}
	}
	return ""
}

func (m MultiLang) HasGerman() bool {
	return m.De != nil && *m.De != ""
}

// OrderAddress holds location information for a tender.
type OrderAddress struct {
	CountryID  string    `json:"countryId"`
	CantonID   string    `json:"cantonId"`
	PostalCode string    `json:"postalCode"`
	City       MultiLang `json:"city"`
}

// Project is a single result entry from the project-search endpoint.
type Project struct {
	ID                string       `json:"id"`
	Title             MultiLang    `json:"title"`
	ProjectNumber     string       `json:"projectNumber"`
	ProjectType       string       `json:"projectType"`
	ProjectSubType    string       `json:"projectSubType"`
	ProcessType       string       `json:"processType"`
	PublicationID     string       `json:"publicationId"`
	PublicationDate   string       `json:"publicationDate"`
	PublicationNumber string       `json:"publicationNumber"`
	PubType           string       `json:"pubType"`
	ProcOfficeName    MultiLang    `json:"procOfficeName"`
	OrderAddress      OrderAddress `json:"orderAddress"`
}

// ProjectDetail holds the full publication detail for a project.
func (p Project) HasGermanContent() bool {
	return p.Title.HasGerman() ||
		p.ProcOfficeName.HasGerman() ||
		p.OrderAddress.City.HasGerman()
}

type ProjectDetail struct {
	ID          string    `json:"id"`
	Title       MultiLang `json:"title"`
	Description MultiLang `json:"description"`
	CPVCode     string    `json:"cpvCode"`
}

// SearchResponse is the top-level response from the project-search endpoint.
type SearchResponse struct {
	Projects   []Project `json:"projects"`
	TotalCount int       `json:"totalCount"`
}

// Client wraps the simap public API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New creates a new simap API client.
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchRecentTendersByCPV fetches tenders for all relevant CPV codes
// published within the last lookbackDays days. Duplicates are deduplicated by ID.
func (c *Client) FetchRecentTendersByCPV(lookbackDays int) ([]Project, error) {
	from := time.Now().AddDate(0, 0, -lookbackDays).Format("2006-01-02")

	seen := map[string]bool{}
	var result []Project

	for _, cpv := range topCPVCodes {
		projects, err := c.searchPage(from, cpv, 50)
		if err != nil {
			fmt.Printf("warning: CPV %s search failed: %v\n", cpv, err)
			continue
		}

		fmt.Printf("CPV %s returned %d projects\n", cpv, len(projects))

		for _, p := range projects {
			if !p.HasGermanContent() {
				continue
			}

			if seen[p.ID] {
				continue
			}

			seen[p.ID] = true
			result = append(result, p)
		}

		time.Sleep(300 * time.Millisecond)
	}

	return result, nil
}

// searchPage performs a single call to the project-search endpoint.
func (c *Client) searchPage(from, cpvCode string, size int) ([]Project, error) {
	params := url.Values{}

	params.Set("newestPublicationFrom", from)
	params.Set("newestPubTypes", "tender")
	params.Set("size", fmt.Sprintf("%d", size))
	params.Set("sort", "publicationDate,desc")

	if cpvCode != "" {
		params.Add("cpvCodes", cpvCode)
	}

	params.Add("lang", "de")

	endpoint := fmt.Sprintf(
		"%s/api/publications/v2/project/project-search?%s",
		c.baseURL,
		params.Encode(),
	)

	fmt.Printf("GET %s\n", endpoint)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "project-radar/1.0 (aiza.ch)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var sr SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}

	return sr.Projects, nil
}

// FetchDetail fetches the full publication detail for a project ID.
// Returns nil (no error) if the detail endpoint is unavailable for this project.
func (c *Client) FetchDetail(projectID string) (*ProjectDetail, error) {
	endpoint := fmt.Sprintf(
		"%s/api/publications/v2/project/%s/project-search-detail?lang=de",
		c.baseURL,
		url.PathEscape(projectID),
	)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "project-radar/1.0 (aiza.ch)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var detail ProjectDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, err
	}

	return &detail, nil
}
