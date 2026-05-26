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

var _ datasource.DataSource = &UserDataSource{}

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

// UserDataSource reads a single Ceph Dashboard user by username.
type UserDataSource struct {
	client *ceph.Client
}

// UserDataSourceModel describes the data source data model.
type UserDataSourceModel struct {
	Username          types.String `tfsdk:"username"`
	Roles             types.Set    `tfsdk:"roles"`
	Name              types.String `tfsdk:"name"`
	Email             types.String `tfsdk:"email"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	PwdExpirationDate types.Int64  `tfsdk:"pwd_expiration_date"`
	PwdUpdateRequired types.Bool   `tfsdk:"pwd_update_required"`
	LastUpdate        types.Int64  `tfsdk:"last_update"`
}

func (d *UserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a Ceph Dashboard user account by username.",
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Description: "Dashboard username to look up.",
				Required:    true,
			},
			"roles": schema.SetAttribute{
				Description: "Dashboard role names granted to the user.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"name":                schema.StringAttribute{Description: "Full name of the user.", Computed: true},
			"email":               schema.StringAttribute{Description: "Email address of the user.", Computed: true},
			"enabled":             schema.BoolAttribute{Description: "Whether the user is enabled.", Computed: true},
			"pwd_expiration_date": schema.Int64Attribute{Description: "Password expiration date as a Unix timestamp.", Computed: true},
			"pwd_update_required": schema.BoolAttribute{Description: "Whether the user must change their password on next login.", Computed: true},
			"last_update":         schema.Int64Attribute{Description: "Unix timestamp of the user's last update.", Computed: true},
		},
	}
}

func (d *UserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	username := data.Username.ValueString()
	user, err := d.client.GetUser(ctx, username)
	if errors.Is(err, ceph.ErrNotFound) {
		resp.Diagnostics.AddError("User not found", fmt.Sprintf("No dashboard user named %q exists.", username))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read user %q, got error: %s", username, err))
		return
	}

	roles, diags := types.SetValueFrom(ctx, types.StringType, user.Roles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Roles = roles
	data.Name = utils.NullableString(user.Name)
	data.Email = utils.NullableString(user.Email)
	data.Enabled = types.BoolValue(user.Enabled)
	data.PwdUpdateRequired = types.BoolValue(user.PwdUpdateRequired)
	if user.PwdExpirationDate != nil {
		data.PwdExpirationDate = types.Int64Value(*user.PwdExpirationDate)
	} else {
		data.PwdExpirationDate = types.Int64Null()
	}
	data.LastUpdate = types.Int64Value(user.LastUpdate)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
