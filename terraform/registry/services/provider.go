package services

import "context"

type PorviderService struct {
}

func NewPorviderService() *PorviderService {
	return &PorviderService{}
}

func (service *PorviderService) ProviderVersions(ctx context.Context, registryName, namespace, name string) {

}
