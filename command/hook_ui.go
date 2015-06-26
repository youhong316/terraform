package command

import (
	"github.com/hashicorp/terraform/terraform"
)

// UiHook is an interface that must be implemented by any Ui implementations
// for the Terraform CLI.
type UiHook interface {
	// All UI hooks must be terraform hooks
	terraform.Hook

	// Init and Close are called once per Ui in order to set them up
	// and tear them down.
	Init() error
	Close() error
}
