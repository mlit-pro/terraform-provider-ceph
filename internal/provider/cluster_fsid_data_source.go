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

var _ datasource.DataSource = &ClusterFSIDDataSource{}

func NewClusterFSIDDataSource() datasource.DataSource {
	return &ClusterFSIDDataSource{}
}

// ClusterFSIDDataSource exposes the cluster's FSID.
type ClusterFSIDDataSource struct {
	client *ceph.Client
}

// ClusterFSIDDataSourceModel describes the data source data model.
type ClusterFSIDDataSourceModel struct {
	FSID types.String `tfsdk:"fsid"`
}

func (d *ClusterFSIDDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_fsid"
}

func (d *ClusterFSIDDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the Ceph cluster's FSID (unique cluster identifier).",
		Attributes: map[string]schema.Attribute{
			"fsid": schema.StringAttribute{
				Description: "Unique identifier of the Ceph cluster.",
				Computed:    true,
			},
		},
	}
}

func (d *ClusterFSIDDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ClusterFSIDDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	fsid, err := d.client.GetClusterFSID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read cluster FSID, got error: %s", err))
		return
	}

	data := ClusterFSIDDataSourceModel{
		FSID: types.StringValue(fsid),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
