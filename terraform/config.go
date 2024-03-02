package terraform

import (
	"os"
	"strings"

	"github.com/genelet/determined/dethcl"
	"github.com/gruntwork-io/boilerplate/errors"
	"github.com/hashicorp/terraform/command/cliconfig"
)

func NewConfigHost(services map[string]any) *cliconfig.ConfigHost {
	return &cliconfig.ConfigHost{
		Services: services,
	}
}

type Config struct {
	*cliconfig.Config
}

func LoadConfig() (*Config, error) {
	terrafromCfg, diag := cliconfig.LoadConfig()
	if diag.HasErrors() {
		return nil, errors.WithStackTrace(diag.Err())
	}

	if terrafromCfg.Hosts == nil {
		terrafromCfg.Hosts = make(map[string]*cliconfig.ConfigHost)
	}

	return &Config{
		Config: terrafromCfg,
	}, nil
}

func (cfg *Config) AddHost(name string, host *cliconfig.ConfigHost) {
	cfg.Hosts[name] = host
}

func (cfg *Config) SaveConfig(cliConfigFile string) error {
	hclBytes, err := dethcl.Marshal(cfg)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	if err := os.WriteFile(cliConfigFile, hclBytes, os.FileMode(0644)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// IsPluginCacheUsed returns true if the terraform plugin cache dir is specified, https://developer.hashicorp.com/terraform/cli/config/config-file#provider-plugin-cache
func IsPluginCacheUsed() bool {
	if strings.TrimSpace(os.Getenv("TF_PLUGIN_CACHE_DIR")) != "" {
		return true
	}

	cfg, _ := cliconfig.LoadConfig()
	return cfg.PluginCacheDir != ""
}
