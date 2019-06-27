package verifier

import (
	"context"
	"encoding/json"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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

	invalidDetails = "Invalid external details"
	invalidTXHash  = "Invalid ethereum transaction hash"
	txFailed       = "Transaction failed"
)

type SentDetails struct {
	EthTxHash string `json:"eth_tx_hash"`
}

type ExternalDetails struct {
	Data []json.RawMessage `json:"data"`
}

func (s *Service) confirmWithdrawSuccessful(ctx context.Context, request regources.ReviewableRequest, details *regources.CreateWithdrawRequest) error {
	detailsbb := []byte(request.Attributes.ExternalDetails)
	extDetails := ExternalDetails{}
	err := json.Unmarshal(detailsbb, &extDetails)
	if err != nil {
		s.log.WithField("request_id", request.ID).WithError(err).Warn("Unable to unmarshal creator details")
		return s.permanentReject(ctx, request, invalidDetails)
	}
	withdrawDetails := s.getWithdrawDetails(extDetails)

	if withdrawDetails.EthTxHash == "" {
		s.log.
			WithField("external_details", request.Attributes.ExternalDetails).
			WithError(err).
			Warn("tx hash missing")
		return s.permanentReject(ctx, request, invalidTXHash)
	}

	receipt, err := s.client.TransactionReceipt(ctx, common.HexToHash(withdrawDetails.EthTxHash))
	if err == ethereum.NotFound {
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "transfer failed")
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return s.permanentReject(ctx, request, txFailed)
	}

	if !s.ensureEnoughConfirmations(ctx, receipt.BlockNumber.Int64()) {
		return nil
	}

	err = s.approveRequest(ctx, request, 0, taskCheckTxConfirmed, map[string]interface{}{
		"eth_block_number": receipt.BlockNumber.Int64(),
	})

	if err != nil {
		return errors.Wrap(err, "failed to review request second time", logan.F{"request_id": request.ID})
	}

	return nil
}

func (s *Service) ensureEnoughConfirmations(ctx context.Context, blockNumber int64) bool {
	_, err := s.client.BlockByNumber(ctx, big.NewInt(blockNumber+20))
	if err == ethereum.NotFound {
		return false
	}
	if err != nil {
		s.log.WithError(err).Error("got error trying to fetch block")
	}

	return true
}

func (s *Service) getWithdrawDetails(ext ExternalDetails) SentDetails {
	details := SentDetails{}
	for _, raw := range ext.Data {
		_ = json.Unmarshal([]byte(raw), &details)
		if details.EthTxHash != "" {
			return details
		}
	}
	return SentDetails{}
}
