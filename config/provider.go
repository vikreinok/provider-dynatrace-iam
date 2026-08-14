package config

import (
	// Note(turkenh): we are importing this to embed provider schema document
	_ "embed"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/vikreinok/provider-dynatrace-iam/config/iam"
	"github.com/vikreinok/provider-dynatrace-iam/config/mgmz"
	"github.com/vikreinok/provider-dynatrace-iam/config/policy"
	"github.com/vikreinok/provider-dynatrace-iam/config/user"
)

const (
	resourcePrefix = "dynatrace"
	modulePath     = "github.com/vikreinok/provider-dynatrace-iam"
)

//go:embed schema.json
var providerSchema string

//go:embed provider-metadata.yaml
var providerMetadata string

// GetProvider returns provider configuration
func GetProvider() *ujconfig.Provider {
	pc := ujconfig.NewProvider([]byte(providerSchema), resourcePrefix, modulePath, []byte(providerMetadata),
		ujconfig.WithRootGroup("dynatrace.crossplane.io"),
		ujconfig.WithIncludeList(ExternalNameConfigured()),
		ujconfig.WithFeaturesPackage("internal/features"),
		ujconfig.WithDefaultResourceOptions(
			ExternalNameConfigurations(),
			DefaultResourceConfigurations(),
		))

	for _, configure := range []func(provider *ujconfig.Provider){
		iam.Configure,
		mgmz.Configure,
		policy.Configure,
		user.Configure,
	} {
		configure(pc)
	}

	pc.ConfigureResources()
	return pc
}

// GetProviderNamespaced returns the namespaced provider configuration
func GetProviderNamespaced() *ujconfig.Provider {
	pc := ujconfig.NewProvider([]byte(providerSchema), resourcePrefix, modulePath, []byte(providerMetadata),
		ujconfig.WithRootGroup("dynatrace.m.crossplane.io"),
		ujconfig.WithIncludeList(ExternalNameConfigured()),
		ujconfig.WithFeaturesPackage("internal/features"),
		ujconfig.WithDefaultResourceOptions(
			ExternalNameConfigurations(),
			DefaultResourceConfigurations(),
		),
		ujconfig.WithExampleManifestConfiguration(ujconfig.ExampleManifestConfiguration{
			ManagedResourceNamespace: "crossplane-system",
		}))

	for _, configure := range []func(provider *ujconfig.Provider){
		iam.Configure,
		mgmz.Configure,
		policy.Configure,
		user.Configure,
	} {
		configure(pc)
	}

	pc.ConfigureResources()
	return pc
}

// DefaultResourceConfigurations sets dynamic initializers and Terraform conversions
func DefaultResourceConfigurations() ujconfig.ResourceOption {
	return func(r *ujconfig.Resource) {
		// Register custom lookup initializer for dynatrace_iam_group and dynatrace_iam_policy_boundary
		switch r.Name {
		case "dynatrace_iam_group", "dynatrace_iam_policy_boundary":
			r.InitializerFns = append(r.InitializerFns, func(client client.Client) managed.Initializer {
				return NewDynatraceImportInitializer(client, r.Name)
			})
		}

		// Register HCLInterpolationEscaper to automatically escape `${` -> `$${` in strings sent ToTerraform
		r.TerraformConversions = append(r.TerraformConversions, &HCLInterpolationEscaper{})
	}
}

type HCLInterpolationEscaper struct{}

func (e *HCLInterpolationEscaper) Convert(params map[string]any, r *ujconfig.Resource, mode ujconfig.Mode) (map[string]any, error) {
	if mode != ujconfig.ToTerraform {
		return params, nil
	}
	return escapeMap(params), nil
}

func escapeMap(m map[string]any) map[string]any {
	res := make(map[string]any, len(m))
	for k, v := range m {
		res[k] = escapeValue(v)
	}
	return res
}

func escapeValue(v any) any {
	switch val := v.(type) {
	case string:
		return escapeHCLString(val)
	case map[string]any:
		return escapeMap(val)
	case []any:
		res := make([]any, len(val))
		for i, item := range val {
			res[i] = escapeValue(item)
		}
		return res
	default:
		return v
	}
}

func escapeHCLString(s string) string {
	s = strings.ReplaceAll(s, "$${", "\x00")
	s = strings.ReplaceAll(s, "${", "$${")
	s = strings.ReplaceAll(s, "\x00", "$${")
	return s
}
