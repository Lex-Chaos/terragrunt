package models

import "github.com/gruntwork-io/gruntwork-cli/collections"

type Links []string

func (links Links) Contains(targetLinks Links) bool {
	for _, targetLink := range targetLinks {
		if collections.ListContainsElement(links, targetLink) {
			return true
		}
	}
	return false
}

type ProviderPlugin struct {
	RegistryName  string
	Namespace     string
	Name          string
	Version       string
	OS            string
	Arch          string
	DownloadLinks Links

	locked bool
}

type ProviderPlugins []*ProviderPlugin

func (models ProviderPlugins) Lock() {
	for _, model := range models {
		model.locked = true
	}
}

func (models ProviderPlugins) Unlock() {
	for _, model := range models {
		model.locked = false
	}
}

func (models ProviderPlugins) IsLocked() bool {
	for _, model := range models {
		if model.locked {
			return true
		}
	}
	return false
}

func (models ProviderPlugins) Find(target *ProviderPlugin) ProviderPlugins {
	var foundModels ProviderPlugins

	for _, model := range models {
		if (model.RegistryName == "" || target.RegistryName == "" || model.RegistryName == target.RegistryName) &&
			(model.Namespace == "" || target.Namespace == "" || model.Namespace == target.Namespace) &&
			(model.Name == "" || target.Name == "" || model.Name == target.Name) &&
			(model.Version == "" || target.Version == "" || model.Version == target.Version) &&
			(model.OS == "" || target.OS == "" || model.OS == target.OS) &&
			(model.Arch == "" || target.Arch == "" || model.Arch == target.Arch) &&
			(len(model.DownloadLinks) == 0 || len(target.DownloadLinks) == 0 || model.DownloadLinks.Contains(target.DownloadLinks)) {

			foundModels = append(foundModels, model)
		}
	}

	return foundModels
}
