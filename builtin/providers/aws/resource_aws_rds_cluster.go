package aws

import (
	"fmt"
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsRDSCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRDSClusterCreate,
		Read:   resourceAwsRDSClusterRead,
		Update: resourceAwsRDSClusterUpdate,
		Delete: resourceAwsRDSClusterDelete,

		Schema: map[string]*schema.Schema{

			"availability_zones": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				ForceNew: true,
				Computed: true,
				Set:      schema.HashString,
			},

			"cluster_identifier": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if !regexp.MustCompile(`^[0-9a-z-]+$`).MatchString(value) {
						errors = append(errors, fmt.Errorf(
							"only lowercase alphanumeric characters and hyphens allowed in %q", k))
					}
					if !regexp.MustCompile(`^[a-z]`).MatchString(value) {
						errors = append(errors, fmt.Errorf(
							"first character of %q must be a letter", k))
					}
					if regexp.MustCompile(`--`).MatchString(value) {
						errors = append(errors, fmt.Errorf(
							"%q cannot contain two consecutive hyphens", k))
					}
					if regexp.MustCompile(`-$`).MatchString(value) {
						errors = append(errors, fmt.Errorf(
							"%q cannot end with a hyphen", k))
					}
					return
				},
			},

			"database_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"engine": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"master_username": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"master_password": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			// apply_immediately is used to determine when the update modifications
			// take place.
			// See http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.DBInstance.Modifying.html
			"apply_immediately": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"vpc_security_group_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsRDSClusterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	createOpts := &rds.CreateDBClusterInput{
		DBClusterIdentifier: aws.String(d.Get("cluster_identifier").(string)),
		Engine:              aws.String("aurora"),
		MasterUserPassword:  aws.String(d.Get("master_password").(string)),
		MasterUsername:      aws.String(d.Get("master_username").(string)),
	}

	if v := d.Get("database_name"); v.(string) != "" {
		createOpts.DatabaseName = aws.String(v.(string))
	}

	if attr, ok := d.GetOk("port"); ok {
		createOpts.Port = aws.Int64(int64(attr.(int)))
	}

	if attr := d.Get("vpc_security_group_ids").(*schema.Set); attr.Len() > 0 {
		createOpts.VPCSecurityGroupIDs = expandStringList(attr.List())
	}

	resp, err := conn.CreateDBCluster(createOpts)
	if err != nil {
		log.Printf("[ERROR] Error creating RDS Cluster: %s", err)
		return err
	}

	log.Printf("[DEBUG]: Cluster create response: %s", resp)

	d.SetId(*resp.DBCluster.DBClusterIdentifier)

	// tags := tagsFromMapRDS(d.Get("tags").(map[string]interface{}))
	return resourceAwsRDSClusterRead(d, meta)
}

func resourceAwsRDSClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	resp, err := conn.DescribeDBClusters(&rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(d.Id()),
		// final snapshot identifier
	})

	if err != nil {
		log.Printf("[WARN] Error describing RDS Cluster (%s)", d.Id())
		return err
	}

	if len(resp.DBClusters) != 1 ||
		*resp.DBClusters[0].DBClusterIdentifier != d.Id() {
		log.Printf("[WARN] RDS DB Cluster (%s) not found", d.Id())
		d.SetId("")
		return nil
	}

	dbc := resp.DBClusters[0]

	d.Set("database_name", dbc.DatabaseName)
	d.Set("engine", dbc.Engine)
	d.Set("master_username", dbc.MasterUsername)
	d.Set("port", dbc.Port)
	if err := d.Set("availability_zones", aws.StringValueSlice(dbc.AvailabilityZones)); err != nil {
		log.Printf("[DEBUG] Error saving AvailabilityZones to state for RDS Cluster (%s):", d.Id(), err)
	}

	var vpcg []string
	for _, g := range dbc.VPCSecurityGroups {
		vpcg = append(vpcg, *g.VPCSecurityGroupID)
	}
	if err := d.Set("vpc_security_group_ids", vpcg); err != nil {
		log.Printf("[DEBUG] Error saving VPC Security Group IDs to state for RDS Cluster (%s):", d.Id(), err)
	}

	return nil
}

func resourceAwsRDSClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	req := &rds.ModifyDBClusterInput{
		ApplyImmediately:    aws.Bool(d.Get("apply_immediately").(bool)),
		DBClusterIdentifier: aws.String(d.Id()),
	}

	if d.HasChange("master_password") {
		req.MasterUserPassword = aws.String(d.Get("master_password").(string))
	}

	if d.HasChange("vpc_security_group_ids") {
		if attr := d.Get("vpc_security_group_ids").(*schema.Set); attr.Len() > 0 {
			req.VPCSecurityGroupIDs = expandStringList(attr.List())
		} else {
			req.VPCSecurityGroupIDs = []*string{}
		}
	}

	resp, err := conn.ModifyDBCluster(req)
	if err != nil {
		return fmt.Errorf("[WARN] Error modifying RDS Cluster (%s): %s", d.Id(), err)
	}

	log.Printf("\n\n-----\n modify response: %s", resp)

	return resourceAwsRDSClusterRead(d, meta)
}

func resourceAwsRDSClusterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	log.Printf("[DEBUG] Destroying RDS Cluster (%s)", d.Id())

	r, err := conn.DeleteDBCluster(&rds.DeleteDBClusterInput{
		DBClusterIdentifier: aws.String(d.Id()),
		SkipFinalSnapshot:   aws.Bool(true),
		// final snapshot identifier
	})

	log.Printf("\n\n-----\n Delete response: %s", r)

	// wait for state delete

	if err != nil {
		log.Printf("[WARN] Error deleting RDS Cluster (%s): %s", d.Id(), err)
		return err
	}

	return nil
}
