package internal

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/elliotchance/pie/v2"
)

// tasksRunningStateRetryable is a function used by the waiter to decide if a
// task state is retryable or a terminal state. The function returns an error in
// case of a failure state. In case of retry state, this function returns a bool
// value of true and nil error, while in case of success it returns a bool value
// of false and nil error.
func taskWaiter(containerName string) func(ctx context.Context, input *ecs.DescribeTasksInput, output *ecs.DescribeTasksOutput, err error) (bool, error) {

	return func(ctx context.Context, input *ecs.DescribeTasksInput, output *ecs.DescribeTasksOutput, err error) (bool, error) {
		if len(output.Tasks) == 1 {
			task := output.Tasks[0]

			if *task.LastStatus == "PENDING" {
				return true, nil
			}

			if *task.LastStatus == "RUNNING" {
				return true, nil
			}

			if *task.LastStatus == "STOPPED" {
				if task.StopCode != types.TaskStopCodeEssentialContainerExited {
					return false, fmt.Errorf("Task failed: %s", task.StopCode)
				}

				// Check exit code from the container
				index := pie.FindFirstUsing(task.Containers, func(c types.Container) bool {
					return *c.Name == containerName
				})

				// This should never happen ;-)
				if index < 0 {
					return false, fmt.Errorf("cannot find container reference")
				}

				container := task.Containers[index]
				if *container.ExitCode == 0 {
					return false, nil
				}
				return false, fmt.Errorf("command returned exit code %d", *container.ExitCode)
			}
		}
		return false, err
	}
}
