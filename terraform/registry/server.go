package registry

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gruntwork-io/go-commons/errors"
	"github.com/gruntwork-io/terragrunt/pkg/log"
	"github.com/gruntwork-io/terragrunt/terraform/registry/controllers"
	"github.com/gruntwork-io/terragrunt/terraform/registry/handlers"
	"github.com/gruntwork-io/terragrunt/terraform/registry/router"
	"github.com/gruntwork-io/terragrunt/terraform/registry/services"
	"github.com/gruntwork-io/terragrunt/util"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/sync/errgroup"
)

const (
	defaultShutdownTimeout = time.Second * 30
	defaultHostname        = "localhost"
	defaultPort            = 8080
)

type Server struct {
	handler         http.Handler
	shutdownTimeout time.Duration
	hostname        string
	port            int
}

// NewServer returns a new Server instance.
func NewServer() *Server {
	porviderService := services.NewPorviderService()
	authorization := &handlers.Authorization{
		// This is fake data, used only for testing and development.
		ApiKey: "e2b8996a-ffa5-4feb-9942-33f8804aaf52",
	}

	providerController := &controllers.ProviderController{
		Service:       porviderService,
		Authorization: authorization,
	}

	discoveryController := &controllers.DiscoveryController{
		Endpoints: []controllers.DiscoveryEndpoints{providerController},
	}

	rootRouter := router.New()
	rootRouter.Register(discoveryController)

	v1Group := rootRouter.Group("v1")
	v1Group.Register(providerController)

	rootRouter.Use(middleware.Logger())

	return &Server{
		handler:         rootRouter,
		shutdownTimeout: defaultShutdownTimeout,
		hostname:        defaultHostname,
		port:            defaultPort,
	}
}

func (server *Server) Run(ctx context.Context) error {
	log.Infof("Start Terrafrom Registry server")

	addr := net.JoinHostPort(server.hostname, strconv.Itoa(server.port))
	srv := &http.Server{Addr: addr, Handler: server.handler}

	ctx, cancel := context.WithCancel(ctx)
	util.RegisterInterruptHandler(func() {
		cancel()
	})

	httpGroup, ctx := errgroup.WithContext(ctx)
	httpGroup.Go(func() error {
		<-ctx.Done()
		log.Infof("Shutting down Terrafrom Registry server")

		ctx, cancel := context.WithTimeout(ctx, server.shutdownTimeout)
		defer cancel()

		srv.SetKeepAlivesEnabled(false)
		if err := srv.Shutdown(ctx); err != nil {
			return errors.WithStackTrace(err)
		}
		return nil
	})

	log.Infof("Terrafrom Registry server started, listening on %q", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return errors.Errorf("error starting Terrafrom Registry server: %w", err)
	}
	defer log.Infof("Terrafrom Registry server stoped")

	err := httpGroup.Wait()
	return err
}
