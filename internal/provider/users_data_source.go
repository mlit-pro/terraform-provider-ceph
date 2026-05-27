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

var _ datasource.DataSource = &UsersDataSource{}

func NewUsersDataSource() datasource.DataSource {
	return &UsersDataSource{}
}

// UsersDataSource lists all Ceph Dashboard users.
type UsersDataSource struct {
	client *ceph.Client
}

// UsersDataSourceModel describes the data source data model.
type UsersDataSourceModel struct {
	Users []UserSummaryModel `tfsdk:"users"`
}

// UserSummaryModel is one entry in the users list.
type UserSummaryModel struct {
	Username          types.String `tfsdk:"username"`
	Roles             types.Set    `tfsdk:"roles"`
	Name              types.String `tfsdk:"name"`
	Email             types.String `tfsdk:"email"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	PwdExpirationDate types.Int64  `tfsdk:"pwd_expiration_date"`
	PwdUpdateRequired types.Bool   `tfsdk:"pwd_update_required"`
	LastUpdate        types.Int64  `tfsdk:"last_update"`
}

func (d *UsersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

func (d *UsersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Ceph Dashboard user accounts.",
		Attributes: map[string]schema.Attribute{
			"users": schema.ListNestedAttribute{
				Description: "All dashboard users.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"username": schema.StringAttribute{Description: "Dashboard username.", Computed: true},
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
				},
			},
		},
	}
}

func (d *UsersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *UsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	users, err := d.client.GetUsers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list users, got error: %s", err))
		return
	}

	var data UsersDataSourceModel
	for _, u := range users {
		roles, diags := types.SetValueFrom(ctx, types.StringType, u.Roles)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		entry := UserSummaryModel{
			Username:          types.StringValue(u.Username),
			Roles:             roles,
			Name:              utils.NullableString(u.Name),
			Email:             utils.NullableString(u.Email),
			Enabled:           types.BoolValue(u.Enabled),
			PwdUpdateRequired: types.BoolValue(u.PwdUpdateRequired),
			LastUpdate:        types.Int64Value(u.LastUpdate),
		}
		if u.PwdExpirationDate != nil {
			entry.PwdExpirationDate = types.Int64Value(*u.PwdExpirationDate)
		} else {
			entry.PwdExpirationDate = types.Int64Null()
		}
		data.Users = append(data.Users, entry)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
