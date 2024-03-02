package models

type ProviderPlugin struct {
	RegistryName  string
	Namespace     string
	Name          string
	Version       string
	OS            string
	Arch          string
	DownloadLinks []string

	isLocked bool
}

func (model *ProviderPlugin) Lock() bool {
	if model.isLocked {
		return false
	}

	model.isLocked = true
	return true
}

func (model *ProviderPlugin) Unlock() bool {
	if !model.isLocked {
		return false
	}

	model.isLocked = false
	return true
}

func (model *ProviderPlugin) IsLocked() bool {
	return model.isLocked
}
