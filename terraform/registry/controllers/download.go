package controllers

import (
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/gruntwork-io/terragrunt/pkg/log"
	"github.com/gruntwork-io/terragrunt/terraform/registry/handlers"
	"github.com/gruntwork-io/terragrunt/terraform/registry/models"
	"github.com/gruntwork-io/terragrunt/terraform/registry/router"
	"github.com/gruntwork-io/terragrunt/terraform/registry/services"
	"github.com/labstack/echo/v4"
)

const (
	downloadPath = "/downloads"
	pluginPath   = "/plugin"
)

type DownloadController struct {
	ReverseProxy    *handlers.ReverseProxy
	ProviderService *services.ProviderService

	DownloadedPlugins []string

	basePath string
}

func (controller *DownloadController) PluginURL() *url.URL {
	proxyURL := *controller.ReverseProxy.ServerURL
	proxyURL.Path = path.Join(proxyURL.Path, controller.basePath, pluginPath)
	return &proxyURL
}

// Paths implements router.Controller.Register
func (controller *DownloadController) Register(router *router.Router) {
	router = router.Group(downloadPath)
	controller.basePath = router.Prefix()

	// Download remote file
	router.GET(pluginPath+"/:remote_host/:remote_path", controller.downloadPluginAction)
}

func (controller *DownloadController) downloadPluginAction(ctx echo.Context) error {
	var (
		remoteHost = ctx.Param("remote_host")
		remotePath = ctx.Param("remote_path")
	)
	target := fmt.Sprintf("https://%s/%s", remoteHost, remotePath)

	providerPackage := &models.ProviderPlugin{DownloadLinks: []string{target}}
	if ok := controller.ProviderService.LockPlugin(providerPackage); !ok {
		return ctx.NoContent(http.StatusConflict)
	}
	defer controller.ProviderService.UnlockPlugin(providerPackage)

	log.Debugf("Registry: start download %q", target)
	defer log.Debugf("Registry: finish download %q", target)

	if err := controller.ReverseProxy.NewRequest(ctx, target); err != nil {
		return err
	}

	controller.DownloadedPlugins = append(controller.DownloadedPlugins, target)
	return nil
}
