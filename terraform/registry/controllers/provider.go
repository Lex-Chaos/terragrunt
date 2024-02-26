package controllers

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gruntwork-io/terragrunt/pkg/log"
	"github.com/gruntwork-io/terragrunt/terraform/registry/handlers"
	"github.com/gruntwork-io/terragrunt/terraform/registry/router"
	"github.com/gruntwork-io/terragrunt/terraform/registry/services"
	"github.com/labstack/echo/v4"
)

const (
	porviderName   = "providers.v1"
	providerPrefix = "providers"
)

type ProviderController struct {
	Service       *services.PorviderService
	Authorization *handlers.Authorization

	paths string
}

// Name implements controllers.DiscoveryService.Name
func (controller *ProviderController) Name() string {
	return porviderName
}

// Paths implements controllers.DiscoveryEndpoints.Paths
func (controller *ProviderController) Paths() any {
	return controller.paths
}

// Paths implements router.Controller.Register
func (controller *ProviderController) Register(router *router.Router) {
	router = router.Group(providerPrefix)
	controller.paths = router.Prefix()

	if controller.Authorization != nil {
		router.Use(controller.Authorization.KeyAuth())
	}

	// Api should be compliant with the Terraform Registry Protocol for providers.
	// https://www.terraform.io/docs/internals/provider-registry-protocol.html#find-a-provider-package

	// Provider Versions
	router.GET("/:registry_name/:namespace/:name/versions", controller.versionsAction)

	// Find a Provider Package
	router.GET("/:registry_name/:namespace/:name/:version/download/:os/:arch", controller.findPackageAction)
}

func (controller *ProviderController) versionsAction(ctx echo.Context) error {
	var (
		registryName = ctx.Param("registry_name")
		namespace    = ctx.Param("namespace")
		name         = ctx.Param("name")
	)

	target := fmt.Sprintf("https://%s/v1/providers/%s/%s/versions", registryName, namespace, name)
	targetURL, err := url.Parse(target)
	if err != nil {
		log.Errorf("unable to parse target URL %q: %v", target, err)
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.Out.Host = registryName
			r.Out.URL = targetURL
		},
		ErrorHandler: func(resp http.ResponseWriter, req *http.Request, err error) {
			log.Errorf("remote %s unreachable, could not forward: %v", targetURL, err)
			ctx.Error(echo.NewHTTPError(http.StatusServiceUnavailable))
		},
	}
	proxy.ServeHTTP(ctx.Response(), ctx.Request())

	return nil
}

func (controller *ProviderController) findPackageAction(ctx echo.Context) error {
	var (
		registryName = ctx.Param("registry_name")
		namespace    = ctx.Param("namespace")
		name         = ctx.Param("name")
		version      = ctx.Param("version")
		os           = ctx.Param("os")
		arch         = ctx.Param("arch")
	)

	target := fmt.Sprintf("https://%s/v1/providers/%s/%s/%s/download/%s/%s", registryName, namespace, name, version, os, arch)
	targetURL, err := url.Parse(target)
	if err != nil {
		log.Errorf("unable to parse target URL %q: %v", target, err)
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.Out.Host = registryName
			r.Out.URL = targetURL
		},
		ModifyResponse: func(*http.Response) error {
			return nil
		},
		ErrorHandler: func(resp http.ResponseWriter, req *http.Request, err error) {
			log.Errorf("remote %s unreachable, could not forward: %v", targetURL, err)
			ctx.Error(echo.NewHTTPError(http.StatusServiceUnavailable))
		},
	}
	proxy.ServeHTTP(ctx.Response(), ctx.Request())

	return nil
}
