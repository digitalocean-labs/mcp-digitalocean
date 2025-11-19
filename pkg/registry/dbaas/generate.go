package dbaas

//go:generate mockgen -destination=./mocks/databases_service_mock.go -package=mocks github.com/digitalocean/godo DatabasesService
