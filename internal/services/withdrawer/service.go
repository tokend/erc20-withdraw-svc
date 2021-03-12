package withdrawer

import (
	"context"
	"github.com/tokend/erc20-withdraw-svc/internal/horizon/getters"
	"github.com/tokend/erc20-withdraw-svc/internal/horizon/submit"
	"github.com/tokend/erc20-withdraw-svc/internal/services/oracle"
	"github.com/tokend/erc20-withdraw-svc/internal/services/verifier"
	"github.com/tokend/erc20-withdraw-svc/internal/services/watchlist"
	"gitlab.com/distributed_lab/logan/v3"
)

func (s *Service) Run(ctx context.Context) {
	s.log.Info("service is started")
	go s.assetWatcher.Run(ctx)

	s.Add(2)
	go s.spawner(ctx)
	go s.cancellor(ctx)
	s.Wait()
}

func (s *Service) spawner(ctx context.Context) {
	defer s.Done()
	for asset := range s.assetsToAdd {
		if _, ok := s.spawned.Load(asset.ID); !ok {
			s.spawn(ctx, asset)
		}
	}
}

func (s *Service) cancellor(ctx context.Context) {
	defer s.Done()
	for asset := range s.assetsToRemove {
		if raw, ok := s.spawned.Load(asset); ok {
			cancelFunc := raw.(context.CancelFunc)
			cancelFunc()
			s.spawned.Delete(asset)
		}
	}
}

func (s *Service) spawn(ctx context.Context, details watchlist.Details) {

	oracleService := oracle.New(oracle.Opts{
		Builder:   s.builder,
		Log:       s.log,
		Config:    s.config,
		Submitter: submit.New(s.config.Horizon()),
		Client:    *s.config.EthClient(),
		Asset:     details,

		Streamer: getters.NewDefaultCreateWithdrawRequestHandler(s.config.Horizon()),
	})
	verifierService := verifier.New(verifier.Opts{
		Builder:   s.builder,
		Log:       s.log,
		Config:    s.config,
		Submitter: submit.New(s.config.Horizon()),
		Client:    *s.config.EthClient(),
		Asset:     details,

		Streamer: getters.NewDefaultCreateWithdrawRequestHandler(s.config.Horizon()),
	})

	innerCtx, cancelFunc := context.WithCancel(ctx)
	s.spawned.Store(details.Asset.ID, cancelFunc)

	go oracleService.Run(innerCtx)
	go verifierService.Run(innerCtx)

	s.log.WithFields(logan.F{
		"asset_code": details.ID,
	}).Info("Started listening for withdrawals")
}
