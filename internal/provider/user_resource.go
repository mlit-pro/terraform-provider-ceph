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

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mlit-pro/terraform-provider-ceph/ceph"
	"github.com/mlit-pro/terraform-provider-ceph/internal/utils"
)

var (
	_ resource.Resource                = &UserResource{}
	_ resource.ResourceWithConfigure   = &UserResource{}
	_ resource.ResourceWithImportState = &UserResource{}
)

func NewUserResource() resource.Resource {
	return &UserResource{}
}

// UserResource manages a Ceph Dashboard RBAC user account.
type UserResource struct {
	client *ceph.Client
}

// UserResourceModel describes the resource data model.
type UserResourceModel struct {
	Username          types.String `tfsdk:"username"`
	Password          types.String `tfsdk:"password"`
	Roles             types.Set    `tfsdk:"roles"`
	Name              types.String `tfsdk:"name"`
	Email             types.String `tfsdk:"email"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	PwdExpirationDate types.Int64  `tfsdk:"pwd_expiration_date"`
	PwdUpdateRequired types.Bool   `tfsdk:"pwd_update_required"`
	LastUpdate        types.Int64  `tfsdk:"last_update"`
}

func (r *UserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Ceph Dashboard user account (RBAC) and its roles.",
		MarkdownDescription: "Manages a Ceph Dashboard user account (RBAC) and its roles.\n\n" +
			"~> **Warning:** Managing the same dashboard account the provider authenticates as can lock " +
			"the provider out (e.g. by disabling it, removing its administrator role, or deleting it). " +
			"Manage a separate account for Terraform.",
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Description: "Dashboard username. Changing this forces a new user.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				Description:         "User password. Write-only: the API never returns it, so changes made outside Terraform are not detected.",
				MarkdownDescription: "User password. Write-only: the API never returns it, so changes made outside Terraform are not detected.",
				Required:            true,
				Sensitive:           true,
			},
			"roles": schema.SetAttribute{
				Description: "Dashboard role names granted to the user, e.g. [\"administrator\"].",
				ElementType: types.StringType,
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Full name of the user.",
				Optional:    true,
			},
			"email": schema.StringAttribute{
				Description: "Email address of the user.",
				Optional:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the user is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"pwd_expiration_date": schema.Int64Attribute{
				Description: "Password expiration date as a Unix timestamp. Unset means no expiration.",
				Optional:    true,
			},
			"pwd_update_required": schema.BoolAttribute{
				Description: "Whether the user must change their password on next login. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"last_update": schema.Int64Attribute{
				Description: "Unix timestamp of the user's last update.",
				Computed:    true,
			},
		},
	}
}

func (r *UserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*ceph.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ceph.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	spec, ok := r.specFromModel(ctx, data, &resp.Diagnostics)
	if !ok {
		return
	}

	if err := r.client.CreateUser(ctx, spec); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create user %q, got error: %s", spec.Username, err))
		return
	}

	if !r.refresh(ctx, &data, &resp.Diagnostics) {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.GetUser(ctx, data.Username.ValueString())
	if errors.Is(err, ceph.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read user %q, got error: %s", data.Username.ValueString(), err))
		return
	}

	// Password is never returned by the API; leave the state value untouched.
	if !applyUser(ctx, &data, user, &resp.Diagnostics) {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	spec, ok := r.specFromModel(ctx, data, &resp.Diagnostics)
	if !ok {
		return
	}

	if err := r.client.UpdateUser(ctx, spec); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update user %q, got error: %s", spec.Username, err))
		return
	}

	if !r.refresh(ctx, &data, &resp.Diagnostics) {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteUser(ctx, data.Username.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete user %q, got error: %s", data.Username.ValueString(), err))
	}
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("username"), req, resp)
}

// specFromModel builds a UserSpec from the resource model.
func (r *UserResource) specFromModel(ctx context.Context, data UserResourceModel, diags *diag.Diagnostics) (ceph.UserSpec, bool) {
	var roles []string
	diags.Append(data.Roles.ElementsAs(ctx, &roles, false)...)
	if diags.HasError() {
		return ceph.UserSpec{}, false
	}

	spec := ceph.UserSpec{
		Username:          data.Username.ValueString(),
		Password:          data.Password.ValueString(),
		Roles:             roles,
		Name:              data.Name.ValueString(),
		Email:             data.Email.ValueString(),
		Enabled:           data.Enabled.ValueBool(),
		PwdUpdateRequired: data.PwdUpdateRequired.ValueBool(),
	}
	if !data.PwdExpirationDate.IsNull() {
		v := data.PwdExpirationDate.ValueInt64()
		spec.PwdExpirationDate = &v
	}
	return spec, true
}

// refresh reads the user back and maps it into data, preserving the password.
func (r *UserResource) refresh(ctx context.Context, data *UserResourceModel, diags *diag.Diagnostics) bool {
	user, err := r.client.GetUser(ctx, data.Username.ValueString())
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("User %q was written but could not be read back, got error: %s", data.Username.ValueString(), err))
		return false
	}
	return applyUser(ctx, data, user, diags)
}

// applyUser maps a ceph.User into the model, leaving Password untouched.
func applyUser(ctx context.Context, data *UserResourceModel, user ceph.User, diags *diag.Diagnostics) bool {
	roles, d := types.SetValueFrom(ctx, types.StringType, user.Roles)
	diags.Append(d...)
	if diags.HasError() {
		return false
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
	return true
}
