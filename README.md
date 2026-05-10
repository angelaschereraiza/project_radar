# project-radar

Fetches new IT tenders daily from [simap.ch](https://www.simap.ch), analyses them with a locally
running [Mistral](https://mistral.ai) model via [Ollama](https://ollama.ai), and sends matching
tenders as an HTML digest email.

Built for: **Aiza GmbH / Angela Scherer** — Freelance Software & DevOps Engineer

---

## Prerequisites

| Requirement | Version |
|---|---|
| Go | ≥ 1.22 |
| Ollama | latest |
| Mistral model | `ollama pull mistral` |

No external Go dependencies — stdlib only.

---

## Quick start

### 1. Install Ollama and pull Mistral

```bash
curl -fsSL https://ollama.ai/install.sh | sh
ollama pull mistral
# verify
ollama run mistral "Hello"
```

### 2. Configure

```bash
cp .env.example .env
nano .env   # fill in your SMTP settings
```

### 3. Build and run

```bash
go mod tidy
make build
make run-env
```

---

## How it works

```
simap.ch API
    │
    ├─ CPV filter  (IT / software CPV codes: 72xxxxxx, 48xxxxxx)
    │
    ▼
Tender list  (metadata + description from detail endpoint)
    │
    ▼
Ollama / Mistral  (local LLM — no data leaves your machine)
    │  Prompt: does this tender match the freelancer profile?
    │  Output: { is_match, score (0–100), reasoning }
    │
    ├─ score ≥ 50 → match
    │
    ▼
HTML digest email  (all matches with score, reasoning, and simap.ch link)
```

---

## Environment variables

| Variable | Description | Default |
|---|---|---|
| `SIMAP_BASE_URL` | Simap API base URL | `https://www.simap.ch` |
| `LOOKBACK_DAYS` | Days back to scan | `1` |
| `OLLAMA_BASE_URL` | Ollama API URL | `http://localhost:11434` |
| `OLLAMA_MODEL` | Model name | `mistral` |
| `SMTP_HOST` | SMTP server hostname | — |
| `SMTP_PORT` | SMTP port | `587` |
| `SMTP_USER` | SMTP username | — |
| `SMTP_PASSWORD` | SMTP password | — |
| `MAIL_FROM` | Sender address | — |
| `MAIL_TO` | Recipient address | — |

---

## Daily cron job (08:00)

```bash
crontab -e
```

Add:
```
0 8 * * * cd /path/to/project-radar && export $(grep -v '^#' .env | xargs) && ./project-radar >> /var/log/project-radar.log 2>&1
```

Or use `make cron` to print the exact line for your current directory.

---

## CPV codes scanned

| Code | Description |
|---|---|
| 72000000 | IT services — consulting, software, internet, support |
| 72200000 | Software programming and consultancy |
| 72500000 | Computer-related services |
| 72600000 | Computer support and consultancy |
| 48000000 | Software packages and information systems |

---

## Project structure

```
project-radar/
├── cmd/main.go                      ← entry point / pipeline orchestration
├── internal/
│   ├── config/config.go             ← configuration via environment variables
│   ├── simap/client.go              ← simap.ch API client (CPV-filtered)
│   ├── ollama/client.go             ← local Mistral via Ollama
│   └── mailer/mailer.go             ← HTML digest email via SMTP
├── .env.example                     ← configuration template
├── Makefile
└── README.md
```

---

> Disclaimer: "This is not an official publication. The authoritative data is published on www.simap.ch."
