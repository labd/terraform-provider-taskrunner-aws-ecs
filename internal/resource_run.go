package internal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecs_types "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/elliotchance/pie/v2"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &runResource{}
	_ resource.ResourceWithConfigure   = &runResource{}
	_ resource.ResourceWithImportState = &runResource{}
)

// NewOrderResource is a helper function to simplify the provider implementation.
func NewRunResource() resource.Resource {
	return &runResource{}
}

// runResource is the resource implementation.
type runResource struct {
	client *ecs.Client
}

type runResourceModel struct {
	TaskDefinition types.String `tfsdk:"task_definition"`
	ClusterARN     types.String `tfsdk:"ecs_cluster_arn"`
	Command        types.String `tfsdk:"command"`
	Container      types.String `tfsdk:"container"`
	MaxWaitTime    types.Int64  `tfsdk:"max_wait_time"`
}

func (r *runResourceModel) commandList() []string {
	return strings.Fields(r.Command.ValueString())
}

// Metadata returns the data source type name.
func (r *runResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_run"
}

// Schema defines the schema for the data source.
func (r *runResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Run a task until completed",
		Attributes: map[string]schema.Attribute{
			"task_definition": schema.StringAttribute{
				Description: "The task definition (family:revision)",
				Required:    true,
			},
			"ecs_cluster_arn": schema.StringAttribute{
				Description: "ECS Cluster to run the task on",
				Required:    true,
			},
			"container": schema.StringAttribute{
				Description: "Container to run command in",
				Required:    true,
			},
			"command": schema.StringAttribute{
				Description: "Command to run",
				Optional:    true,
			},
			"max_wait_time": schema.Int64Attribute{
				Description: "Max time to wait (default = 5 minutes)",
				Optional:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (r *runResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*ecs.Client)
}

// Create creates the resource and sets the initial Terraform state.
func (r *runResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan runResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = r.runTask(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *runResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state runResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *runResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan runResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = r.runTask(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *runResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state runResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

func (r *runResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *runResource) runTask(ctx context.Context, plan *runResourceModel) diag.Diagnostics {
	diags := diag.Diagnostics{}

	input := &ecs.RunTaskInput{
		TaskDefinition: aws.String(plan.TaskDefinition.ValueString()),
		Cluster:        aws.String(plan.ClusterARN.ValueString()),
		StartedBy:      aws.String("taskrunner-aws-ecs"),
		Count:          aws.Int32(1),
	}

	if plan.Command.ValueString() != "" {
		input.Overrides = &ecs_types.TaskOverride{
			ContainerOverrides: []ecs_types.ContainerOverride{
				{
					Name:    aws.String(plan.Container.ValueString()),
					Command: plan.commandList(),
				},
			},
		}
	}

	output, err := r.client.RunTask(ctx, input)
	if err != nil {
		diags.AddError("failed to run task", err.Error())
		return diags
	}

	waiter := ecs.NewTasksRunningWaiter(r.client)

	params := &ecs.DescribeTasksInput{
		Cluster: aws.String(plan.ClusterARN.ValueString()),
		Tasks: pie.Map(output.Tasks, func(t ecs_types.Task) string {
			return *t.TaskArn
		}),
	}

	if plan.MaxWaitTime.IsUnknown() {
		plan.MaxWaitTime = types.Int64Value(300)
	}
	waitTime := time.Duration(plan.MaxWaitTime.ValueInt64()) * time.Second
	err = waiter.Wait(ctx, params, waitTime, func(t *ecs.TasksRunningWaiterOptions) {
		t.MaxDelay = 15 * time.Second
		t.Retryable = taskWaiter(plan.Container.ValueString())
	})
	if err != nil {
		diags.AddError(
			fmt.Sprintf("failed to wait for task %s", params.Tasks[0]), err.Error())
		return diags
	}

	return diags
}
