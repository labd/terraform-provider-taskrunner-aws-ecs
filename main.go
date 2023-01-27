package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/labd/terraform-provider-aws-ecs-taskrunner/internal"
)

// Provider documentation generation.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name taskrunner-aws-ecs

func main() {
	providerserver.Serve(context.Background(), internal.New, providerserver.ServeOpts{
		Address: "github.com/labd/terraform-provider-taskrunner-aws-ecs",
	})
}
