/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mlit-pro/terraform-provider-ceph/ceph"
)

var (
	_ resource.Resource                = &ClusterUserResource{}
	_ resource.ResourceWithConfigure   = &ClusterUserResource{}
	_ resource.ResourceWithImportState = &ClusterUserResource{}
)

func NewClusterUserResource() resource.Resource {
	return &ClusterUserResource{}
}

// ClusterUserResource manages a CephX cluster user (a `ceph auth` entity).
type ClusterUserResource struct {
	client *ceph.Client
}

// ClusterUserResourceModel describes the resource data model.
type ClusterUserResourceModel struct {
	Entity       types.String `tfsdk:"entity"`
	Capabilities types.Map    `tfsdk:"capabilities"`
	Key          types.String `tfsdk:"key"`
	Keyring      types.String `tfsdk:"keyring"`
}

func (r *ClusterUserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_user"
}

func (r *ClusterUserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a CephX cluster user (a `ceph auth` entity) and its capabilities.",
		Attributes: map[string]schema.Attribute{
			"entity": schema.StringAttribute{
				Description: "CephX entity name, e.g. `client.terraform`. Changing this forces a new user.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"capabilities": schema.MapAttribute{
				Description:         "Capabilities keyed by daemon type, e.g. {mon = \"allow r\", osd = \"allow rwx pool=rbd\"}. Updates overwrite all existing capabilities.",
				MarkdownDescription: "Capabilities keyed by daemon type, e.g. `{ mon = \"allow r\", osd = \"allow rwx pool=rbd\" }`. Updates overwrite all existing capabilities.",
				ElementType:         types.StringType,
				Required:            true,
			},
			"key": schema.StringAttribute{
				Description: "CephX secret key for the entity.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"keyring": schema.StringAttribute{
				Description: "Full keyring text for the entity, suitable for writing to a keyring file. Includes the capability lines, so it changes when capabilities change.",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func (r *ClusterUserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ClusterUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ClusterUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	caps := make(map[string]string)
	resp.Diagnostics.Append(data.Capabilities.ElementsAs(ctx, &caps, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entity := data.Entity.ValueString()
	if err := r.client.CreateClusterUser(ctx, entity, caps); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create cluster user %q, got error: %s", entity, err))
		return
	}

	if !r.applyKeyring(ctx, entity, &data, &resp.Diagnostics) {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ClusterUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entity := data.Entity.ValueString()
	user, found, err := r.client.GetClusterUser(ctx, entity)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read cluster user %q, got error: %s", entity, err))
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	caps, diags := types.MapValueFrom(ctx, types.StringType, user.Caps)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Capabilities = caps

	if !r.applyKeyring(ctx, entity, &data, &resp.Diagnostics) {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ClusterUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	caps := make(map[string]string)
	resp.Diagnostics.Append(data.Capabilities.ElementsAs(ctx, &caps, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entity := data.Entity.ValueString()
	if err := r.client.UpdateClusterUser(ctx, entity, caps); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update cluster user %q, got error: %s", entity, err))
		return
	}

	if !r.applyKeyring(ctx, entity, &data, &resp.Diagnostics) {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ClusterUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entity := data.Entity.ValueString()
	if err := r.client.DeleteClusterUser(ctx, entity); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete cluster user %q, got error: %s", entity, err))
	}
}

func (r *ClusterUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("entity"), req, resp)
}

// applyKeyring exports the entity and populates the key/keyring attributes.
// Returns false if an error diagnostic was added.
func (r *ClusterUserResource) applyKeyring(ctx context.Context, entity string, data *ClusterUserResourceModel, diags *diag.Diagnostics) bool {
	keyring, key, err := r.client.GetClusterUserKeyring(ctx, entity)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Cluster user %q was created/updated but its keyring could not be exported (the configured user may lack permission), got error: %s", entity, err))
		return false
	}
	data.Key = types.StringValue(key)
	data.Keyring = types.StringValue(keyring)
	return true
}
