package withdrawer

import (
	"github.com/tokend/erc20-withdraw-svc/internal/config"
	"github.com/tokend/erc20-withdraw-svc/internal/horizon"
	"github.com/tokend/erc20-withdraw-svc/internal/horizon/getters"
	"github.com/tokend/erc20-withdraw-svc/internal/services/watchlist"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/tokend/go/xdrbuild"
	"sync"
)

type Service struct {
	assetWatcher   *watchlist.Service
	log            *logan.Entry
	config         config.Config
	builder        xdrbuild.Builder
	spawned        sync.Map
	assetsToAdd    <-chan watchlist.Details
	assetsToRemove <-chan string
	*sync.WaitGroup
}

func New(cfg config.Config) *Service {
	assetWatcher := watchlist.New(watchlist.Opts{
		AssetOwner: cfg.WithdrawConfig().Owner.Address(),
		Streamer:   getters.NewDefaultAssetHandler(cfg.Horizon()),
		Log:        cfg.Log(),
	})
	builder, err := horizon.NewConnector(cfg.Horizon()).Builder()
	if err != nil {
		cfg.Log().WithError(err).Fatal("failed to make builder")
	}


	return &Service{
		log:            cfg.Log(),
		config:         cfg,
		assetWatcher:   assetWatcher,
		assetsToAdd:    assetWatcher.GetToAdd(),
		assetsToRemove: assetWatcher.GetToRemove(),
		spawned:        sync.Map{},
		builder:        *builder,
		WaitGroup:      &sync.WaitGroup{},
	}
}
