package oracle

import (
	"context"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/distributed_lab/running"
	regources "gitlab.com/tokend/regources/generated"
	"time"
)

func (s *Service) Run(ctx context.Context) {
	s.prepare()

	withdrawPage := &regources.ReviewableRequestListResponse{}
	var err error

	running.WithBackOff(ctx, s.log, "sender", func(ctx context.Context) error {
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
			err := s.sendWithdraw(ctx, data, details)
			if err != nil {
				s.log.
					WithError(err).
					WithField("details", details).
					Warn("failed to process withdraw request")
			}
		}
		return nil
	}, 15*time.Second, 15*time.Second, time.Hour)
}
