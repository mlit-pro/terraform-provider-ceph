/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mlit-pro/terraform-provider-ceph/ceph"
)

var _ provider.Provider = &CephProvider{}

// CephProvider defines the provider implementation.
type CephProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// CephProviderModel describes the provider data model.
type CephProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	CACert   types.String `tfsdk:"ca_cert"`
	Insecure types.Bool   `tfsdk:"insecure"`
}

func (p *CephProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ceph"
	resp.Version = p.version
}

func (p *CephProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Ceph cluster through the Ceph Manager Dashboard REST API.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Description:         "Base URL of the Ceph Manager Dashboard.",
				MarkdownDescription: "Base URL of the Ceph Manager Dashboard, e.g. `https://ceph-mgr.example.local:8443`. May also be set with the `CEPH_ENDPOINT` environment variable.",
				Optional:            true,
			},
			"username": schema.StringAttribute{
				Description:         "Dashboard username used to authenticate.",
				MarkdownDescription: "Dashboard username used to authenticate. May also be set with the `CEPH_USERNAME` environment variable.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				Description:         "Dashboard password used to authenticate.",
				MarkdownDescription: "Dashboard password used to authenticate. May also be set with the `CEPH_PASSWORD` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"ca_cert": schema.StringAttribute{
				Description:         "PEM-encoded CA certificate bundle used to verify the dashboard's TLS certificate.",
				MarkdownDescription: "PEM-encoded CA certificate bundle used to verify the dashboard's TLS certificate. When set, it replaces the system certificate pool. May also be set with the `CEPH_CA_CERT` environment variable.",
				Optional:            true,
			},
			"insecure": schema.BoolAttribute{
				Description:         "Skip TLS certificate verification.",
				MarkdownDescription: "Skip TLS certificate verification. Intended for development against self-signed certificates; mutually exclusive with `ca_cert`. May also be set with the `CEPH_INSECURE` environment variable. Defaults to `false`.",
				Optional:            true,
			},
		},
	}
}

func (p *CephProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data CephProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Values derived from other resources are not known at configure time.
	if data.Endpoint.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("endpoint"), "Unknown Ceph endpoint",
			"The endpoint cannot be a value that is unknown at plan time.")
	}
	if data.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("username"), "Unknown Ceph username",
			"The username cannot be a value that is unknown at plan time.")
	}
	if data.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("password"), "Unknown Ceph password",
			"The password cannot be a value that is unknown at plan time.")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Merge configuration with environment-variable fallbacks.
	cfg := newConfig(data)

	for _, e := range cfg.validate() {
		resp.Diagnostics.AddAttributeError(path.Root(e.attribute), e.summary, e.detail)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := ceph.New(cfg.endpoint, cfg.username, cfg.password, cfg.caCert, cfg.insecure)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Ceph client", err.Error())
		return
	}

	if err := client.Authenticate(ctx); err != nil {
		resp.Diagnostics.AddError("Unable to authenticate with Ceph", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *CephProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewClusterUserResource,
	}
}

func (p *CephProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewClusterCapacityDataSource,
		NewClusterFSIDDataSource,
		NewClusterUserDataSource,
		NewClusterUsersDataSource,
		NewHealthDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CephProvider{
			version: version,
		}
	}
}
