package bootstrap

import (
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/config"
	"github.com/sirupsen/logrus"
)

func BuildLogger(cfg config.Config) (*logrus.Logger, error) {
	return buildLogger(cfg)
}
