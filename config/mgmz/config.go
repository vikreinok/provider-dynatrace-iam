package mgmz

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

// Configure configures individual resources by adding custom ResourceConfigurators.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("dynatrace_mgmz_permission", func(r *config.Resource) {
		r.ShortGroup = "mgmz"
	})
}
