package policy

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

// Configure configures individual resources by adding custom ResourceConfigurators.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("dynatrace_policy", func(r *config.Resource) {
		r.ShortGroup = "policy"
	})
	p.AddResourceConfigurator("dynatrace_policy_bindings", func(r *config.Resource) {
		r.ShortGroup = "policy"
	})
}
