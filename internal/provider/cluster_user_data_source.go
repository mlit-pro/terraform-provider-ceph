/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mlit-pro/terraform-provider-ceph/ceph"
)

var _ datasource.DataSource = &ClusterUserDataSource{}

func NewClusterUserDataSource() datasource.DataSource {
	return &ClusterUserDataSource{}
}

// ClusterUserDataSource reads a single CephX cluster user by entity.
type ClusterUserDataSource struct {
	client *ceph.Client
}

// ClusterUserDataSourceModel describes the data source data model.
type ClusterUserDataSourceModel struct {
	Entity       types.String `tfsdk:"entity"`
	Capabilities types.Map    `tfsdk:"capabilities"`
	Key          types.String `tfsdk:"key"`
	Keyring      types.String `tfsdk:"keyring"`
}

func (d *ClusterUserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_user"
}

func (d *ClusterUserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a CephX cluster user (a `ceph auth` entity) by entity name.",
		Attributes: map[string]schema.Attribute{
			"entity": schema.StringAttribute{
				Description: "CephX entity name to look up, e.g. `client.admin`.",
				Required:    true,
			},
			"capabilities": schema.MapAttribute{
				Description: "Capabilities keyed by daemon type.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"key": schema.StringAttribute{
				Description: "CephX secret key for the entity.",
				Computed:    true,
				Sensitive:   true,
			},
			"keyring": schema.StringAttribute{
				Description: "Full keyring text for the entity.",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func (d *ClusterUserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*ceph.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ceph.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *ClusterUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ClusterUserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entity := data.Entity.ValueString()
	user, found, err := d.client.GetClusterUser(ctx, entity)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read cluster user %q, got error: %s", entity, err))
		return
	}
	if !found {
		resp.Diagnostics.AddError("Cluster user not found", fmt.Sprintf("No CephX user with entity %q exists.", entity))
		return
	}

	caps, diags := types.MapValueFrom(ctx, types.StringType, user.Caps)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Capabilities = caps

	keyring, key, err := d.client.GetClusterUserKeyring(ctx, entity)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to export keyring for cluster user %q, got error: %s", entity, err))
		return
	}
	data.Key = types.StringValue(key)
	data.Keyring = types.StringValue(keyring)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
