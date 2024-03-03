package runall

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/gruntwork-io/terragrunt/options"
	"github.com/gruntwork-io/terragrunt/terraform"
	"github.com/gruntwork-io/terragrunt/terraform/registry"
	"github.com/gruntwork-io/terragrunt/terraform/registry/services"
	"github.com/gruntwork-io/terragrunt/util"
	"golang.org/x/sync/errgroup"
)

func RunWithProviderCache(ctx context.Context, opts *options.TerragruntOptions) error {
	providerService := services.NewProviderService()

	// to prevent endless loop
	runTerragrunt := opts.RunTerragrunt

	opts.RunTerragrunt = func(opts *options.TerragruntOptions) error {
		writer := &util.TrapWriter{
			Writer:         opts.ErrWriter,
			TargetMsgBytes: []byte(fmt.Sprintf("%d %s", providerService.LockedPluginHTTPStatus, http.StatusText(providerService.LockedPluginHTTPStatus))),
		}
		opts.ErrWriter = writer

		for {
			err := runTerragrunt(opts)
			if err == nil || len(writer.TrappedMsgs) == 0 {
				return err
			}

			for _, msg := range writer.TrappedMsgs {
				for _, plugin := range providerService.LockedPlugins() {
					pluginPath := path.Join(plugin.RegistryName, plugin.Namespace, plugin.Name)
					if strings.Contains(msg, pluginPath) {
						providerService.WaitReleasePlugin(ctx, plugin)
					}
				}
			}
		}
	}

	if err := prepareProviderCacheEnvironment(ctx, opts); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	util.RegisterInterruptHandler(func() {
		cancel()
	})

	errGroup, ctx := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		server := registry.NewServer(opts.RegistryHostname, opts.RegistryPort, opts.RegistryToken, providerService)
		return server.Run(ctx)
	})

	if err := Run(opts); err != nil {
		return err
	}
	cancel()

	return errGroup.Wait()
}

func prepareProviderCacheEnvironment(ctx context.Context, opts *options.TerragruntOptions) error {
	opts.Env[terraform.EnvNameTFPluginCacheMayBreakDependencyLockFile] = "1"

	if opts.RegistryToken == "" {
		opts.RegistryToken = fmt.Sprintf("x-api-key:%s", uuid.New().String())
	}

	for _, registryName := range opts.RegistryNames {
		envName := fmt.Sprintf(terraform.EnvNameTFTokenFmt, strings.ReplaceAll(registryName, ".", "_"))
		opts.Env[envName] = opts.RegistryToken
	}

	if err := createLocalCLIConfigFile(opts); err != nil {
		return err
	}

	return nil
}

func createLocalCLIConfigFile(opts *options.TerragruntOptions) error {
	cfg, err := terraform.LoadConfig()
	if err != nil {
		return err
	}

	if cfg.PluginCacheDir == "" {
		cfg.PluginCacheDir = path.Join(opts.DownloadDir, "plugin-cache")

		if err := os.MkdirAll(cfg.PluginCacheDir, os.ModePerm); err != nil {
			return errors.WithStackTrace(err)
		}
	}

	for _, registryName := range opts.RegistryNames {
		host := terraform.NewConfigHost(map[string]any{
			"providers.v1": fmt.Sprintf("http://%s:%d/v1/providers/%s/", opts.RegistryHostname, opts.RegistryPort, registryName),
		})
		cfg.AddHost(registryName, host)
	}

	cliConfigFile := path.Join(opts.DownloadDir, ".terraformrc")
	if err := cfg.SaveConfig(cliConfigFile); err != nil {
		return err
	}
	opts.Env[terraform.EnvNameTFCLIConfigFile] = cliConfigFile

	return nil
}
