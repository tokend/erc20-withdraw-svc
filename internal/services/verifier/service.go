package verifier

import (
	"context"
	"fmt"
	"github.com/tokend/erc20-withdraw-svc/internal/horizon/page"
	"github.com/tokend/erc20-withdraw-svc/internal/horizon/query"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/distributed_lab/running"
	regources "gitlab.com/tokend/regources/generated"
	"time"
)

func (s *Service) Run(ctx context.Context) {
	s.prepare()
	withdrawPage := &regources.ReviewableRequestListResponse{}
	var err error
	running.WithBackOff(ctx, s.log, "verifier", func(ctx context.Context) error {
		if len(withdrawPage.Data) < requestPageSizeLimit {
			withdrawPage, err = s.withdrawals.List()
		} else {
			withdrawPage, err = s.withdrawals.Next()
		}
		if err != nil {
			return errors.Wrap(err, "error occurred while withdrawal request page fetching")
		}
		for _, data := range withdrawPage.Data {
			details := withdrawPage.Included.MustCreateWithdrawRequest(data.Relationships.RequestDetails.Data.GetKey())
			err := s.confirmWithdrawSuccessful(ctx, data, details)
			if err != nil {
				s.log.
					WithError(err).
					WithField("details", details).
					Warn("failed to process withdraw request")
				continue
			}
		}
		return nil
	}, 15*time.Second, 15*time.Second, time.Hour)
}


func (s *Service) prepare() {

	state := reviewableRequestStatePending
	reviewer := s.withdrawCfg.Owner.Address()
	pendingTasks := fmt.Sprintf("%d", taskCheckTxConfirmed)
	pendingTasksNotSet := fmt.Sprintf("%d", (taskTryTransfer | taskCheckTxSentSuccess))
	filters := query.CreateWithdrawRequestFilters{
		Asset: &s.asset.ID,
		ReviewableRequestFilters: query.ReviewableRequestFilters{
			State:              &state,
			Reviewer:           &reviewer,
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