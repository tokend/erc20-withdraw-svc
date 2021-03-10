package oracle

import (
	"context"
	"encoding/json"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tokend/erc20-withdraw-svc/internal/services/watchlist"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	regources "gitlab.com/tokend/regources/generated"
	"math/big"
)

const (
	taskTryTransfer        uint32 = 2048 // 2^11
	taskCheckTxSentSuccess uint32 = 4096 // 2^12
	taskCheckTxConfirmed   uint32 = 8192 // 2^13

	//Request state
	reviewableRequestStatePending = 1
	//page size
	requestPageSizeLimit = 10

	invalidDetails       = "Invalid creator details"
	invalidTargetAddress = "Invalid target address"
	tooSmallAmount       = "Withdrawn amount too small"
	transferFailed       = "Transfer failed"
)

type PreSentDetails struct {
	TargetAddress string `json:"address"`
}

func (s *Service) sendWithdraw(ctx context.Context, request regources.ReviewableRequest, details *regources.CreateWithdrawRequest) error {
	detailsbb := []byte(details.Attributes.CreatorDetails)
	withdrawDetails := PreSentDetails{}
	err := json.Unmarshal(detailsbb, &withdrawDetails)
	if err != nil {
		s.log.WithField("request_id", request.ID).WithError(err).Warn("Unable to unmarshal creator details")
		return s.permanentReject(ctx, request, invalidDetails)
	}

	if withdrawDetails.TargetAddress == "" {
		s.log.
			WithField("creator_details", details.Attributes.CreatorDetails).
			WithError(err).
			Warn("address missing")
		return s.permanentReject(ctx, request, invalidTargetAddress)
	}

	err = s.approveRequest(ctx, request, taskCheckTxSentSuccess, taskTryTransfer, map[string]interface{}{})
	if err != nil {
		return errors.Wrap(err, "failed to review request first time", logan.F{"request_id": request.ID})
	}

	transferAmount := prepareAmount(s.asset, s.decimals, uint64(details.Attributes.Amount))
	if transferAmount.Uint64() == 0 {
		return s.permanentReject(ctx, request, tooSmallAmount)
	}

	transaction, err := s.callTransfer(ctx, transferAmount, withdrawDetails.TargetAddress)
	if err != nil {
		s.log.WithError(err).Error("Transfer failed - rejecting withdraw request")
		return s.permanentReject(ctx, request, transferFailed)
	}

	err = s.approveRequest(ctx, request, taskCheckTxConfirmed, taskCheckTxSentSuccess, map[string]interface{}{
		"eth_tx_hash": transaction.Hash().String(),
		"amount":      transferAmount.String(),
	})

	if err != nil {
		return errors.Wrap(err, "failed to review request second time", logan.F{"request_id": request.ID})
	}

	return nil
}

func (s *Service) callTransfer(ctx context.Context, amount *big.Int, targetAddress string) (*types.Transaction, error) {
	from := common.HexToAddress(s.transferCfg.Address)
	to := common.HexToAddress(targetAddress)
	nonce, err := s.client.PendingNonceAt(context.Background(), from)
	if err != nil {
		return nil, err
	}
	return s.contract.Transact(&bind.TransactOpts{
		Context: ctx,
		From:    from,
		Signer: func(signer types.Signer, address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			signer = types.NewEIP155Signer(s.chainID)
			signature, err := crypto.Sign(signer.Hash(tx).Bytes(), s.key)
			if err != nil {
				return nil, err
			}
			return tx.WithSignature(signer, signature)
		},
		GasLimit: s.transferCfg.GasLimit,
		GasPrice: big.NewInt(s.transferCfg.GasPrice),
		Nonce:    big.NewInt(int64(nonce)),
	}, "transfer", to, amount)
}

func prepareAmount(asset watchlist.Details, dec uint32, am uint64) *big.Int {
	trailingDigits := int64(asset.Attributes.TrailingDigits)
	decimals := int64(dec)
	var amount *big.Int
	if trailingDigits > decimals {
		amount = big.NewInt(1).
			Quo(big.NewInt(int64(am)), big.NewInt(1).Exp(big.NewInt(10), big.NewInt(trailingDigits-decimals), nil))
	} else {
		amount = big.NewInt(1).
			Mul(big.NewInt(int64(am)), big.NewInt(1).Exp(big.NewInt(10), big.NewInt(decimals-trailingDigits), nil))
	}

	return amount
}
