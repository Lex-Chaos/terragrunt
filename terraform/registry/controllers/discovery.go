package controllers

import (
	"net/http"

	"github.com/gruntwork-io/terragrunt/terraform/registry/router"
	"github.com/labstack/echo/v4"
)

const (
	discoveryPrefix = "/.well-known"
)

type DiscoveryEndpoints interface {
	router.Controller

	// Paths returns a list of relative paths to be passed to the router groups
	Paths() any

	// Name returns the name of the controller.
	Name() string
}

type DiscoveryController struct {
	Endpoints []DiscoveryEndpoints

	paths string
}

// Paths implements router.Controller.Register
func (controller *DiscoveryController) Register(router *router.Router) {
	router = router.Group(discoveryPrefix)
	controller.paths = router.Prefix()

	router.GET("/terraform.json", controller.terraformAction)
}

// terraformAction represents Terraform Service Discovery API endpoint.
// Docs: https://www.terraform.io/internals/remote-service-discovery
func (controller *DiscoveryController) terraformAction(ctx echo.Context) error {
	endpoints := make(map[string]any)

	for _, service := range controller.Endpoints {
		endpoints[service.Name()] = service.Paths()
	}

	return ctx.JSON(http.StatusOK, endpoints)
}
