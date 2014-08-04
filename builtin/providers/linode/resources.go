package linode

import (
	"github.com/hashicorp/terraform/helper/resource"
)

// resourceMap is the mapping of resources we support to their basic
// operations. This makes it easy to implement new resource types.
var resourceMap *resource.Map

func init() {
	resourceMap = &resource.Map{
		Mapping: map[string]resource.Resource{
			"linode_node": resource.Resource{
				ConfigValidator: resource_linode_node_validation(),
				Create:          resource_linode_node_create,
				Destroy:         resource_linode_node_destroy,
				Diff:            resource_linode_node_diff,
				Refresh:         resource_linode_node_refresh,
				Update:          resource_linode_node_update,
			},
		},
	}
}
