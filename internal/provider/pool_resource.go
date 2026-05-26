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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mlit-pro/terraform-provider-ceph/ceph"
	"github.com/mlit-pro/terraform-provider-ceph/internal/utils"
)

var (
	_ resource.Resource                   = &PoolResource{}
	_ resource.ResourceWithConfigure      = &PoolResource{}
	_ resource.ResourceWithImportState    = &PoolResource{}
	_ resource.ResourceWithValidateConfig = &PoolResource{}
)

func NewPoolResource() resource.Resource {
	return &PoolResource{}
}

// PoolResource manages a Ceph pool.
type PoolResource struct {
	client *ceph.Client
}

// PoolResourceModel describes the resource data model.
type PoolResourceModel struct {
	Name                types.String `tfsdk:"name"`
	PoolType            types.String `tfsdk:"pool_type"`
	PgNum               types.Int64  `tfsdk:"pg_num"`
	Size                types.Int64  `tfsdk:"size"`
	MinSize             types.Int64  `tfsdk:"min_size"`
	ErasureCodeProfile  types.String `tfsdk:"erasure_code_profile"`
	CrushRule           types.String `tfsdk:"crush_rule"`
	ApplicationMetadata types.Set    `tfsdk:"application_metadata"`
	PgAutoscaleMode     types.String `tfsdk:"pg_autoscale_mode"`
	QuotaMaxBytes       types.Int64  `tfsdk:"quota_max_bytes"`
	QuotaMaxObjects     types.Int64  `tfsdk:"quota_max_objects"`
	PoolID              types.Int64  `tfsdk:"pool_id"`
}

func (r *PoolResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pool"
}

