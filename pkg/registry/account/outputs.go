package account

import (
	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// Output contracts for this package's tools. Each var is used twice — Schema()
// at registration, Result() in the handler — so the declared outputSchema and
// the emitted structuredContent cannot disagree. Collections go under the
// plural resource name, single resources under the singular one.
//
// key-delete stays text-only: a confirmation message has no payload.
var (
	accountOut        = common.NewOutput[*godo.Account]("account")
	actionOut         = common.NewOutput[*godo.Action]("action")
	actionsOut        = common.NewOutput[[]godo.Action]("actions")
	balanceOut        = common.NewOutput[*godo.Balance]("balance")
	billingHistoryOut = common.NewOutput[*godo.BillingHistory]("billing_history")
	invoiceOut        = common.NewOutput[*godo.Invoice]("invoice")
	invoiceListOut    = common.NewOutput[*godo.InvoiceList]("invoices")
	keyOut            = common.NewOutput[*godo.Key]("key")
	keysOut           = common.NewOutput[[]godo.Key]("keys")
)
