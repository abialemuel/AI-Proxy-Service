package config_test

import (
	"path/filepath"
	"runtime"
	"testing"

	config "github.com/abialemuel/AI-Proxy-Service/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	cnf config.Config
	suite.Suite
}

func (c *Suite) SetupSuite() {
	c.cnf = config.New()
}

func (c *Suite) TestConfig() {
	c.Run("Wrong init file path", func() {
		err := c.cnf.Init("wrongPath")
		assert.NotNil(c.T(), err)
	})

	c.Run("Failed config validation", func() {
		_, filename, _, _ := runtime.Caller(0)
		err := c.cnf.Init(filepath.Clean(filepath.Join(filepath.Dir(filename), "mocks/config.wrong.validate.yaml")))
		assert.NotNil(c.T(), err)
	})

	c.Run("Correct init file path", func() {
		_, filename, _, _ := runtime.Caller(0)
		err := c.cnf.Init(filepath.Clean(filepath.Join(filepath.Dir(filename), "mocks/config.yaml")))
		assert.Nil(c.T(), err)
	})

	c.Run("Get must be not nil", func() {
		assert.NotNil(c.T(), c.cnf.Get())
	})

}

func TestConfig(t *testing.T) {
	suite.Run(t, &Suite{})
}
