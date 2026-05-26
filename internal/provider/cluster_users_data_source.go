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

var _ datasource.DataSource = &ClusterUsersDataSource{}

func NewClusterUsersDataSource() datasource.DataSource {
	return &ClusterUsersDataSource{}
}

// ClusterUsersDataSource lists all CephX cluster users.
type ClusterUsersDataSource struct {
	client *ceph.Client
}

// ClusterUsersDataSourceModel describes the data source data model.
type ClusterUsersDataSourceModel struct {
	Users []ClusterUserSummaryModel `tfsdk:"users"`
}

// ClusterUserSummaryModel is one entry in the users list. Secret keys are not
// included; they are masked by the list endpoint.
type ClusterUserSummaryModel struct {
	Entity       types.String `tfsdk:"entity"`
	Capabilities types.Map    `tfsdk:"capabilities"`
}

func (d *ClusterUsersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_users"
}

func (d *ClusterUsersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Lists all CephX cluster users (ceph auth entities). Secret keys are not included.",
		MarkdownDescription: "Lists all CephX cluster users (`ceph auth` entities). Secret keys are not included.",
		Attributes: map[string]schema.Attribute{
			"users": schema.ListNestedAttribute{
				Description: "All CephX cluster users.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"entity": schema.StringAttribute{
							Description: "CephX entity name.",
							Computed:    true,
						},
						"capabilities": schema.MapAttribute{
							Description: "Capabilities keyed by daemon type.",
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *ClusterUsersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ClusterUsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	users, err := d.client.GetClusterUsers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list cluster users, got error: %s", err))
		return
	}

	var data ClusterUsersDataSourceModel
	for _, u := range users {
		caps, diags := types.MapValueFrom(ctx, types.StringType, u.Caps)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Users = append(data.Users, ClusterUserSummaryModel{
			Entity:       types.StringValue(u.Entity),
			Capabilities: caps,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
