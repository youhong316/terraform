package linode

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pearkes/linode"
)

func TestAccLinodeNode_Basic(t *testing.T) {
	var node linode.Node

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLinodeNodeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckLinodeNodeConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLinodeNodeExists("linode_node.foobar", &node),
					testAccCheckLinodeNodeAttributes(&node),
				),
			},
		},
	})
}

func testAccCheckLinodeNodeDestroy(s *terraform.State) error {
	client := testAccProvider.client

	for _, rs := range s.Resources {
		if rs.Type != "linode_node" {
			continue
		}

		// Try to find the Node
		_, err := client.RetrieveNode(rs.ID)

		// Wait

		if err != nil && !strings.Contains(err.Error(), "404") {
			return fmt.Errorf(
				"Error waiting for node (%s) to be destroyed: %s",
				rs.ID, err)
		}
	}

	return nil
}

func testAccCheckLinodeNodeAttributes(node *linode.Node) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// check things
		return nil
	}
}

func testAccCheckLinodeNodeExists(n string, node *linode.Node) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No Node ID is set")
		}

		client := testAccProvider.client

		retrieveNode, err := client.RetrieveNode(rs.ID)

		if err != nil {
			return err
		}

		if retrieveNode.ID != rs.ID {
			return fmt.Errorf("Node not found")
		}

		*node = retrieveNode

		return nil
	}
}

const testAccCheckLinodeNodeConfig_basic = `
resource "linode_node" "foobar" {
}
`
