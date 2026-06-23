package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const sentTendersFile = "sent_tenders.json"

// SentTenders holds the list of tenders already sent via email.
type SentTenders struct {
	Tenders []string `json:"sent_tenders"`
}

// LoadSentTenders loads the list of previously sent tender IDs from disk.
func LoadSentTenders() (map[string]bool, error) {
	data, err := os.ReadFile(sentTendersFile)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]bool), nil
		}
		return nil, fmt.Errorf("read sent tenders file: %w", err)
	}

	var st SentTenders
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("parse sent tenders file: %w", err)
	}

	result := make(map[string]bool)
	for _, id := range st.Tenders {
		result[id] = true
	}
	return result, nil
}

// SaveSentTenders saves the list of sent tender IDs to disk.
func SaveSentTenders(publicationIDs []string) error {
	st := SentTenders{
		Tenders: publicationIDs,
	}

	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sent tenders: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(sentTendersFile)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}
	}

	if err := os.WriteFile(sentTendersFile, data, 0644); err != nil {
		return fmt.Errorf("write sent tenders file: %w", err)
	}

	return nil
}

// AppendSentTender adds a new publication ID to the sent list and saves to disk.
func AppendSentTender(publicationID string) error {
	sent, err := LoadSentTenders()
	if err != nil {
		return err
	}

	sent[publicationID] = true

	ids := make([]string, 0, len(sent))
	for id := range sent {
		ids = append(ids, id)
	}

	return SaveSentTenders(ids)
}

// AppendSentTenders adds multiple publication IDs to the sent list and saves to disk.
func AppendSentTenders(publicationIDs []string) error {
	sent, err := LoadSentTenders()
	if err != nil {
		return err
	}

	for _, id := range publicationIDs {
		sent[id] = true
	}

	ids := make([]string, 0, len(sent))
	for id := range sent {
		ids = append(ids, id)
	}

	return SaveSentTenders(ids)
}
