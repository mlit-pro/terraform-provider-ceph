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

var _ datasource.DataSource = &MonitorsDataSource{}

func NewMonitorsDataSource() datasource.DataSource {
	return &MonitorsDataSource{}
}

// MonitorsDataSource lists the cluster's monitors and their addresses.
type MonitorsDataSource struct {
	client *ceph.Client
}

// MonitorsDataSourceModel describes the data source data model.
type MonitorsDataSourceModel struct {
	Monitors []MonitorModel `tfsdk:"monitors"`
}

// MonitorModel is one entry in the monitors list.
type MonitorModel struct {
	Name       types.String `tfsdk:"name"`
	Addr       types.String `tfsdk:"addr"`
	Priority   types.Int64  `tfsdk:"priority"`
	PublicAddr types.String `tfsdk:"public_addr"`
	Rank       types.Int64  `tfsdk:"rank"`
	Weight     types.Int64  `tfsdk:"weight"`
	InQuorum   types.Bool   `tfsdk:"in_quorum"`
}

func (d *MonitorsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitors"
}

func (d *MonitorsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the Ceph cluster's monitors and their public addresses.",
		Attributes: map[string]schema.Attribute{
			"monitors": schema.ListNestedAttribute{
				Description: "All monitors in the cluster, ordered as returned by the monmap.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":        schema.StringAttribute{Description: "Monitor name.", Computed: true},
						"addr":        schema.StringAttribute{Description: "Monitor address.", Computed: true},
						"priority":    schema.Int64Attribute{Description: "Monitor priority within the monmap.", Computed: true},
						"public_addr": schema.StringAttribute{Description: "Monitor public address.", Computed: true},
						"rank":        schema.Int64Attribute{Description: "Monitor rank within the monmap.", Computed: true},
						"weight":      schema.Int64Attribute{Description: "Monitor weight.", Computed: true},
						"in_quorum":   schema.BoolAttribute{Description: "Whether the monitor is currently part of the quorum.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *MonitorsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *MonitorsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	monitors, err := d.client.GetMonitors(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list monitors, got error: %s", err))
		return
	}

	var data MonitorsDataSourceModel
	for _, m := range monitors {
		data.Monitors = append(data.Monitors, MonitorModel{
			Name:       types.StringValue(m.Name),
			Addr:       types.StringValue(m.Addr),
			Priority:   types.Int64Value(m.Priority),
			PublicAddr: types.StringValue(m.PublicAddr),
			Rank:       types.Int64Value(m.Rank),
			Weight:     types.Int64Value(m.Weight),
			InQuorum:   types.BoolValue(m.InQuorum),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
