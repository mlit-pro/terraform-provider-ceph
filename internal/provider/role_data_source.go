/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mlit-pro/terraform-provider-ceph/ceph"
	"github.com/mlit-pro/terraform-provider-ceph/internal/utils"
)

var _ datasource.DataSource = &RoleDataSource{}

func NewRoleDataSource() datasource.DataSource {
	return &RoleDataSource{}
}

// RoleDataSource reads a single Ceph Dashboard role by name.
type RoleDataSource struct {
	client *ceph.Client
}

// RoleDataSourceModel describes the data source data model.
type RoleDataSourceModel struct {
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	ScopesPermissions types.Map    `tfsdk:"scopes_permissions"`
	System            types.Bool   `tfsdk:"system"`
}

func (d *RoleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *RoleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a Ceph Dashboard RBAC role by name.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Role name to look up.",
				Required:    true,
			},
			"description": schema.StringAttribute{Description: "Human-readable description of the role.", Computed: true},
			"scopes_permissions": schema.MapAttribute{
				Description: "Permissions keyed by security scope.",
				ElementType: types.SetType{ElemType: types.StringType},
				Computed:    true,
			},
			"system": schema.BoolAttribute{Description: "Whether the role is a built-in system role.", Computed: true},
		},
	}
}

func (d *RoleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RoleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	role, err := d.client.GetRole(ctx, name)
	if errors.Is(err, ceph.ErrNotFound) {
		resp.Diagnostics.AddError("Role not found", fmt.Sprintf("No dashboard role named %q exists.", name))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read role %q, got error: %s", name, err))
		return
	}

	scopes, diags := types.MapValueFrom(ctx, types.SetType{ElemType: types.StringType}, role.ScopesPermissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ScopesPermissions = scopes
	data.Description = utils.NullableString(role.Description)
	data.System = types.BoolValue(role.System)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
