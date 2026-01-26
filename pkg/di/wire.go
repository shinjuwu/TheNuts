//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"
)

func InitApp(configPath string) (*App, error) {
	wire.Build(
		InfrastructureSet,
		DatabaseSet,
		RepositorySet,
		AuthSet,
		GameSet,
		wire.Struct(new(App), "*"),
	)
	return nil, nil
}
