package provider

import (
	"context"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// GetBalance reads the prepaid balance of an account.
type GetBalance struct{}

// GetBalanceArgs are the lookup arguments.
type GetBalanceArgs struct {
	AccountSid *string `pulumi:"accountSid,optional"`
}

func (a *GetBalanceArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.AccountSid, "SID of the account. Defaults to the provider account.")
}

// GetBalanceResult is the looked-up balance.
type GetBalanceResult struct {
	AccountSid string `pulumi:"accountSid"`
	Balance    string `pulumi:"balance"`
	Currency   string `pulumi:"currency"`
}

func (a *GetBalanceResult) Annotate(an infer.Annotator) {
	an.Describe(&a.AccountSid, "SID of the account.")
	an.Describe(&a.Balance, "The balance as a decimal string, for example 12.3400.")
	an.Describe(&a.Currency, "ISO 4217 currency code of the balance.")
}

func (f *GetBalance) Annotate(a infer.Annotator) {
	a.Describe(&f, "Read the account balance (GET /2010-04-01/Accounts/{AccountSid}/Balance.json).")
}

func (f *GetBalance) Invoke(ctx context.Context, req infer.FunctionRequest[GetBalanceArgs]) (infer.FunctionResponse[GetBalanceResult], error) {
	c, err := clientFor(ctx)
	if err != nil {
		return infer.FunctionResponse[GetBalanceResult]{}, err
	}
	acct := c.acct(req.Input.AccountSid)
	o, err := c.do(ctx, "GET", "api", api2010(acct)+"/Balance.json", nil, nil)
	if err != nil {
		return infer.FunctionResponse[GetBalanceResult]{}, lookupFailure("getBalance", "balance of account "+acct, err)
	}
	return infer.FunctionResponse[GetBalanceResult]{Output: GetBalanceResult{
		AccountSid: acct, Balance: o.str("balance"), Currency: o.str("currency"),
	}}, nil
}
