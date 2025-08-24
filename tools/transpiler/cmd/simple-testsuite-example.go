package main

import (
	"fmt"
	"github.com/healtronlabs/gofasta/packages/core"
	"github.com/healtronlabs/gofasta/packages/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

type UserTests struct {
	suite.Suite
	userService *UserService `inject:""`
}

func (suite *UserTests) SetupSuite() {
	// Setup before all tests
}

func (suite *UserTests) SetupTest() {
	// Setup before each test
}

func (suite *UserTests) TearDownTest() {
	// Cleanup after each test
}

func (suite *UserTests) TearDownSuite() {
	// Cleanup after all tests
}

func (suite *UserTests) TestCreateUser() {
	// should create user
	// TODO: Implement test logic
	// Use suite.Assert() methods for assertions
	assert := suite.Assert()
	_ = assert // Remove unused variable warning
}

func TestUserTests(t *testing.T) {
	suite.Run(t, new(UserTests))
}
