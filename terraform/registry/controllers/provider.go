package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	porviderName   = "providers.v1"
	providerPrefix = "providers"
)

// downloadLinkNames contains links that must be modified to forward terraform requests through this server.
var downloadLinkNames = []string{
	"download_url",
	"shasums_url",
	"shasums_signature_url",
}

type Downloader interface {
	PathURL() *url.URL
}

type ProviderController struct {
	Authorization   *handlers.Authorization
	ReverseProxy    *handlers.ReverseProxy
	Downloader      Downloader
	ProviderService *services.ProviderService

	path string
}

// Endpoints implements controllers.Endpointer.Endpoints
func (controller *ProviderController) Endpoints() map[string]any {
	return map[string]any{porviderName: controller.path}
}

// Paths implements router.Controller.Register
func (controller *ProviderController) Register(router *router.Router) {
	router = router.Group(providerPrefix)
	controller.path = router.Prefix()

	if controller.Authorization != nil {
		router.Use(controller.Authorization.MiddlewareFunc())
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

		providerPlugin = &models.ProviderPlugin{
			RegistryName: registryName,
			Namespace:    namespace,
			Name:         name,
		}
	)

	if controller.ProviderService.IsPluginLocked(providerPlugin) {
		return ctx.NoContent(http.StatusConflict)
	}

	target := fmt.Sprintf("https://%s/v1/providers/%s/%s/versions", registryName, namespace, name)

	return controller.ReverseProxy.NewRequest(ctx, target)
}

func (controller *ProviderController) findPackageAction(ctx echo.Context) error {
	var (
		registryName = ctx.Param("registry_name")
		namespace    = ctx.Param("namespace")
		name         = ctx.Param("name")
		version      = ctx.Param("version")
		os           = ctx.Param("os")
		arch         = ctx.Param("arch")

		target   = fmt.Sprintf("https://%s/v1/providers/%s/%s/%s/download/%s/%s", registryName, namespace, name, version, os, arch)
		proxyURL = controller.Downloader.PathURL()

		providerPlugin = &models.ProviderPlugin{
			RegistryName: registryName,
			Namespace:    namespace,
			Name:         name,
			Version:      version,
			OS:           os,
			Arch:         arch,
		}
	)

	if controller.ProviderService.IsPluginLocked(providerPlugin) {
		return ctx.NoContent(http.StatusConflict)
	}

	return controller.ReverseProxy.
		WithRewrite(func(req *httputil.ProxyRequest) {
			// Remove all encoding parameters, otherwise we will not be able to modify the body response without decoding.
			req.Out.Header.Del("Accept-Encoding")
		}).
		WithModifyResponse(func(resp *http.Response) error {
			if resp.StatusCode != http.StatusOK {
				return nil
			}

			var (
				body   map[string]json.RawMessage
				buffer = new(bytes.Buffer)
			)

			if _, err := buffer.ReadFrom(resp.Body); err != nil {
				return err
			}

			decoder := json.NewDecoder(buffer)
			if err := decoder.Decode(&body); err != nil {
				return err
			}

			for _, name := range downloadLinkNames {
				linkBytes, ok := body[name]
				if !ok || linkBytes == nil {
					continue
				}
				link := string(linkBytes)

				link, err := strconv.Unquote(link)
				if err != nil {
					return err
				}
				providerPlugin.DownloadLinks = append(providerPlugin.DownloadLinks, link)

				linkURL, err := url.Parse(link)
				if err != nil {
					return err
				}

				// Modify link to htpp://localhost/downloads/remote_host/remote_path
				linkURL.Path = path.Join(proxyURL.Path, linkURL.Host, linkURL.Path)
				linkURL.Scheme = proxyURL.Scheme
				linkURL.Host = proxyURL.Host

				link = strconv.Quote(linkURL.String())
				body[name] = []byte(link)
			}

			encoder := json.NewEncoder(buffer)
			if err := encoder.Encode(body); err != nil {
				return err
			}

			resp.Body = io.NopCloser(buffer)
			resp.ContentLength = int64(buffer.Len())

			controller.ProviderService.AddPlugin(providerPlugin)
			return nil

		}).
		NewRequest(ctx, target)
}
