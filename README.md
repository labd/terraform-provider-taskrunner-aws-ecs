# terraform-provider-taskrunner-aws-ecs

Provider to easily run one-off tasks on an ECS cluster.

## Example
```hcl
resource "taskrunner-aws-ecs_run" "run_migrations" {
  task_definition = "${aws_ecs_task_definition.web.family}:${aws_ecs_task_definition.web.revision}"
  ecs_cluster_arn = var.cluster_arn
  command         = "manage.py migrate"
  container       = "django"
  max_wait_time   = 600 # 10 minutes
}
```
