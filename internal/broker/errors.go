package broker

import "errors"

var (
	ErrBrokerNotSupported   = errors.New("broker not supported")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrSessionExpired       = errors.New("session expired")
	ErrInsufficientFunds    = errors.New("insufficient funds")
	ErrMarketClosed         = errors.New("market is closed")
	ErrInvalidSymbol        = errors.New("invalid symbol")
	ErrOrderRejected        = errors.New("order rejected")
	ErrInvalidOrderType     = errors.New("invalid order type")
	ErrInvalidQuantity      = errors.New("invalid quantity")
	ErrInvalidPrice         = errors.New("invalid price")
	ErrMaxPositionsReached  = errors.New("maximum positions reached")
)
