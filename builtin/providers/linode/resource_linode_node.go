package linode

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pearkes/linode"
)

func resource_linode_node_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	// Build up our creation options
	opts := linode.CreateNode{}

	log.Printf("[DEBUG] Node create configuration: %#v", opts)

	id, err := client.CreateNode(&opts)

	if err != nil {
		return nil, fmt.Errorf("Error creating Node: %s", err)
	}

	// Assign the nodes id
	rs.ID = id

	log.Printf("[INFO] Node ID: %s", id)

	return resource_linode_node_update_state(rs, node)
}

func resource_linode_node_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	client := p.client

	log.Printf("[INFO] Deleting Node: %s", s.ID)

	// Destroy the node
	err := client.DestroyNode(s.ID)

	// Handle remotely destroyed nodes
	if err != nil && strings.Contains(err.Error(), "404 Not Found") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("Error deleting Node: %s", err)
	}

	return nil
}

func resource_linode_node_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	node, err := resource_linode_node_retrieve(s.ID, client)

	// Handle remotely destroyed nodes
	if err != nil && strings.Contains(err.Error(), "404 Not Found") {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return resource_linode_node_update_state(s, node)
}

func resource_linode_node_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{},

		ComputedAttrs: []string{},
	}

	return b.Diff(s, c)
}

func resource_linode_node_update_state(
	s *terraform.ResourceState,
	node *linode.Node) (*terraform.ResourceState, error) {

	return s, nil
}

func resource_linode_node_retrieve(id string, client *linode.Client) (*linode.Node, error) {
	node, err := client.RetrieveNode(id)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving node: %s", err)
	}

	return &node, nil
}

func resource_linode_node_validation() *config.Validator {
	return &config.Validator{
		Required: []string{},
		Optional: []string{},
	}
}
