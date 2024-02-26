package handlers

import (
	"fmt"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
)

var (
	ErrInvalidFormat = errors.New("invalid format")
	ErrInvalidValue  = errors.New("token either expired or inexistent")
)

type Authorization struct {
	ApiKey string
}

func (auth *Authorization) Auth(bearerToken string, ctx echo.Context) (bool, error) {
	if strings.HasPrefix(bearerToken, "x-api-key:") {
		var apiKey string
		if _, err := fmt.Sscanf(bearerToken, "x-api-key:%s", &apiKey); err != nil {
			return false, errors.Errorf("Authorization: %s", ErrInvalidFormat)
		}
		if apiKey != auth.ApiKey {
			return false, fmt.Errorf("Authorization: %w", ErrInvalidValue)
		}
	}

	return true, nil
}

func (auth *Authorization) KeyAuth() echo.MiddlewareFunc {
	return middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		Skipper:    middleware.DefaultSkipper,
		KeyLookup:  "header:" + echo.HeaderAuthorization,
		AuthScheme: "Bearer",
		Validator:  auth.Auth,
	})
}
