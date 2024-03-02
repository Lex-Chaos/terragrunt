package runall

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/gruntwork-io/terragrunt/configstack"
	"github.com/gruntwork-io/terragrunt/options"
	"github.com/gruntwork-io/terragrunt/shell"
	"github.com/gruntwork-io/terragrunt/terraform"
	"github.com/gruntwork-io/terragrunt/terraform/registry"
	"github.com/gruntwork-io/terragrunt/util"
	"golang.org/x/sync/errgroup"
)

// Known terraform commands that are explicitly not supported in run-all due to the nature of the command. This is
// tracked as a map that maps the terraform command to the reasoning behind disallowing the command in run-all.
var runAllDisabledCommands = map[string]string{
	terraform.CommandNameImport:      "terraform import should only be run against a single state representation to avoid injecting the wrong object in the wrong state representation.",
	terraform.CommandNameTaint:       "terraform taint should only be run against a single state representation to avoid using the wrong state address.",
	terraform.CommandNameUntaint:     "terraform untaint should only be run against a single state representation to avoid using the wrong state address.",
	terraform.CommandNameConsole:     "terraform console requires stdin, which is shared across all instances of run-all when multiple modules run concurrently.",
	terraform.CommandNameForceUnlock: "lock IDs are unique per state representation and thus should not be run with run-all.",

	// MAINTAINER'S NOTE: There are a few other commands that might not make sense, but we deliberately allow it for
	// certain use cases that are documented here:
	// - state          : Supporting `state` with run-all could be useful for a mass pull and push operation, which can
	//                    be done en masse with the use of relative pathing.
	// - login / logout : Supporting `login` with run-all could be useful when used in conjunction with tfenv and
	//                    multi-terraform version setups, where multiple terraform versions need to be configured.
	// - version        : Supporting `version` with run-all could be useful for sanity checking a multi-version setup.
}

func createLocalCLIConfigFileForPluginCache(opts *options.TerragruntOptions) error {
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

func RunWithProviderCache(ctx context.Context, opts *options.TerragruntOptions) error {
	if opts.RegistryToken == "" {
		opts.RegistryToken = fmt.Sprintf("x-api-key:%s", uuid.New().String())
	}

	for _, registryName := range opts.RegistryNames {
		envName := fmt.Sprintf(terraform.EnvNameTFTokenFmt, strings.ReplaceAll(registryName, ".", "_"))
		opts.Env[envName] = opts.RegistryToken
	}

	if err := createLocalCLIConfigFileForPluginCache(opts); err != nil {
		return err
	}

	registryServer := registry.NewServer(opts.RegistryHostname, opts.RegistryPort)
	registryServer.Token = opts.RegistryToken

	ctx, cancel := context.WithCancel(ctx)
	util.RegisterInterruptHandler(func() {
		cancel()
	})

	errGroup, ctx := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		return registryServer.Run(ctx)
	})

	for {
		downloadedPlugins := registryServer.DownloadedPlugins

		if err := Run(opts); err == nil {
			break
		} else if len(registryServer.DownloadedPlugins) == len(downloadedPlugins) {
			return err
		}

		opts.Env[terraform.EnvNameTFPluginCacheMayBreakDependencyLockFile] = "1"
	}
	cancel()

	return errGroup.Wait()
}

func Run(opts *options.TerragruntOptions) error {
	if opts.TerraformCommand == "" {
		return errors.WithStackTrace(MissingCommand{})
	}

	reason, isDisabled := runAllDisabledCommands[opts.TerraformCommand]
	if isDisabled {
		return RunAllDisabledErr{
			command: opts.TerraformCommand,
			reason:  reason,
		}
	}

	stack, err := configstack.FindStackInSubfolders(opts, nil)
	if err != nil {
		return err
	}

	return RunAllOnStack(opts, stack)
}

func RunAllOnStack(opts *options.TerragruntOptions, stack *configstack.Stack) error {
	opts.Logger.Debugf("%s", stack.String())
	if err := stack.LogModuleDeployOrder(opts.Logger, opts.TerraformCommand); err != nil {
		return err
	}

	var prompt string
	switch opts.TerraformCommand {
	case terraform.CommandNameApply:
		prompt = "Are you sure you want to run 'terragrunt apply' in each folder of the stack described above?"
	case terraform.CommandNameDestroy:
		prompt = "WARNING: Are you sure you want to run `terragrunt destroy` in each folder of the stack described above? There is no undo!"
	case terraform.CommandNameState:
		prompt = "Are you sure you want to manipulate the state with `terragrunt state` in each folder of the stack described above? Note that absolute paths are shared, while relative paths will be relative to each working directory."
	}
	if prompt != "" {
		shouldRunAll, err := shell.PromptUserForYesNo(prompt, opts)
		if err != nil {
			return err
		}
		if !shouldRunAll {
			return nil
		}
	}

	return stack.Run(opts)
}
