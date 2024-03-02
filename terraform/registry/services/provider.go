package services

import (
	"sync"

	"github.com/gruntwork-io/terragrunt/terraform/registry/models"
	"github.com/gruntwork-io/terragrunt/util"
)

type ProviderService struct {
	plugins []*models.ProviderPlugin
	mu      sync.Mutex
}

func (service *ProviderService) AddNewPlugin(new *models.ProviderPlugin) {
	service.mu.Lock()
	defer service.mu.Unlock()

	if foundPlugins := service.FindPlugins(new); len(foundPlugins) == 0 {
		service.plugins = append(service.plugins, new)
	}
}

func (service *ProviderService) LockPlugin(target *models.ProviderPlugin) bool {
	service.mu.Lock()
	defer service.mu.Unlock()

	plugins := service.FindPlugins(target)
	if len(plugins) == 0 {
		return false
	}

	for _, plugin := range plugins {
		if !plugin.Lock() {
			return false
		}
	}

	return true
}

func (service *ProviderService) UnlockPlugin(target *models.ProviderPlugin) bool {
	service.mu.Lock()
	defer service.mu.Unlock()

	for _, plugin := range service.FindPlugins(target) {
		if !plugin.Unlock() {
			return false
		}
	}

	return true
}

func (service *ProviderService) IsPluginLocked(target *models.ProviderPlugin) bool {
	service.mu.Lock()
	defer service.mu.Unlock()

	for _, plugin := range service.FindPlugins(target) {
		if plugin.IsLocked() {
			return true
		}
	}

	return false
}

func (service *ProviderService) FindPlugins(target *models.ProviderPlugin) []*models.ProviderPlugin {
	var foundPlugins []*models.ProviderPlugin

	for _, plugin := range service.plugins {
		if (plugin.RegistryName == "" || target.RegistryName == "" || plugin.RegistryName == target.RegistryName) &&
			(plugin.Namespace == "" || target.Namespace == "" || plugin.Namespace == target.Namespace) &&
			(plugin.Name == "" || target.Name == "" || plugin.Name == target.Name) &&
			(plugin.Version == "" || target.Version == "" || plugin.Version == target.Version) &&
			(plugin.OS == "" || target.OS == "" || plugin.OS == target.OS) &&
			(plugin.Arch == "" || target.Arch == "" || plugin.Arch == target.Arch) &&
			(len(plugin.DownloadLinks) == 0 || len(target.DownloadLinks) == 0 || util.ListContainsSublist(plugin.DownloadLinks, target.DownloadLinks)) {

			foundPlugins = append(foundPlugins, plugin)
		}
	}

	return foundPlugins
}
