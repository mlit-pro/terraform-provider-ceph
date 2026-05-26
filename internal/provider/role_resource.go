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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mlit-pro/terraform-provider-ceph/ceph"
	"github.com/mlit-pro/terraform-provider-ceph/internal/utils"
)

var (
	_ resource.Resource                = &RoleResource{}
	_ resource.ResourceWithConfigure   = &RoleResource{}
	_ resource.ResourceWithImportState = &RoleResource{}
)

func NewRoleResource() resource.Resource {
	return &RoleResource{}
}

// RoleResource manages a custom Ceph Dashboard RBAC role.
type RoleResource struct {
	client *ceph.Client
}

// RoleResourceModel describes the resource data model.
type RoleResourceModel struct {
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	ScopesPermissions types.Map    `tfsdk:"scopes_permissions"`
	System            types.Bool   `tfsdk:"system"`
}

func (r *RoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *RoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a custom Ceph Dashboard RBAC role and its scope permissions.",
		MarkdownDescription: "Manages a custom Ceph Dashboard RBAC role and its scope permissions.\n\n" +
			"~> **Warning:** Built-in roles (e.g. `administrator`, `read-only`) are system roles and cannot " +
			"be managed by this resource. Removing permissions from a role the provider's account depends on " +
			"can lock the provider out.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Role name. Changing this forces a new role.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Human-readable description of the role.",
				Optional:    true,
			},
			"scopes_permissions": schema.MapAttribute{
				Description:         "Permissions keyed by security scope, e.g. {pool = [\"read\", \"update\"], monitor = [\"read\"]}. Valid permissions are create, delete, read, update.",
				MarkdownDescription: "Permissions keyed by security scope, e.g. `{ pool = [\"read\", \"update\"], monitor = [\"read\"] }`. Valid permissions are `create`, `delete`, `read`, `update`.",
				ElementType:         types.SetType{ElemType: types.StringType},
				Required:            true,
			},
			"system": schema.BoolAttribute{
				Description: "Whether the role is a built-in system role.",
				Computed:    true,
			},
		},
	}
}

func (r *RoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	spec, ok := r.specFromModel(ctx, data, &resp.Diagnostics)
	if !ok {
		return
	}

	if err := r.client.CreateRole(ctx, spec); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create role %q, got error: %s", spec.Name, err))
		return
	}

	if !r.refresh(ctx, &data, &resp.Diagnostics) {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.GetRole(ctx, data.Name.ValueString())
	if errors.Is(err, ceph.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read role %q, got error: %s", data.Name.ValueString(), err))
		return
	}

	if !applyRole(ctx, &data, role, &resp.Diagnostics) {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	spec, ok := r.specFromModel(ctx, data, &resp.Diagnostics)
	if !ok {
		return
	}

	if err := r.client.UpdateRole(ctx, spec); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update role %q, got error: %s", spec.Name, err))
		return
	}

	if !r.refresh(ctx, &data, &resp.Diagnostics) {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteRole(ctx, data.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete role %q, got error: %s", data.Name.ValueString(), err))
	}
}

func (r *RoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

// specFromModel builds a RoleSpec from the resource model.
func (r *RoleResource) specFromModel(ctx context.Context, data RoleResourceModel, diags *diag.Diagnostics) (ceph.RoleSpec, bool) {
	scopes := make(map[string][]string)
	diags.Append(data.ScopesPermissions.ElementsAs(ctx, &scopes, false)...)
	if diags.HasError() {
		return ceph.RoleSpec{}, false
	}

	return ceph.RoleSpec{
		Name:              data.Name.ValueString(),
		Description:       data.Description.ValueString(),
		ScopesPermissions: scopes,
	}, true
}

// refresh reads the role back and maps it into data.
func (r *RoleResource) refresh(ctx context.Context, data *RoleResourceModel, diags *diag.Diagnostics) bool {
	role, err := r.client.GetRole(ctx, data.Name.ValueString())
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Role %q was written but could not be read back, got error: %s", data.Name.ValueString(), err))
		return false
	}
	return applyRole(ctx, data, role, diags)
}

// applyRole maps a ceph.Role into the model.
func applyRole(ctx context.Context, data *RoleResourceModel, role ceph.Role, diags *diag.Diagnostics) bool {
	scopes, d := types.MapValueFrom(ctx, types.SetType{ElemType: types.StringType}, role.ScopesPermissions)
	diags.Append(d...)
	if diags.HasError() {
		return false
	}
	data.ScopesPermissions = scopes
	data.Description = utils.NullableString(role.Description)
	data.System = types.BoolValue(role.System)
	return true
}
