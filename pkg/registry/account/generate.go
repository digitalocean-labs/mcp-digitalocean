package account

//go:generate mockgen -destination=./mocks.go -package account github.com/digitalocean/godo  AccountService,ActionsService,BalanceService,BillingHistoryService,InvoicesService,KeysService
