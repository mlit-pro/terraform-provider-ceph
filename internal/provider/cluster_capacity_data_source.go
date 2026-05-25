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

var _ datasource.DataSource = &ClusterCapacityDataSource{}

func NewClusterCapacityDataSource() datasource.DataSource {
	return &ClusterCapacityDataSource{}
}

// ClusterCapacityDataSource exposes the cluster's raw capacity figures.
type ClusterCapacityDataSource struct {
	client *ceph.Client
}

// ClusterCapacityDataSourceModel describes the data source data model.
type ClusterCapacityDataSourceModel struct {
	TotalAvailBytes   types.Int64 `tfsdk:"total_avail_bytes"`
	TotalBytes        types.Int64 `tfsdk:"total_bytes"`
	TotalUsedRawBytes types.Int64 `tfsdk:"total_used_raw_bytes"`
}

func (d *ClusterCapacityDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_capacity"
}

func (d *ClusterCapacityDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the Ceph cluster's raw capacity figures.",
		Attributes: map[string]schema.Attribute{
			"total_avail_bytes": schema.Int64Attribute{
				Description: "Total available (unused) raw capacity in bytes.",
				Computed:    true,
			},
			"total_bytes": schema.Int64Attribute{
				Description: "Total raw capacity in bytes.",
				Computed:    true,
			},
			"total_used_raw_bytes": schema.Int64Attribute{
				Description: "Total used raw capacity in bytes.",
				Computed:    true,
			},
		},
	}
}

func (d *ClusterCapacityDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ClusterCapacityDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	capacity, err := d.client.GetClusterCapacity(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read cluster capacity, got error: %s", err))
		return
	}

	data := ClusterCapacityDataSourceModel{
		TotalAvailBytes:   types.Int64Value(capacity.TotalAvailBytes),
		TotalBytes:        types.Int64Value(capacity.TotalBytes),
		TotalUsedRawBytes: types.Int64Value(capacity.TotalUsedRawBytes),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
