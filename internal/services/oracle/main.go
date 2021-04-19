package oracle

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"github.com/tokend/erc20-withdraw-svc/internal/config"
	"github.com/tokend/erc20-withdraw-svc/internal/horizon/getters"
	"github.com/tokend/erc20-withdraw-svc/internal/horizon/page"
	"github.com/tokend/erc20-withdraw-svc/internal/horizon/query"
	"github.com/tokend/erc20-withdraw-svc/internal/horizon/submit"
	"github.com/tokend/erc20-withdraw-svc/internal/services/watchlist"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/tokend/go/xdrbuild"
)

const erc20ABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_spender\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"name\":\"\",\"type\":\"uint8\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"name\":\"balance\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"}]"

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
	transferCfg config.TransferConfig
	asset       watchlist.Details

	builder     xdrbuild.Builder
	withdrawals getters.CreateWithdrawRequestHandler
	txSubmitter submit.Interface
	log         *logan.Entry

	key      *ecdsa.PrivateKey
	contract *bind.BoundContract
	client   *ethclient.Client

	decimals uint32
	chainID  *big.Int
}

func New(opts Opts) *Service {
	chainID, err := opts.Client.ChainID(context.Background())
	if err != nil {
		panic(errors.Wrap(err, "failed to get chain id"))
	}

	parsed, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		opts.Log.WithError(err).Fatal("failed to parse contract ABI")
	}

	contract := bind.NewBoundContract(
		opts.Asset.ERC20.Address,
		parsed,
		&opts.Client,
		&opts.Client,
		&opts.Client,
	)

	decimals := new(uint8)
	err = contract.Call(&bind.CallOpts{}, decimals, "decimals")
	if err != nil {
		opts.Log.WithError(err).Error("failed to get decimals of erc20 contract")
		return nil
	}

	key, err := crypto.HexToECDSA(opts.Config.TransferConfig().Seed)
	if err != nil {
		panic(err)
	}

	return &Service{
		client:      &opts.Client,
		log:         opts.Log,
		contract:    contract,
		withdrawCfg: opts.Config.WithdrawConfig(),
		transferCfg: opts.Config.TransferConfig(),
		txSubmitter: opts.Submitter,
		builder:     opts.Builder,
		asset:       opts.Asset,
		key:         key,
		withdrawals: opts.Streamer,
		decimals:    uint32(*decimals),
		chainID:     chainID,
	}
}

func (s *Service) prepare() {
	state := reviewableRequestStatePending
	pendingTasks := fmt.Sprintf("%d", taskTryTransfer)
	pendingTasksNotSet := fmt.Sprintf("%d", taskCheckTxSentSuccess)
	filters := query.CreateWithdrawRequestFilters{
		Asset: &s.asset.ID,
		ReviewableRequestFilters: query.ReviewableRequestFilters{
			State:              &state,
			Reviewer:           &s.asset.Relationships.Owner.Data.ID,
			PendingTasks:       &pendingTasks,
			PendingTasksNotSet: &pendingTasksNotSet,
		},
	}
	s.withdrawals.SetFilters(filters)
	s.withdrawals.SetIncludes(query.CreateWithdrawRequestIncludes{
		ReviewableRequestIncludes: query.ReviewableRequestIncludes{
			RequestDetails: true,
		},
	})
	limit := fmt.Sprintf("%d", requestPageSizeLimit)
	s.withdrawals.SetPageParams(page.Params{Limit: &limit})
}
