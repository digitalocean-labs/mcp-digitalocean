package insights

//go:generate mockgen -destination=./mocks.go -package insights github.com/digitalocean/godo UptimeChecksService,MonitoringService