func (r *PoolResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Ceph pool.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Pool name. Renaming is applied in place (the pool is not recreated).",
				Required:    true,
			},
			"pool_type": schema.StringAttribute{
				Description: "Pool type: replicated or erasure. Changing this forces a new pool.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"pg_num": schema.Int64Attribute{
				Description:         "Number of placement groups. With pg_autoscale_mode = \"on\", Ceph may change this, causing drift; set autoscale \"off\" to manage it precisely.",
				MarkdownDescription: "Number of placement groups. With `pg_autoscale_mode = \"on\"`, Ceph may change this, causing drift; set autoscale `\"off\"` to manage it precisely.",
				Required:            true,
			},
			"size": schema.Int64Attribute{
				Description: "Number of replicas (replicated pools).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"min_size": schema.Int64Attribute{
				Description: "Minimum number of replicas required for I/O (replicated pools).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"erasure_code_profile": schema.StringAttribute{
				Description: "Erasure code profile name. Required for erasure pools, forbidden for replicated pools. Changing this forces a new pool.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"crush_rule": schema.StringAttribute{
				Description: "CRUSH rule name. Defaults to the cluster's default rule for the pool type.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_metadata": schema.SetAttribute{
				Description: "Applications enabled on the pool, e.g. [\"rbd\"], [\"cephfs\"], [\"rgw\"].",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"pg_autoscale_mode": schema.StringAttribute{
				Description: "Placement group autoscale mode: on, warn, or off.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"quota_max_bytes": schema.Int64Attribute{
				Description: "Maximum bytes quota (0 = unlimited).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"quota_max_objects": schema.Int64Attribute{
				Description: "Maximum objects quota (0 = unlimited).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"pool_id": schema.Int64Attribute{
				Description: "Numeric pool identifier assigned by Ceph.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *PoolResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PoolResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data PoolResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || data.PoolType.IsUnknown() || data.PoolType.IsNull() {
		return
	}

	hasProfile := !data.ErasureCodeProfile.IsNull() && !data.ErasureCodeProfile.IsUnknown()
	switch data.PoolType.ValueString() {
	case "erasure":
		if !hasProfile {
			resp.Diagnostics.AddAttributeError(path.Root("erasure_code_profile"),
				"Missing erasure_code_profile",
				"erasure_code_profile is required when pool_type is \"erasure\".")
		}
	case "replicated":
		if hasProfile {
			resp.Diagnostics.AddAttributeError(path.Root("erasure_code_profile"),
				"Unexpected erasure_code_profile",
				"erasure_code_profile must not be set when pool_type is \"replicated\".")
		}
	}
}

func (r *PoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PoolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	spec := ceph.PoolSpec{
		Name:                data.Name.ValueString(),
		PoolType:            data.PoolType.ValueString(),
		PgNum:               utils.Int64Ptr(data.PgNum),
		Size:                utils.Int64Ptr(data.Size),
		MinSize:             utils.Int64Ptr(data.MinSize),
		ErasureCodeProfile:  utils.StringPtr(data.ErasureCodeProfile),
		CrushRule:           utils.StringPtr(data.CrushRule),
		PgAutoscaleMode:     utils.StringPtr(data.PgAutoscaleMode),
		QuotaMaxBytes:       utils.Int64Ptr(data.QuotaMaxBytes),
		QuotaMaxObjects:     utils.Int64Ptr(data.QuotaMaxObjects),
		ApplicationMetadata: utils.StringSetPtr(ctx, data.ApplicationMetadata, &resp.Diagnostics),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.CreatePool(ctx, spec); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create pool %q, got error: %s", spec.Name, err))
		return
	}

	resp.Diagnostics.Append(r.readInto(ctx, spec.Name, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PoolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pool, err := r.client.GetPool(ctx, data.Name.ValueString())
	if errors.Is(err, ceph.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read pool %q, got error: %s", data.Name.ValueString(), err))
		return
	}

	resp.Diagnostics.Append(poolToModel(ctx, pool, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PoolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	spec := ceph.PoolSpec{Name: state.Name.ValueString()}
	if !plan.Name.Equal(state.Name) {
		rename := plan.Name.ValueString()
		spec.Rename = &rename
	}
	if !plan.PgNum.Equal(state.PgNum) {
		spec.PgNum = utils.Int64Ptr(plan.PgNum)
	}
	if !plan.Size.Equal(state.Size) {
		spec.Size = utils.Int64Ptr(plan.Size)
	}
	if !plan.MinSize.Equal(state.MinSize) {
		spec.MinSize = utils.Int64Ptr(plan.MinSize)
	}
	if !plan.CrushRule.Equal(state.CrushRule) {
		spec.CrushRule = utils.StringPtr(plan.CrushRule)
	}
	if !plan.PgAutoscaleMode.Equal(state.PgAutoscaleMode) {
		spec.PgAutoscaleMode = utils.StringPtr(plan.PgAutoscaleMode)
	}
	if !plan.QuotaMaxBytes.Equal(state.QuotaMaxBytes) {
		spec.QuotaMaxBytes = utils.Int64Ptr(plan.QuotaMaxBytes)
	}
	if !plan.QuotaMaxObjects.Equal(state.QuotaMaxObjects) {
		spec.QuotaMaxObjects = utils.Int64Ptr(plan.QuotaMaxObjects)
	}
	if !plan.ApplicationMetadata.Equal(state.ApplicationMetadata) {
		spec.ApplicationMetadata = utils.StringSetPtr(ctx, plan.ApplicationMetadata, &resp.Diagnostics)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdatePool(ctx, spec); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update pool %q, got error: %s", spec.Name, err))
		return
	}

	resp.Diagnostics.Append(r.readInto(ctx, plan.Name.ValueString(), &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PoolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeletePool(ctx, data.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete pool %q, got error: %s", data.Name.ValueString(), err))
	}
}

func (r *PoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

// readInto fetches the named pool and writes its state into data.
func (r *PoolResource) readInto(ctx context.Context, name string, data *PoolResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	pool, err := r.client.GetPool(ctx, name)
	if errors.Is(err, ceph.ErrNotFound) {
		diags.AddError("Client Error", fmt.Sprintf("Pool %q was not found immediately after a successful write.", name))
		return diags
	}
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to read pool %q after write, got error: %s", name, err))
		return diags
	}
	diags.Append(poolToModel(ctx, pool, data)...)
	return diags
}

// poolToModel maps an API pool onto the resource model.
func poolToModel(ctx context.Context, p ceph.Pool, m *PoolResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	m.Name = types.StringValue(p.PoolName)
	m.PoolType = types.StringValue(p.Type)
	m.PgNum = types.Int64Value(p.PgNum)
	m.Size = types.Int64Value(p.Size)
	m.MinSize = types.Int64Value(p.MinSize)
	m.CrushRule = types.StringValue(p.CrushRule)
	m.PgAutoscaleMode = types.StringValue(p.PgAutoscaleMode)
	m.QuotaMaxBytes = types.Int64Value(p.QuotaMaxBytes)
	m.QuotaMaxObjects = types.Int64Value(p.QuotaMaxObjects)
	m.PoolID = types.Int64Value(p.Pool)

	if p.ErasureCodeProfile == "" {
		m.ErasureCodeProfile = types.StringNull()
	} else {
		m.ErasureCodeProfile = types.StringValue(p.ErasureCodeProfile)
	}

	apps, d := types.SetValueFrom(ctx, types.StringType, p.ApplicationMetadata)
	diags.Append(d...)
	m.ApplicationMetadata = apps

	return diags
}
