package main

import (
	"fmt"
	"github.com/healtronlabs/gofasta/packages/core"
	"github.com/healtronlabs/gofasta/packages/http"
	"net/http"
	"strconv"
	"strings"
)

type AppModule struct {
	core.BaseModule
}

func (m *AppModule) Configure(container *core.DIContainer) error {
	// Register controllers
	// TODO: Register UserController controller
	// TODO: Register ProductController controller
	// TODO: Register AuthController controller

	// Register providers
	if err := RegisterUserServiceProvider(container); err != nil {
		return err
	}

	if err := RegisterProductServiceProvider(container); err != nil {
		return err
	}

	if err := RegisterAuthServiceProvider(container); err != nil {
		return err
	}

	if err := RegisterEmailServiceProvider(container); err != nil {
		return err
	}

	if err := RegisterUserRepositoryProvider(container); err != nil {
		return err
	}

	if err := RegisterProductRepositoryProvider(container); err != nil {
		return err
	}

	if err := RegisterLoggerProvider(container); err != nil {
		return err
	}

	// Import other modules
	// TODO: Import DatabaseModule module
	// TODO: Import ConfigModule module

	return nil
}
