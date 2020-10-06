package v7

import (
	"fmt"

	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3/constant"
	"code.cloudfoundry.org/cli/command/flag"
)

type DeleteServiceCommand struct {
	BaseCommand

	RequiredArgs    flag.ServiceInstance `positional-args:"yes"`
	Force           bool                 `short:"f" long:"force" description:"Force deletion without confirmation"`
	Wait            bool                 `short:"w" long:"wait" description:"Wait for the delete operation to complete"`
	relatedCommands interface{}          `related_commands:"unbind-service, services"`
}

func (cmd DeleteServiceCommand) Execute(args []string) error {
	if err := cmd.SharedActor.CheckTarget(true, true); err != nil {
		return err
	}

	if !cmd.Force {
		delete, err := cmd.displayPrompt()
		if err != nil {
			return err
		}

		if !delete {
			cmd.UI.DisplayText("Delete cancelled")
			return nil
		}
	}

	if err := cmd.displayEvent(); err != nil {
		return err
	}

	stream, warnings, err := cmd.Actor.DeleteServiceInstance(
		string(cmd.RequiredArgs.ServiceInstance),
		cmd.Config.TargetedSpace().GUID,
	)
	cmd.UI.DisplayWarnings(warnings)
	if err != nil {
		return err
	}

	var (
		previousState constant.JobState
		gone          bool
	)

	for event := range stream {
		if event.State != previousState {
			cmd.UI.DisplayNewline()
			cmd.UI.DisplayText("The job has changed to {{.State}} state.", map[string]interface{}{"State": event.State})
			previousState = event.State
		} else {
			fmt.Fprint(cmd.UI.Writer(), ".")
		}

		switch {

		case event.State == constant.JobComplete:
			gone = true
			break
		case event.State == constant.JobPolling && !cmd.Wait:
			break
		}
	}

	cmd.UI.DisplayNewline()
	switch gone {
	case true:
		cmd.UI.DisplayText("Service instance {{.ServiceInstanceName}} deleted.", cmd.serviceInstanceName())
	default:
		cmd.UI.DisplayText("Delete in progress. Use 'cf services' or 'cf service {{.ServiceInstanceName}}' to check operation status.", cmd.serviceInstanceName())
	}
	cmd.UI.DisplayOK()

	return nil
}

func (cmd DeleteServiceCommand) Usage() string {
	return "CF_NAME delete-service SERVICE_INSTANCE [-f] [-w]"
}

func (cmd DeleteServiceCommand) displayEvent() error {
	user, err := cmd.Config.CurrentUser()
	if err != nil {
		return err
	}

	cmd.UI.DisplayTextWithFlavor(
		"Deleting service instance {{.ServiceInstanceName}} in org {{.OrgName}} / space {{.SpaceName}} as {{.Username}}...",
		map[string]interface{}{
			"ServiceInstanceName": cmd.RequiredArgs.ServiceInstance,
			"OrgName":             cmd.Config.TargetedOrganization().Name,
			"SpaceName":           cmd.Config.TargetedSpace().Name,
			"Username":            user.Name,
		},
	)

	return nil
}

func (cmd DeleteServiceCommand) displayPrompt() (bool, error) {
	delete, err := cmd.UI.DisplayBoolPrompt(
		false,
		"Really delete the service instance {{.ServiceInstanceName}}?",
		cmd.serviceInstanceName(),
	)
	if err != nil {
		return false, err
	}

	return delete, nil
}

func (cmd DeleteServiceCommand) serviceInstanceName() map[string]interface{} {
	return map[string]interface{}{
		"ServiceInstanceName": cmd.RequiredArgs.ServiceInstance,
	}
}
