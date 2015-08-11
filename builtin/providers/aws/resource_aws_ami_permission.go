package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func resourceAwsAmiPermission() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAmiPermissionPut,
		Read:   resourceAwsAmiPermissionRead,
		Update: resourceAwsAmiPermissionUpdate,
		Delete: resourceAwsAmiPermissionDelete,

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

func resourceAwsAmiPermissionPut(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	ami := d.Get("ami").(string)
	perms := resourceAwsAmiPermissionGetLP(d)
	log.Printf("%#v", perms)

	_, err := ec2conn.ModifyImageAttribute(
		&ec2.ModifyImageAttributeInput{
			ImageID:   aws.String(ami),
			Attribute: aws.String(ec2.ImageAttributeNameLaunchPermission),
			LaunchPermission: &ec2.LaunchPermissionModifications{
				Add: perms,
			},
		})

	if err != nil {
		return fmt.Errorf("Error modifying AMI permissions (%s): %s", ami, err)
	}
	return resourceAwsAmiPermissionRead(d, meta)
}

func resourceAwsAmiPermissionUpdate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	// First check if we have any changes
	if !d.HasChange("accounts") {
		return nil
	}

	// Get the AMI ID
	ami := d.Get("ami").(string)

	// Gather all of the changes and diff the sets
	o, n := d.GetChange("accounts")
	oas := o.(*schema.Set).Difference(n.(*schema.Set))
	nas := n.(*schema.Set).Difference(o.(*schema.Set))

	// Accumulate any new accounts that need permissions added
	addPerms := make([]*ec2.LaunchPermission, nas.Len())
	for _, acc := range nas.List() {
		addPerms = append(addPerms, &ec2.LaunchPermission{
			UserID: aws.String(acc.(string)),
		})
	}

	// Add any new accounts to the AMI permissions
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
		delPerms = append(delPerms, &ec2.LaunchPermission{
			UserID: aws.String(acc.(string)),
		})
	}

	// Modify the AMI permissions to bring the state in sync
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

	// Save the current state
	accounts := o.(*schema.Set).Intersection(n.(*schema.Set))
	d.Set("accounts", accounts)

	return nil
}

func resourceAwsAmiPermissionRead(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	ami := d.Get("ami").(string)

	resp, err := ec2conn.DescribeImageAttribute(
		&ec2.DescribeImageAttributeInput{
			ImageID:   aws.String(ami),
			Attribute: aws.String(ec2.ImageAttributeNameLaunchPermission),
		})

	if err != nil {
		// If we get a 404, mark the perms as destroyed
		if awsErr, ok := err.(awserr.RequestFailure); ok && awsErr.StatusCode() == 404 {
			d.SetId("")
			log.Printf("[WARN] Error reading AMI permission (%s), not found (HTTP status 404)", ami)
			return nil
		}
		return err
	}

	d.SetId(ami)
	exist := d.Get("accounts").(*schema.Set)
OUTER:
	for _, lp := range exist.List() {
		for _, remote := range resp.LaunchPermissions {
			if *remote.UserID == lp.(string) {
				continue OUTER
			}
		}
		exist.Remove(lp)
	}
	for _, lp := range resp.LaunchPermissions {
		exist.Add(*lp.UserID)
	}
	d.Set("accounts", exist)

	log.Printf("[DEBUG] Reading AMI permission meta: %s", resp)
	return nil
}

func resourceAwsAmiPermissionDelete(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	ami := d.Get("ami").(string)
	perms := resourceAwsAmiPermissionGetLP(d)

	// Build the new perms
	_, err := ec2conn.ModifyImageAttribute(
		&ec2.ModifyImageAttributeInput{
			ImageID:   aws.String(ami),
			Attribute: aws.String(ec2.ImageAttributeNameLaunchPermission),
			LaunchPermission: &ec2.LaunchPermissionModifications{
				Remove: perms,
			},
		})

	if err != nil {
		return fmt.Errorf("Error removing AMI permissions (%s): %s", ami, err)
	}
	return nil
}

func resourceAwsAmiPermissionGetLP(d *schema.ResourceData) []*ec2.LaunchPermission {
	perms := d.Get("accounts").(*schema.Set)
	lp := make([]*ec2.LaunchPermission, perms.Len())
	for _, perm := range perms.List() {
		lp = append(lp, &ec2.LaunchPermission{
			UserID: aws.String(perm.(string)),
		})
	}
	return lp
}
