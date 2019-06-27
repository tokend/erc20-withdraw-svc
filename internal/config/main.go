package config

import (
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
	"gitlab.com/distributed_lab/logan/v3"
)

type config struct {
	transferConfig TransferConfig
	withdrawConfig WithdrawConfig

	getter kv.Getter
	once   comfig.Once
	Horizoner
	Ether
	comfig.Logger
}

type Config interface {
	WithdrawConfig() WithdrawConfig
	TransferConfig() TransferConfig
	Log() *logan.Entry
	Horizoner
	Ether
}

func NewConfig(getter kv.Getter) Config {
	return &config{
		getter:    getter,
		Horizoner: NewHorizoner(getter),
		Ether: NewEther(getter),
		Logger:    comfig.NewLogger(getter, comfig.LoggerOpts{}),
	}
}
