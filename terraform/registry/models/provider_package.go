package models

import (
	"github.com/gruntwork-io/terragrunt/util"
)

const (
	ProviderPluginDownloadURLName         ProviderPluginLinkName = "download_url"
	ProviderPluginSHASUMSURLName          ProviderPluginLinkName = "shasums_url"
	ProviderPluginSHASUMSSignatureURLName ProviderPluginLinkName = "shasums_signature_url"
)

var (
	// ProviderPluginDownloadLinkNames contains links that must be modified to forward terraform requests through this server.
	ProviderPluginDownloadLinkNames = []ProviderPluginLinkName{
		ProviderPluginDownloadURLName,
		ProviderPluginSHASUMSURLName,
		ProviderPluginSHASUMSSignatureURLName,
	}
)

type ProviderPluginLinkName string

type ProviderPluginLinks []string

func (urls ProviderPluginLinks) Contains(sublist ProviderPluginLinks) bool {
	return util.ListContainsSublist(urls, sublist)
}

func (urls ProviderPluginLinks) Remove(sublist ProviderPluginLinks) ProviderPluginLinks {
	return util.RemoveSublistFromList(urls, sublist)
}

type ProviderPlugins []*ProviderPlugin

func (plugins ProviderPlugins) Remove(item *ProviderPlugin) ProviderPlugins {
	return util.RemoveElementFromList(plugins, item)
}

func (plugins ProviderPlugins) Find(search *ProviderPlugin) *ProviderPlugin {
	for _, plugin := range plugins {
		if plugin.Match(search) {
			return plugin
		}
	}
	return nil
}

type ProviderPlugin struct {
	RegistryName string
	Namespace    string
	Name         string
	Version      string
	OS           string
	Arch         string
	Links        ProviderPluginLinks
}

func (plugin *ProviderPlugin) Match(target *ProviderPlugin) bool {
	if (plugin.RegistryName == "" || target.RegistryName == "" || plugin.RegistryName == target.RegistryName) &&
		(plugin.Namespace == "" || target.Namespace == "" || plugin.Namespace == target.Namespace) &&
		(plugin.Name == "" || target.Name == "" || plugin.Name == target.Name) &&
		(plugin.Version == "" || target.Version == "" || plugin.Version == target.Version) &&
		(plugin.OS == "" || target.OS == "" || plugin.OS == target.OS) &&
		(plugin.Arch == "" || target.Arch == "" || plugin.Arch == target.Arch) &&
		(len(plugin.Links) != 0 || len(target.Links) != 0 || plugin.Links.Contains(target.Links)) {

		return true
	}

	return false
}
