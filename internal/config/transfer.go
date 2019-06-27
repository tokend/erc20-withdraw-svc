package config

import (
	"gitlab.com/distributed_lab/figure"
	"gitlab.com/distributed_lab/kit/kv"
	"gitlab.com/distributed_lab/logan/v3/errors"
)

type TransferConfig struct {
	Seed string `fig:"seed"`
	Address string `fig:"address"`
}

func (c *config) TransferConfig() TransferConfig {
	c.once.Do(func() interface{} {
		var result TransferConfig

		err := figure.Out(&result).
			With(figure.BaseHooks).
			From(kv.MustGetStringMap(c.getter, "transfer")).
			Please()
		if err != nil {
			panic(errors.Wrap(err, "failed to figure out transfer"))
		}
		c.transferConfig = result
		return nil
	})
	return c.transferConfig
}