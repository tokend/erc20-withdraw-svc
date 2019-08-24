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
	transferFailed = "Transfer failed"
)

type ERC20Transfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log
}

type SentDetails struct {
	Amount    string `json:"amount"`
	EthTxHash string `json:"eth_tx_hash"`
}

type ExternalDetails struct {
	Data []json.RawMessage `json:"data"`
}

func (s *Service) confirmWithdrawSuccessful(ctx context.Context, request regources.ReviewableRequest, details *regources.CreateWithdrawRequest) error {
	fields := logan.F{
		"request_id": request.ID,
		"amount":     details.Attributes.Amount,
		"asset":      s.asset.ID,
	}
	detailsbb := []byte(request.Attributes.ExternalDetails)
	extDetails := ExternalDetails{}
	err := json.Unmarshal(detailsbb, &extDetails)
	if err != nil {
		s.log.WithFields(fields).WithError(err).Warn("Unable to unmarshal creator details")
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
	fields["eth_tx_hash"] = withdrawDetails.EthTxHash
	receipt, err := s.client.TransactionReceipt(ctx, common.HexToHash(withdrawDetails.EthTxHash))
	if err == ethereum.NotFound {
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "failed to get transaction receipt", logan.F{
			"tx_hash": withdrawDetails.EthTxHash,
		})
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		s.log.WithFields(fields).Info("Transaction unsuccessful, rejecting request...")
		return s.permanentReject(ctx, request, txFailed)
	}

	if !s.LogsSuccessful(receipt, getAddress(details.Attributes.CreatorDetails), withdrawDetails.Amount) {
		return s.permanentReject(ctx, request, transferFailed)
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

	s.log.WithFields(fields).Info("Successfully approved request")
	return nil
}

func (s *Service) ensureEnoughConfirmations(ctx context.Context, blockNumber int64) bool {
	_, err := s.client.BlockByNumber(ctx, big.NewInt(blockNumber+s.ethCfg.Confirmations))
	if err == ethereum.NotFound {
		return false
	}
	if err != nil {
		s.log.WithError(err).Error("got error trying to fetch block")
		return false
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

func (s *Service) LogsSuccessful(receipt *types.Receipt, destination string, amount string) bool {
	for _, log := range receipt.Logs {
		if log.Removed {
			return false
		}
		parsed := new(ERC20Transfer)
		if err := s.contract.UnpackLog(parsed, "Transfer", *log); err != nil {
			continue
		}
		if parsed.To.String() != destination {
			continue
		}

		if parsed.Value.String() != amount {
			continue
		}

		return true
	}

	return false
}

func getAddress(details []byte) string {
	withdrawDetails := struct {
		Address string `json:"address"`
	}{}
	json.Unmarshal(details, &withdrawDetails)

	return withdrawDetails.Address
}
