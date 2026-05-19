package v1

import (
	"fmt"

	"markpost/internal/domain/user"
	"markpost/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type testEngineConfig struct {
	validators []testutil.ValidatorRegistration
}

type testEngineOption func(*testEngineConfig)

func newTestEngine(opts ...testEngineOption) *gin.Engine {
	cfg := testEngineConfig{}
	for _, o := range opts {
		o(&cfg)
	}
	return testutil.NewTestEngine(testutil.TestEngineConfig{
		LocalesPath: "../../../../locales",
		Validators:  cfg.validators,
	})
}

func withValidators(vs ...testutil.ValidatorRegistration) testEngineOption {
	return func(cfg *testEngineConfig) {
		cfg.validators = append(cfg.validators, vs...)
	}
}

var postValidators = []testutil.ValidatorRegistration{
	{Tag: "titlesize", Fn: func(fl validator.FieldLevel) bool {
		return len(fl.Field().String()) <= 255
	}},
	{Tag: "bodysize", Fn: func(fl validator.FieldLevel) bool {
		return len(fl.Field().String()) <= 100000
	}},
}

func withTestUser(userID int) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user", &user.User{
			ID:       userID,
			Email:    fmt.Sprintf("user%d@example.com", userID),
			Username: fmt.Sprintf("user%d", userID),
		})
		c.Next()
	}
}
