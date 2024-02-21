package registry

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gruntwork-io/go-commons/errors"
	"github.com/gruntwork-io/terragrunt/pkg/log"
	"github.com/gruntwork-io/terragrunt/terraform/registry/controllers/provider"
	"github.com/gruntwork-io/terragrunt/terraform/registry/router"
	"github.com/gruntwork-io/terragrunt/util"
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
	providerController := provider.NewController()

	router := router.New()

	v1Group := router.Group("v1")
	v1Group.Register(providerController)

	return &Server{
		handler:         router,
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

	log.Infof("Terrafrom Registry server is listening on %q", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return errors.Errorf("error starting Terrafrom Registry server: %w", err)
	}
	defer log.Infof("Terrafrom Registry server stoped")

	err := httpGroup.Wait()
	return err
}
