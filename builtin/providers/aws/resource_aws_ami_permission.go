package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func resourceAwsAmiPermission() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAmiPermissionUpdate,
		Read:   resourceAwsAmiPermissionRead,
		Update: resourceAwsAmiPermissionUpdate,
		Delete: resourceAwsAmiPermissionUpdate,

		Schema: map[string]*schema.Schema{
			"ami": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"accounts": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Set: schema.HashString,
			},
		},
	}
}

func resourceAwsAmiPermissionUpdate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	// Get the AMI ID
	ami := d.Get("ami").(string)

	// First check if we have any changes
	if !d.HasChange("accounts") {
		log.Printf("[DEBUG] No changes detected for AMI permissions (%s)", ami)
		return nil
	}

	// Gather all of the changes and diff the sets
	o, n := d.GetChange("accounts")
	oas := o.(*schema.Set).Difference(n.(*schema.Set))
	nas := n.(*schema.Set).Difference(o.(*schema.Set))

	// Accumulate any new accounts that need permissions added
	addPerms := make([]*ec2.LaunchPermission, nas.Len())
	for _, acc := range nas.List() {
		account := acc.(string)
		log.Printf("[DEBUG] Adding AMI permission (%s): %s", ami, account)
		addPerms = append(addPerms, &ec2.LaunchPermission{
			UserID: aws.String(account),
		})
	}

	// Add the new account permissions
	_, err := ec2conn.ModifyImageAttribute(
		&ec2.ModifyImageAttributeInput{
			ImageID:   aws.String(ami),
			Attribute: aws.String(ec2.ImageAttributeNameLaunchPermission),
			LaunchPermission: &ec2.LaunchPermissionModifications{
				Add: addPerms,
			},
		})
	if err != nil {
		return fmt.Errorf("Error adding AMI permissions (%s): %s", ami, err)
	}

	// Accumulate all of the obsolete account ID's to delete
	delPerms := make([]*ec2.LaunchPermission, oas.Len())
	for _, acc := range oas.List() {
		account := acc.(string)
		log.Printf("[DEBUG] Removing AMI permission (%s): %s", ami, account)
		delPerms = append(delPerms, &ec2.LaunchPermission{
			UserID: aws.String(account),
		})
	}

	// Remove the obsolete account ID's
	_, err = ec2conn.ModifyImageAttribute(
		&ec2.ModifyImageAttributeInput{
			ImageID:   aws.String(ami),
			Attribute: aws.String(ec2.ImageAttributeNameLaunchPermission),
			LaunchPermission: &ec2.LaunchPermissionModifications{
				Remove: delPerms,
			},
		})
	if err != nil {
		return fmt.Errorf("Error removing AMI permissions (%s): %s", ami, err)
	}

	return resourceAwsAmiPermissionRead(d, meta)
}

func resourceAwsAmiPermissionRead(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	ami := d.Get("ami").(string)

	// Get the current set of permissions from AWS
	resp, err := ec2conn.DescribeImageAttribute(
		&ec2.DescribeImageAttributeInput{
			ImageID:   aws.String(ami),
			Attribute: aws.String(ec2.ImageAttributeNameLaunchPermission),
		})
	if err != nil {
		return fmt.Errorf("Error describing AMI permissions (%s): %s", ami, err)
	}

	d.SetId(ami)

	// Get the current set of accounts from the local state
	acc := d.Get("accounts").(*schema.Set)

	// Add all of the remote accounts into our local state
	for _, lp := range resp.LaunchPermissions {
		acc.Add(*lp.UserID)
	}

	// Build the set of remote account permissions
	remotes := make(map[string]struct{}, acc.Len())
	for _, remote := range resp.LaunchPermissions {
		remotes[*remote.UserID] = struct{}{}
	}

	// Go over all of the locally known accounts and ensure
	// they still exist remotely in AWS.
	for _, lp := range acc.List() {
		if _, ok := remotes[lp.(string)]; !ok {
			acc.Remove(lp)
		}
	}

	// Save the current state of the AMI permissions
	d.Set("accounts", acc)
	return nil
}
