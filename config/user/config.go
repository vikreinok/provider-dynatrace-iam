package user

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

// Configure configures individual resources by adding custom ResourceConfigurators.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("dynatrace_user", func(r *config.Resource) {
		r.ShortGroup = "user"
	})
	p.AddResourceConfigurator("dynatrace_user_group", func(r *config.Resource) {
		r.ShortGroup = "user"
	})
}
