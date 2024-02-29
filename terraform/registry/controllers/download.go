package controllers

import (
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/gruntwork-io/terragrunt/terraform/registry/handlers"
	"github.com/gruntwork-io/terragrunt/terraform/registry/models"
	"github.com/gruntwork-io/terragrunt/terraform/registry/router"
	"github.com/gruntwork-io/terragrunt/terraform/registry/services"
	"github.com/labstack/echo/v4"
)

const (
	downloadPrefix = "downloads"
)

type DownloadController struct {
	Authorization   *handlers.Authorization
	ReverseProxy    *handlers.ReverseProxy
	ProviderService *services.ProviderService

	path string
}

func (controller *DownloadController) PathURL() *url.URL {
	proxyURL := *controller.ReverseProxy.ServerURL
	proxyURL.Path = path.Join(proxyURL.Path, controller.path)
	return &proxyURL
}

// Paths implements router.Controller.Register
func (controller *DownloadController) Register(router *router.Router) {
	router = router.Group(downloadPrefix)
	controller.path = router.Prefix()

	if controller.Authorization != nil {
		router.Use(controller.Authorization.MiddlewareFunc())
	}

	// Download remote file
	router.GET("/:remote_host/:remote_path", controller.downloadAction)
}

func (controller *DownloadController) downloadAction(ctx echo.Context) error {
	var (
		remoteHost = ctx.Param("remote_host")
		remotePath = ctx.Param("remote_path")

		target          = fmt.Sprintf("https://%s/%s", remoteHost, remotePath)
		providerPackage = &models.ProviderPlugin{DownloadLinks: []string{target}}
	)

	if ok := controller.ProviderService.LockPlugin(providerPackage); !ok {
		return ctx.NoContent(http.StatusConflict)
	}
	defer controller.ProviderService.UnlockPlugin(providerPackage)

	return controller.ReverseProxy.NewRequest(ctx, target)
}
