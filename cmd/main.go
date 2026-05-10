package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"project_radar/internal/config"
	"project_radar/internal/mailer"
	"project_radar/internal/ollama"
	"project_radar/internal/simap"
)

const minScore = 50 // tenders below this score are discarded

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	log.Println("=== project-radar started ===")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v\n\nPlease check your .env file (see .env.example)", err)
	}

	simapClient := simap.New(cfg.SimapBaseURL)
	ollamaClient := ollama.New(cfg.OllamaBaseURL, cfg.OllamaModel)
	mailClient := mailer.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.MailFrom, cfg.MailTo)

	// Fetch tenders from simap.ch
	log.Printf("Fetching tenders from the last %d day(s) via %s ...", cfg.LookbackDays, cfg.SimapBaseURL)

	projects, err := simapClient.FetchRecentTendersByCPV(cfg.LookbackDays)
	if err != nil {
		log.Fatalf("simap fetch failed: %v", err)
	}
	log.Printf("Found %d tenders with IT-relevant CPV codes", len(projects))

	if len(projects) == 0 {
		log.Println("No new tenders found. Sending empty digest.")
		sendDigest(mailClient, nil)
		os.Exit(0)
	}

	// AI analysis via local Mistral
	log.Printf("Running AI analysis with Ollama (%s) ...", cfg.OllamaModel)

	var matched []mailer.MatchedTender

	for i, p := range projects {
		title := p.Title.Best()
		if title == "" {
			title = "(no title)"
		}
		procOffice := p.ProcOfficeName.Best()
		location := buildLocation(p)
		simapURL := fmt.Sprintf("https://www.simap.ch/de/publikationen/%s", p.PublicationID)

		// Try to enrich with full description from the detail endpoint
		description := ""
		if detail, err := simapClient.FetchDetail(p.ID); err == nil && detail != nil {
			description = detail.Description.Best()
		}

		log.Printf("[%d/%d] Analysing: %s", i+1, len(projects), title)

		result, err := ollamaClient.Analyze(
			title,
			description,
			p.ProjectSubType,
			procOffice,
			location,
			p.PublicationDate,
		)
		if err != nil {
			log.Printf("  ⚠  Ollama error for %q: %v — skipping", title, err)
			continue
		}

		log.Printf("  Score: %d | Match: %v | %s", result.Score, result.IsMatch, result.Reasoning)

		if result.IsMatch && result.Score >= minScore {
			matched = append(matched, mailer.MatchedTender{
				Title:       title,
				ProcOffice:  procOffice,
				Location:    location,
				PubDate:     p.PublicationDate,
				Score:       result.Score,
				Reasoning:   result.Reasoning,
				SimapURL:    simapURL,
				SubType:     p.ProjectSubType,
				ProcessType: p.ProcessType,
			})
		}

		time.Sleep(500 * time.Millisecond) // avoid overwhelming the local model
	}

	// Send digest email
	log.Printf("Found %d matching tender(s). Sending digest to %s ...", len(matched), cfg.MailTo)
	sendDigest(mailClient, matched)
	log.Println("✅ Done.")
}

func buildLocation(p simap.Project) string {
	city := p.OrderAddress.City.Best()
	canton := p.OrderAddress.CantonID
	switch {
	case city != "" && canton != "":
		return fmt.Sprintf("%s (%s)", city, canton)
	case city != "":
		return city
	case canton != "":
		return canton
	default:
		return "Switzerland"
	}
}

func sendDigest(m *mailer.Mailer, tenders []mailer.MatchedTender) {
	if err := m.Send(tenders); err != nil {
		log.Fatalf("failed to send digest email: %v", err)
	}
}
