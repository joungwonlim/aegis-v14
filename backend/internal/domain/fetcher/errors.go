package fetcher

import "errors"

// Domain errors
var (
	// Stock errors
	ErrStockNotFound      = errors.New("stock not found")
	ErrInvalidStockCode   = errors.New("invalid stock code")
	ErrInvalidMarket      = errors.New("invalid market")

	// Price errors
	ErrPriceNotFound      = errors.New("price not found")
	ErrInvalidDateRange   = errors.New("invalid date range")
	ErrNoDataAvailable    = errors.New("no data available for the given period")

	// Flow errors
	ErrFlowNotFound       = errors.New("investor flow not found")

	// Fundamentals errors
	ErrFundamentalsNotFound = errors.New("fundamentals not found")

	// MarketCap errors
	ErrMarketCapNotFound  = errors.New("market cap not found")

	// Disclosure errors
	ErrDisclosureNotFound = errors.New("disclosure not found")
	ErrDuplicateDisclosure = errors.New("duplicate disclosure")

	// Job errors
	ErrJobNotFound        = errors.New("fetch job not found")
	ErrJobAlreadyRunning  = errors.New("fetch job already running")
	ErrJobFailed          = errors.New("fetch job failed")

	// External API errors
	ErrExternalAPITimeout = errors.New("external API timeout")
	ErrExternalAPIError   = errors.New("external API error")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded")
	ErrInvalidResponse    = errors.New("invalid response from external API")
)

// IsNotFoundError checks if the error is a not found error
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrStockNotFound) ||
		errors.Is(err, ErrPriceNotFound) ||
		errors.Is(err, ErrFlowNotFound) ||
		errors.Is(err, ErrFundamentalsNotFound) ||
		errors.Is(err, ErrMarketCapNotFound) ||
		errors.Is(err, ErrDisclosureNotFound) ||
		errors.Is(err, ErrJobNotFound)
}

// IsExternalError checks if the error is an external API error
func IsExternalError(err error) bool {
	return errors.Is(err, ErrExternalAPITimeout) ||
		errors.Is(err, ErrExternalAPIError) ||
		errors.Is(err, ErrRateLimitExceeded) ||
		errors.Is(err, ErrInvalidResponse)
}
