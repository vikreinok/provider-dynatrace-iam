package iam

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

const groupIAM = "iam"

// Configure configures individual resources by adding custom ResourceConfigurators.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("dynatrace_iam_group", func(r *config.Resource) {
		r.ShortGroup = groupIAM
	})
	p.AddResourceConfigurator("dynatrace_iam_permission", func(r *config.Resource) {
		r.ShortGroup = groupIAM
	})
	p.AddResourceConfigurator("dynatrace_iam_policy", func(r *config.Resource) {
		r.ShortGroup = groupIAM
	})
	p.AddResourceConfigurator("dynatrace_iam_policy_bindings", func(r *config.Resource) {
		r.ShortGroup = groupIAM
	})
	p.AddResourceConfigurator("dynatrace_iam_policy_bindings_v2", func(r *config.Resource) {
		r.ShortGroup = groupIAM
		r.References["group"] = config.Reference{
			Type: "Group",
		}
		r.References["policy.id"] = config.Reference{
			Type: "Policy",
		}
		r.References["policy.boundaries"] = config.Reference{
			Type: "PolicyBoundary",
		}
	})
	p.AddResourceConfigurator("dynatrace_iam_policy_boundary", func(r *config.Resource) {
		r.ShortGroup = groupIAM
	})
	p.AddResourceConfigurator("dynatrace_iam_service_user", func(r *config.Resource) {
		r.ShortGroup = groupIAM
	})
	p.AddResourceConfigurator("dynatrace_iam_user", func(r *config.Resource) {
		r.ShortGroup = groupIAM
	})
}
