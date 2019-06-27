package query

import (
	"fmt"
	"github.com/tokend/erc20-withdraw-svc/internal/horizon/page"
	"net/url"
)

type CreateWithdrawRequestFilters struct {
	ReviewableRequestFilters
	Balance *string
	Asset   *string
}

type CreateWithdrawRequestIncludes struct {
	ReviewableRequestIncludes
	Balance bool
	Asset   bool
}

type CreateWithdrawRequestParams struct {
	Includes   CreateWithdrawRequestIncludes
	Filters    CreateWithdrawRequestFilters
	PageParams page.Params
}

func (p CreateWithdrawRequestParams) Prepare() url.Values {
	result := url.Values{}
	p.Filters.prepare(&result)
	p.PageParams.Prepare(&result)
	p.Includes.prepare(&result)
	return result
}

func (p CreateWithdrawRequestFilters) prepare(result *url.Values) {
	p.ReviewableRequestFilters.prepare(result)
	if p.Balance != nil {
		result.Add("filter[request_details.balance]", fmt.Sprintf("%s", *p.Balance))
	}
	if p.Asset != nil {
		result.Add("filter[request_details.asset]", fmt.Sprintf("%s", *p.Asset))
	}
}

func (p CreateWithdrawRequestIncludes) prepare(result *url.Values) {
	p.ReviewableRequestIncludes.prepare(result)

	if p.Asset {
		result.Add("include", "request_details.asset")
	}

	if p.Balance {
		result.Add("include", "request_details.balance")
	}
}

func (p CreateWithdrawRequestIncludes) Prepare() url.Values {
	result := url.Values{}
	p.prepare(&result)
	return result
}

func CreateWithdrawRequestByID(code string) string {
	return fmt.Sprintf("/v3/create_withdraw_requests/%s", code)
}

func CreateWithdrawRequestList() string {
	return "/v3/create_withdraw_requests"
}
