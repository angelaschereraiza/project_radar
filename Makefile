.PHONY: build run run-env tidy cron

build:
	go build -o project-radar ./cmd/main.go

run: build
	./project-radar

run-env: build
	export $$(grep -v '^#' .env | xargs) && ./project-radar

tidy:
	go mod tidy

# Print the cron-job line for a daily run at 08:00
cron:
	@echo "Add this line to your crontab (crontab -e):"
	@echo "0 8 * * * cd $(PWD) && export \$$(grep -v '^#' .env | xargs) && ./project-radar >> /var/log/project-radar.log 2>&1"
