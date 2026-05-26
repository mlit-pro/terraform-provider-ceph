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

var _ datasource.DataSource = &PoolsDataSource{}

func NewPoolsDataSource() datasource.DataSource {
	return &PoolsDataSource{}
}

// PoolsDataSource lists all Ceph pools.
type PoolsDataSource struct {
	client *ceph.Client
}

// PoolsDataSourceModel describes the data source data model.
type PoolsDataSourceModel struct {
	Pools []PoolSummaryModel `tfsdk:"pools"`
}

// PoolSummaryModel is one entry in the pools list.
type PoolSummaryModel struct {
	Name                types.String `tfsdk:"name"`
	PoolID              types.Int64  `tfsdk:"pool_id"`
	PoolType            types.String `tfsdk:"pool_type"`
	PgNum               types.Int64  `tfsdk:"pg_num"`
	Size                types.Int64  `tfsdk:"size"`
	MinSize             types.Int64  `tfsdk:"min_size"`
	CrushRule           types.String `tfsdk:"crush_rule"`
	ApplicationMetadata types.Set    `tfsdk:"application_metadata"`
	PgAutoscaleMode     types.String `tfsdk:"pg_autoscale_mode"`
	QuotaMaxBytes       types.Int64  `tfsdk:"quota_max_bytes"`
	QuotaMaxObjects     types.Int64  `tfsdk:"quota_max_objects"`
}

func (d *PoolsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pools"
}

func (d *PoolsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Ceph pools.",
		Attributes: map[string]schema.Attribute{
			"pools": schema.ListNestedAttribute{
				Description: "All pools in the cluster.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":              schema.StringAttribute{Description: "Pool name.", Computed: true},
						"pool_id":           schema.Int64Attribute{Description: "Numeric pool identifier.", Computed: true},
						"pool_type":         schema.StringAttribute{Description: "Pool type: replicated or erasure.", Computed: true},
						"pg_num":            schema.Int64Attribute{Description: "Number of placement groups.", Computed: true},
						"size":              schema.Int64Attribute{Description: "Number of replicas.", Computed: true},
						"min_size":          schema.Int64Attribute{Description: "Minimum number of replicas required for I/O.", Computed: true},
						"crush_rule":        schema.StringAttribute{Description: "CRUSH rule name.", Computed: true},
						"pg_autoscale_mode": schema.StringAttribute{Description: "Placement group autoscale mode.", Computed: true},
						"quota_max_bytes":   schema.Int64Attribute{Description: "Maximum bytes quota (0 = unlimited).", Computed: true},
						"quota_max_objects": schema.Int64Attribute{Description: "Maximum objects quota (0 = unlimited).", Computed: true},
						"application_metadata": schema.SetAttribute{
							Description: "Applications enabled on the pool.",
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *PoolsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PoolsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	pools, err := d.client.GetPools(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list pools, got error: %s", err))
		return
	}

	var data PoolsDataSourceModel
	for _, p := range pools {
		apps, diags := types.SetValueFrom(ctx, types.StringType, p.ApplicationMetadata)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Pools = append(data.Pools, PoolSummaryModel{
			Name:                types.StringValue(p.PoolName),
			PoolID:              types.Int64Value(p.Pool),
			PoolType:            types.StringValue(p.Type),
			PgNum:               types.Int64Value(p.PgNum),
			Size:                types.Int64Value(p.Size),
			MinSize:             types.Int64Value(p.MinSize),
			CrushRule:           types.StringValue(p.CrushRule),
			ApplicationMetadata: apps,
			PgAutoscaleMode:     types.StringValue(p.PgAutoscaleMode),
			QuotaMaxBytes:       types.Int64Value(p.QuotaMaxBytes),
			QuotaMaxObjects:     types.Int64Value(p.QuotaMaxObjects),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
