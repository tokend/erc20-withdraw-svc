package verifier

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokend/erc20-withdraw-svc/internal/config"
	"github.com/tokend/erc20-withdraw-svc/internal/horizon/getters"
	"github.com/tokend/erc20-withdraw-svc/internal/horizon/submit"
	"github.com/tokend/erc20-withdraw-svc/internal/services/watchlist"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/tokend/go/xdrbuild"
)

type Opts struct {
	Client ethclient.Client

	Submitter submit.Interface
	Builder   xdrbuild.Builder
	Log       *logan.Entry
	Streamer  getters.CreateWithdrawRequestHandler
	Config    config.Config
	Asset     watchlist.Details
}

type Service struct {
	withdrawCfg config.WithdrawConfig
	ethCfg      config.TransferConfig
	asset       watchlist.Details

	builder     xdrbuild.Builder
	withdrawals getters.CreateWithdrawRequestHandler
	txSubmitter submit.Interface
	log         *logan.Entry

	client *ethclient.Client
}

func New(opts Opts) *Service {

	return &Service{
		client:      &opts.Client,
		log:         opts.Log,
		withdrawCfg: opts.Config.WithdrawConfig(),
		ethCfg:      opts.Config.TransferConfig(),
		txSubmitter: opts.Submitter,
		builder:     opts.Builder,
		asset:       opts.Asset,

		withdrawals: opts.Streamer,
	}
}
