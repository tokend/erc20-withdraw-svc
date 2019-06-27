package config

import (
	"gitlab.com/distributed_lab/figure"
	"gitlab.com/distributed_lab/kit/kv"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/tokend/keypair"
	"gitlab.com/tokend/keypair/figurekeypair"
)

type WithdrawConfig struct {
	Signer keypair.Full    `fig:"signer"`
	Owner  keypair.Address `fig:"owner"`
}

func (c *config) WithdrawConfig() WithdrawConfig {
	c.once.Do(func() interface{} {
		var result WithdrawConfig

		err := figure.
			Out(&result).
			With(figure.BaseHooks, figurekeypair.Hooks).
			From(kv.MustGetStringMap(c.getter, "withdraw")).
			Please()
		if err != nil {
			panic(errors.Wrap(err, "failed to figure out withdraw"))
		}

		c.withdrawConfig = result
		return nil
	})
	return c.withdrawConfig
}
