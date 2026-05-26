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
	"github.com/mlit-pro/terraform-provider-ceph/internal/utils"
)

var _ datasource.DataSource = &RolesDataSource{}

func NewRolesDataSource() datasource.DataSource {
	return &RolesDataSource{}
}

// RolesDataSource lists all Ceph Dashboard roles.
type RolesDataSource struct {
	client *ceph.Client
}

// RolesDataSourceModel describes the data source data model.
type RolesDataSourceModel struct {
	Roles []RoleSummaryModel `tfsdk:"roles"`
}

// RoleSummaryModel is one entry in the roles list.
type RoleSummaryModel struct {
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	ScopesPermissions types.Map    `tfsdk:"scopes_permissions"`
	System            types.Bool   `tfsdk:"system"`
}

func (d *RolesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_roles"
}

func (d *RolesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Ceph Dashboard RBAC roles.",
		Attributes: map[string]schema.Attribute{
			"roles": schema.ListNestedAttribute{
				Description: "All dashboard roles.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":        schema.StringAttribute{Description: "Role name.", Computed: true},
						"description": schema.StringAttribute{Description: "Human-readable description of the role.", Computed: true},
						"scopes_permissions": schema.MapAttribute{
							Description: "Permissions keyed by security scope.",
							ElementType: types.SetType{ElemType: types.StringType},
							Computed:    true,
						},
						"system": schema.BoolAttribute{Description: "Whether the role is a built-in system role.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *RolesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	roles, err := d.client.GetRoles(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list roles, got error: %s", err))
		return
	}

	var data RolesDataSourceModel
	for _, role := range roles {
		scopes, diags := types.MapValueFrom(ctx, types.SetType{ElemType: types.StringType}, role.ScopesPermissions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Roles = append(data.Roles, RoleSummaryModel{
			Name:              types.StringValue(role.Name),
			Description:       utils.NullableString(role.Description),
			ScopesPermissions: scopes,
			System:            types.BoolValue(role.System),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
