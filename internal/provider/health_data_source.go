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

var _ datasource.DataSource = &HealthDataSource{}

func NewHealthDataSource() datasource.DataSource {
	return &HealthDataSource{}
}

// HealthDataSource exposes the cluster's overall health status.
type HealthDataSource struct {
	client *ceph.Client
}

// HealthDataSourceModel describes the data source data model.
type HealthDataSourceModel struct {
	HealthStatus types.String `tfsdk:"health_status"`
}

func (d *HealthDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_health"
}

func (d *HealthDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the Ceph cluster's overall health status.",
		Attributes: map[string]schema.Attribute{
			"health_status": schema.StringAttribute{
				Description:         "Overall cluster health status: HEALTH_OK, HEALTH_WARN, or HEALTH_ERR.",
				MarkdownDescription: "Overall cluster health status: `HEALTH_OK`, `HEALTH_WARN`, or `HEALTH_ERR`.",
				Computed:            true,
			},
		},
	}
}

func (d *HealthDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *HealthDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	report, err := d.client.GetHealthMinimal(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read cluster health, got error: %s", err))
		return
	}

	data := HealthDataSourceModel{
		HealthStatus: types.StringValue(report.Health.Status),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
