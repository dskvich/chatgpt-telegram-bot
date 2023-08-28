package digitalocean

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
)

type client struct {
	api *godo.Client
}

func NewClient(token string) *client {
	return &client{
		api: godo.NewFromToken(token),
	}
}

func (c *client) GetBalanceMessage(ctx context.Context) (string, error) {
	b, _, err := c.api.Balance.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("fetching balance: %v", err)
	}

	res := fmt.Sprintf("Server Balance Info:\n\nMonth-To-Date Balance: $%v \nAccount Balance: $%v",
		b.MonthToDateBalance, b.AccountBalance)
	return res, nil
}
