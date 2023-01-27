package internal

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ provider.Provider = &taskRunnerProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New() provider.Provider {
	return &taskRunnerProvider{}
}

// taskRunnerProvider is the provider implementation.
type taskRunnerProvider struct{}

// taskRunnerProviderModel maps provider schema data to a Go type.
type taskRunnerProviderModel struct {
}

// Metadata returns the provider type name.
func (p *taskRunnerProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "taskrunner-aws-ecs"
}

// Schema defines the provider-level schema for configuration data.
func (p *taskRunnerProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with ECS tasks",
		Attributes:  map[string]schema.Attribute{},
	}
}

// Configure prepares a ECS API client for data sources and resources.
func (p *taskRunnerProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring AWS ECS client")

	// Retrieve provider data from configuration
	var providerCfg taskRunnerProviderModel
	diags := req.Config.Get(ctx, &providerCfg)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating AWS ECS client")

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create AWS ECS API Client",
			"An unexpected error occurred when creating the AWS ECS API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"TaskRunner Client Error: "+err.Error(),
		)
		return
	}
	client := ecs.NewFromConfig(cfg)

	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Configured AWS ECS client", map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *taskRunnerProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}

}

// Resources defines the resources implemented in the provider.
func (p *taskRunnerProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewRunResource,
	}
}
