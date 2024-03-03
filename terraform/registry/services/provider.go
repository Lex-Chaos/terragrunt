package services

import (
	"context"
	"net/http"
	"sync"

	"github.com/gruntwork-io/terragrunt/terraform/registry/models"
)

const (
	defaultLockedPluginHTTPStatus = http.StatusConflict
)

type ProviderService struct {
	LockedPluginHTTPStatus int

	lockedPlugins    models.ProviderPlugins
	releasedPluginCh chan *models.ProviderPlugin
	pluginMu         sync.RWMutex
}

func NewProviderService() *ProviderService {
	return &ProviderService{
		LockedPluginHTTPStatus: defaultLockedPluginHTTPStatus,
		releasedPluginCh:       make(chan *models.ProviderPlugin),
	}
}

func (service *ProviderService) LockedPlugins() models.ProviderPlugins {
	service.pluginMu.RLock()
	defer service.pluginMu.RUnlock()

	return service.lockedPlugins
}

func (service *ProviderService) IsPluginLocked(target *models.ProviderPlugin) bool {
	service.pluginMu.RLock()
	defer service.pluginMu.RUnlock()

	if plugin := service.lockedPlugins.Find(target); plugin != nil {
		return true
	}
	return false
}

func (service *ProviderService) LockPlugin(target *models.ProviderPlugin) bool {
	service.pluginMu.Lock()
	defer service.pluginMu.Unlock()

	if plugin := service.lockedPlugins.Find(target); plugin != nil {
		return false
	}

	service.lockedPlugins = append(service.lockedPlugins, target)
	return true
}

func (service *ProviderService) UnlockPlugin(target *models.ProviderPlugin) bool {
	service.pluginMu.Lock()
	defer service.pluginMu.Unlock()

	if plugin := service.lockedPlugins.Find(target); plugin != nil {
		plugin.Links = plugin.Links.Remove(target.Links)

		if len(plugin.Links) == 0 {
			service.lockedPlugins = service.lockedPlugins.Remove(plugin)

			for {
				select {
				case service.releasedPluginCh <- plugin:
				default:
					return true
				}
			}
		}
	}

	return false
}

func (service *ProviderService) WaitReleasePlugin(ctx context.Context, target *models.ProviderPlugin) {
	for {
		select {
		case releasedPlugin := <-service.releasedPluginCh:
			if releasedPlugin.Match(target) {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
