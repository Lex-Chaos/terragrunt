package registry

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gruntwork-io/go-commons/errors"
	"github.com/gruntwork-io/terragrunt/pkg/log"
	"github.com/gruntwork-io/terragrunt/terraform/registry/controllers"
	"github.com/gruntwork-io/terragrunt/terraform/registry/handlers"
	"github.com/gruntwork-io/terragrunt/terraform/registry/router"
	"github.com/gruntwork-io/terragrunt/terraform/registry/services"
	"golang.org/x/sync/errgroup"
)

const (
	defaultShutdownTimeout = time.Second * 30
)

type Server struct {
	*controllers.DownloadController
	*controllers.ProviderController
	*handlers.Authorization

	handler         http.Handler
	shutdownTimeout time.Duration
	hostname        string
	port            int
}

// NewServer returns a new Server instance.
func NewServer(hostname string, port int) *Server {
	providerService := &services.ProviderService{}
	authorization := &handlers.Authorization{}

	reverseProxy := &handlers.ReverseProxy{
		ServerURL: &url.URL{
			Scheme: "http",
			Host:   net.JoinHostPort(hostname, strconv.Itoa(port)),
		},
	}

	downloadController := &controllers.DownloadController{
		ReverseProxy:    reverseProxy,
		ProviderService: providerService,
	}

	providerController := &controllers.ProviderController{
		Authorization:   authorization,
		ReverseProxy:    reverseProxy,
		Downloader:      downloadController,
		ProviderService: providerService,
	}

	discoveryController := &controllers.DiscoveryController{
		Endpointers: []controllers.Endpointer{providerController},
	}

	rootRouter := router.New()
	rootRouter.Register(discoveryController, downloadController)

	v1Group := rootRouter.Group("v1")
	v1Group.Register(providerController)

	return &Server{
		DownloadController: downloadController,
		ProviderController: providerController,
		Authorization:      authorization,

		handler:         rootRouter,
		shutdownTimeout: defaultShutdownTimeout,
		hostname:        hostname,
		port:            port,
	}
}

func (server *Server) Run(ctx context.Context) error {
	log.Infof("Start Private Registry")

	addr := net.JoinHostPort(server.hostname, strconv.Itoa(server.port))
	srv := &http.Server{Addr: addr, Handler: server.handler}

	errGroup, ctx := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		<-ctx.Done()
		log.Infof("Shutting down Private Registry")

		ctx, cancel := context.WithTimeout(ctx, server.shutdownTimeout)
		defer cancel()

		srv.SetKeepAlivesEnabled(false)
		if err := srv.Shutdown(ctx); err != nil {
			return errors.WithStackTrace(err)
		}
		return nil
	})

	log.Infof("Private Registry started, listening on %q", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return errors.Errorf("error starting Terrafrom Registry server: %w", err)
	}
	defer log.Infof("Private Registry stopped")

	err := errGroup.Wait()
	return err
}
