package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strconv"

	"github.com/gruntwork-io/terragrunt/terraform/registry/handlers"
	"github.com/gruntwork-io/terragrunt/terraform/registry/models"
	"github.com/gruntwork-io/terragrunt/terraform/registry/router"
	"github.com/gruntwork-io/terragrunt/terraform/registry/services"
	"github.com/labstack/echo/v4"
)

const (
	porviderName = "providers.v1"
	providerPath = "/providers"
)

type Downloader interface {
	PluginURL() *url.URL
}

type ProviderController struct {
	Authorization   *handlers.Authorization
	ReverseProxy    *handlers.ReverseProxy
	Downloader      Downloader
	ProviderService *services.ProviderService

	basePath string
}

// Endpoints implements controllers.Endpointer.Endpoints
func (controller *ProviderController) Endpoints() map[string]any {
	return map[string]any{porviderName: controller.basePath}
}

// Paths implements router.Controller.Register
func (controller *ProviderController) Register(router *router.Router) {
	router = router.Group(providerPath)
	controller.basePath = router.Prefix()

	if controller.Authorization != nil {
		router.Use(controller.Authorization.MiddlewareFunc())
	}

	// Api should be compliant with the Terraform Registry Protocol for providers.
	// https://www.terraform.io/docs/internals/provider-registry-protocol.html#find-a-provider-package

	// Provider Versions
	router.GET("/:registry_name/:namespace/:name/versions", controller.versionsAction)

	// Find a Provider Package
	router.GET("/:registry_name/:namespace/:name/:version/download/:os/:arch", controller.findPluginAction)
}

func (controller *ProviderController) versionsAction(ctx echo.Context) error {
	var (
		registryName = ctx.Param("registry_name")
		namespace    = ctx.Param("namespace")
		name         = ctx.Param("name")
	)

	providerPlugin := &models.ProviderPlugin{
		RegistryName: registryName,
		Namespace:    namespace,
		Name:         name,
	}
	if controller.ProviderService.IsPluginLocked(providerPlugin) {
		return ctx.NoContent(controller.ProviderService.LockedPluginHTTPStatus)
	}

	target := fmt.Sprintf("https://%s/v1/providers/%s/%s/versions", registryName, namespace, name)
	return controller.ReverseProxy.NewRequest(ctx, target)
}

func (controller *ProviderController) findPluginAction(ctx echo.Context) error {
	var (
		registryName = ctx.Param("registry_name")
		namespace    = ctx.Param("namespace")
		name         = ctx.Param("name")
		version      = ctx.Param("version")
		os           = ctx.Param("os")
		arch         = ctx.Param("arch")

		proxyURL = controller.Downloader.PluginURL()
	)

	providerPlugin := &models.ProviderPlugin{
		RegistryName: registryName,
		Namespace:    namespace,
		Name:         name,
		Version:      version,
		OS:           os,
		Arch:         arch,
	}
	if controller.ProviderService.LockPlugin(providerPlugin) {
		return ctx.NoContent(controller.ProviderService.LockedPluginHTTPStatus)
	}

	target := fmt.Sprintf("https://%s/v1/providers/%s/%s/%s/download/%s/%s", registryName, namespace, name, version, os, arch)
	return controller.ReverseProxy.
		WithRewrite(func(req *httputil.ProxyRequest) {
			// Remove all encoding parameters, otherwise we will not be able to modify the body response without decoding.
			req.Out.Header.Del("Accept-Encoding")
		}).
		WithModifyResponse(func(resp *http.Response) error {
			var body map[string]json.RawMessage

			return modifyJSONBody(resp, &body, func() error {
				for _, name := range models.ProviderPluginDownloadLinkNames {
					linkBytes, ok := body[string(name)]
					if !ok || linkBytes == nil {
						continue
					}
					link := string(linkBytes)

					link, err := strconv.Unquote(link)
					if err != nil {
						return err
					}
					providerPlugin.Links = append(providerPlugin.Links, link)

					linkURL, err := url.Parse(link)
					if err != nil {
						return err
					}

					// Modify link to htpp://localhost/downloads/remote_host/remote_path
					linkURL.Path = path.Join(proxyURL.Path, linkURL.Host, linkURL.Path)
					linkURL.Scheme = proxyURL.Scheme
					linkURL.Host = proxyURL.Host

					link = strconv.Quote(linkURL.String())
					body[string(name)] = []byte(link)
				}

				return nil
			})
		}).
		NewRequest(ctx, target)
}
