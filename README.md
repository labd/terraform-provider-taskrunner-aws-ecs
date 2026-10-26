# terraform-provider-taskrunner-aws-ecs

A Terraform provider that runs one-off tasks on an AWS ECS cluster as part of
`terraform apply`. It's typically used to run database migrations or other
setup commands with a freshly deployed task definition before (or alongside)
rolling out a service. The `taskrunner-aws-ecs_run` resource starts a single
ECS task, optionally overrides the command of one container, waits for the
task to stop, and fails the apply when the task fails or the container exits
with a non-zero exit code.

## Requirements

- Terraform 1.0+ or OpenTofu (the provider uses plugin protocol v6)
- AWS credentials with permission to call `ecs:RunTask` and `ecs:DescribeTasks`,
  plus `iam:PassRole` for the task definition's task and execution roles
- To build from source: Go 1.27 (see [`go.mod`](go.mod))

## Installation

The provider is published on the Terraform Registry as
[`labd/taskrunner-aws-ecs`](https://registry.terraform.io/providers/labd/taskrunner-aws-ecs/latest).

```hcl
terraform {
  required_providers {
    taskrunner-aws-ecs = {
      source  = "labd/taskrunner-aws-ecs"
      version = "~> 0.0.2"
    }
  }
}

provider "taskrunner-aws-ecs" {}
```

## Configuration and authentication

The provider has no configuration arguments. It builds its AWS client with the
AWS SDK for Go v2 default configuration chain (`config.LoadDefaultConfig`), so
it reads the standard AWS settings, for example:

- `AWS_REGION` / `AWS_DEFAULT_REGION`
- `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`
- `AWS_PROFILE` together with `~/.aws/config` and `~/.aws/credentials`
- web identity, SSO, ECS container, and EC2 instance role credentials

The provider doesn't share configuration with the `hashicorp/aws` provider.
You have to set the region and credentials through the environment or the
shared AWS config files.

## Usage

The provider has a single resource,
[`taskrunner-aws-ecs_run`](https://registry.terraform.io/providers/labd/taskrunner-aws-ecs/latest/docs/resources/run)
(see also [`docs/resources/run.md`](docs/resources/run.md)).

```hcl
resource "taskrunner-aws-ecs_run" "run_migrations" {
  task_definition = "${aws_ecs_task_definition.web.family}:${aws_ecs_task_definition.web.revision}"
  ecs_cluster_arn = var.cluster_arn
  container       = "django"
  command         = "manage.py migrate"
  max_wait_time   = 600 # seconds
}

resource "aws_ecs_service" "web" {
  # ...
  depends_on = [taskrunner-aws-ecs_run.run_migrations]
}
```

| Argument          | Required | Description                                                         |
| ----------------- | -------- | ------------------------------------------------------------------- |
| `task_definition` | yes      | Task definition to run, as `family:revision` (or an ARN).           |
| `ecs_cluster_arn` | yes      | ECS cluster to run the task on.                                     |
| `container`       | yes      | Container whose command is overridden and whose exit code is checked. |
| `command`         | no       | Command for `container`. If you leave it out, the task definition's command runs. |
| `max_wait_time`   | no       | Maximum time in seconds to wait for the task to finish.             |

How it behaves:

- The task runs on **create** and again on **every update**, meaning any change
  to an argument, such as a new task definition revision. Destroying the
  resource does nothing in AWS.
- The provider splits `command` on whitespace and passes it as the container
  command. It doesn't interpret shell quoting, so
  `command = "sh -c 'a && b'"` won't work as you'd expect.
- The provider calls `RunTask` with only the cluster, the task definition and
  a count of 1. It passes no launch type, capacity provider, or network
  configuration, so the cluster's default capacity provider strategy applies.
  A Fargate or `awsvpc` task that needs an explicit network configuration
  can't be run this way.
- The apply fails when the task stops for any reason other than an essential
  container exiting, or when `container` exits with a non-zero exit code.
- Always set `max_wait_time`. The schema documents a 5 minute default, but the
  provider only applies it when the value is unknown, not when you leave it
  out. If you leave it out, the task still starts, but the wait then fails
  with `maximum wait time for waiter must be greater than zero`.

## Development

```sh
make build        # go build
make test         # go test -v ./...
make format       # go fmt ./...
make docs         # go generate (regenerates docs/ with tfplugindocs)
make build-local  # install as version 99.0.0 in ~/.terraform.d/plugins (darwin_arm64)
```

CI ([`.github/workflows/tests.yml`](.github/workflows/tests.yml)) runs
golangci-lint, `go test -race -v ./...` and `govulncheck`. It also runs
`go generate ./...` and fails when `docs/` changes, so commit regenerated docs
along with any schema change.

To test a local build, run `make build-local` and pin `version = "99.0.0"` in
`required_providers`. The Makefile hardcodes `OS_ARCH = darwin_arm64`, so on
another platform you have to override it, for example
`make build-local OS_ARCH=linux_amd64`.

## Releasing

Releases are built by [GoReleaser](https://goreleaser.com/) through
[`.github/workflows/release.yml`](.github/workflows/release.yml) when a `v*`
tag is pushed. The workflow signs the checksums with the repository's GPG key
and publishes a GitHub release, which the Terraform Registry picks up.

1. Add an entry to [`CHANGELOG.md`](CHANGELOG.md).
2. Tag the release commit on `main` (for example `git tag v0.0.3`) and push the
   tag.

## License

The repository has no license file.
