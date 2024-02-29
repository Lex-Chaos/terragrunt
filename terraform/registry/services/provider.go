package services

import (
	"sync"

	"github.com/gruntwork-io/terragrunt/terraform/registry/models"
)

type ProviderService struct {
	plugins models.ProviderPlugins
	mu      sync.RWMutex
}

func (service *ProviderService) AddPlugin(plugin *models.ProviderPlugin) {
	service.mu.Lock()
	defer service.mu.Unlock()

	if len(service.plugins.Find(plugin)) == 0 {
		service.plugins = append(service.plugins, plugin)
	}
}

func (service *ProviderService) LockPlugin(plugin *models.ProviderPlugin) bool {
	service.mu.Lock()
	defer service.mu.Unlock()

	if plugins := service.plugins.Find(plugin); len(plugins) == 0 || plugins.IsLocked() {
		return false
	}

	service.plugins.Find(plugin).Lock()
	return true
}

func (service *ProviderService) UnlockPlugin(plugin *models.ProviderPlugin) {
	service.mu.Lock()
	defer service.mu.Unlock()

	service.plugins.Find(plugin).Unlock()
}

func (service *ProviderService) IsPluginLocked(plugin *models.ProviderPlugin) bool {
	service.mu.RLock()
	defer service.mu.RUnlock()

	return service.plugins.Find(plugin).IsLocked()
}
