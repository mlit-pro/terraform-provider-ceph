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
)

var _ datasource.DataSource = &PoolDataSource{}

func NewPoolDataSource() datasource.DataSource {
	return &PoolDataSource{}
}

// PoolDataSource reads a single Ceph pool by name.
type PoolDataSource struct {
	client *ceph.Client
}

// PoolDataSourceModel describes the data source data model.
type PoolDataSourceModel struct {
	Name                types.String `tfsdk:"name"`
	PoolID              types.Int64  `tfsdk:"pool_id"`
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
}

func (d *PoolDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pool"
}

func (d *PoolDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a Ceph pool by name.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Pool name to look up.",
				Required:    true,
			},
			"pool_id": schema.Int64Attribute{
				Description: "Numeric pool identifier.",
				Computed:    true,
			},
			"pool_type": schema.StringAttribute{
				Description: "Pool type: replicated or erasure.",
				Computed:    true,
			},
			"pg_num": schema.Int64Attribute{
				Description: "Number of placement groups.",
				Computed:    true,
			},
			"size": schema.Int64Attribute{
				Description: "Number of replicas.",
				Computed:    true,
			},
			"min_size": schema.Int64Attribute{
				Description: "Minimum number of replicas required for I/O.",
				Computed:    true,
			},
			"erasure_code_profile": schema.StringAttribute{
				Description: "Erasure code profile name (erasure pools).",
				Computed:    true,
			},
			"crush_rule": schema.StringAttribute{
				Description: "CRUSH rule name.",
				Computed:    true,
			},
			"application_metadata": schema.SetAttribute{
				Description: "Applications enabled on the pool.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"pg_autoscale_mode": schema.StringAttribute{
				Description: "Placement group autoscale mode.",
				Computed:    true,
			},
			"quota_max_bytes": schema.Int64Attribute{
				Description: "Maximum bytes quota (0 = unlimited).",
				Computed:    true,
			},
			"quota_max_objects": schema.Int64Attribute{
				Description: "Maximum objects quota (0 = unlimited).",
				Computed:    true,
			},
		},
	}
}

func (d *PoolDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PoolDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PoolDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	pool, err := d.client.GetPool(ctx, name)
	if errors.Is(err, ceph.ErrNotFound) {
		resp.Diagnostics.AddError("Pool not found", fmt.Sprintf("No pool named %q exists.", name))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read pool %q, got error: %s", name, err))
		return
	}

	apps, diags := types.SetValueFrom(ctx, types.StringType, pool.ApplicationMetadata)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.PoolID = types.Int64Value(pool.Pool)
	data.PoolType = types.StringValue(pool.Type)
	data.PgNum = types.Int64Value(pool.PgNum)
	data.Size = types.Int64Value(pool.Size)
	data.MinSize = types.Int64Value(pool.MinSize)
	data.CrushRule = types.StringValue(pool.CrushRule)
	data.ApplicationMetadata = apps
	data.PgAutoscaleMode = types.StringValue(pool.PgAutoscaleMode)
	data.QuotaMaxBytes = types.Int64Value(pool.QuotaMaxBytes)
	data.QuotaMaxObjects = types.Int64Value(pool.QuotaMaxObjects)
	if pool.ErasureCodeProfile == "" {
		data.ErasureCodeProfile = types.StringNull()
	} else {
		data.ErasureCodeProfile = types.StringValue(pool.ErasureCodeProfile)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
