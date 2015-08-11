package aws

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsRDSClusterInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRDSClusterInstanceCreate,
		Read:   resourceAwsRDSClusterInstanceRead,
		Update: resourceAwsRDSClusterInstanceUpdate,
		Delete: resourceAwsRDSClusterInstanceDelete,

		Schema: map[string]*schema.Schema{
			"identifier": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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

			"writer": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},

			"cluster_identifier": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},

			"instance_class": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsRDSClusterInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	createOpts := &rds.CreateDBInstanceInput{
		DBInstanceClass:     aws.String(d.Get("instance_class").(string)),
		DBClusterIdentifier: aws.String(d.Get("cluster_identifier").(string)),
		Engine:              aws.String("aurora"),
	}

	if v := d.Get("identifier").(string); v != "" {
		createOpts.DBInstanceIdentifier = aws.String(v)
	} else {
		createOpts.DBInstanceIdentifier = aws.String(resource.UniqueId())
	}

	log.Printf("[DEBUG] Creating RDS DB Instance opts: %s", createOpts)
	resp, err := conn.CreateDBInstance(createOpts)
	if err != nil {
		return err
	}

	log.Printf("\n\n-----\nCreate Instance response: %s", resp)
	// tags := tagsFromMapRDS(d.Get("tags").(map[string]interface{}))
	d.SetId(*resp.DBInstance.DBInstanceIdentifier)

	// Use DB Instance refresh func
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating", "backing-up", "modifying"},
		Target:     "available",
		Refresh:    resourceAwsDbInstanceStateRefreshFunc(d, meta),
		Timeout:    40 * time.Minute,
		MinTimeout: 10 * time.Second,
		Delay:      10 * time.Second, // Wait 30 secs before starting
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsRDSClusterInstanceRead(d, meta)
}

func resourceAwsRDSClusterInstanceRead(d *schema.ResourceData, meta interface{}) error {
	db, err := resourceAwsDbInstanceRetrieve(d, meta)
	if err != nil {
		log.Printf("[WARN] Error on retrieving RDS Cluster Instance (%s): %s", d.Id(), err)
		d.SetId("")
		return nil
	}

	// Retreive DB Cluster information, to determine if this Instance is a writer
	conn := meta.(*AWSClient).rdsconn
	resp, err := conn.DescribeDBClusters(&rds.DescribeDBClustersInput{
		DBClusterIdentifier: db.DBClusterIdentifier,
	})

	var dbc *rds.DBCluster
	for _, c := range resp.DBClusters {
		if *c.DBClusterIdentifier == *db.DBClusterIdentifier {
			dbc = c
		}
	}

	if dbc == nil {
		return fmt.Errorf("[WARN] Error finding RDS Cluster (%s) for Cluster Instance (%s): %s",
			*db.DBClusterIdentifier, *db.DBInstanceIdentifier, err)
	}

	for _, m := range dbc.DBClusterMembers {
		if *db.DBInstanceIdentifier == *m.DBInstanceIdentifier {
			if *m.IsClusterWriter == true {
				d.Set("writer", true)
			} else {
				d.Set("writer", false)
			}
		}
	}

	if db.Endpoint != nil {
		d.Set("endpoint", db.Endpoint.Address)
		// May not be the same as db.InstancePort
		d.Set("port", db.Endpoint.Port)
	}

	return nil
}

func resourceAwsRDSClusterInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	// conn := meta.(*AWSClient).rdsconn

	return resourceAwsRDSClusterInstanceRead(d, meta)
}

func resourceAwsRDSClusterInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	log.Printf("[DEBUG] RDS Cluster Instance destroy: %v", d.Id())

	opts := rds.DeleteDBInstanceInput{DBInstanceIdentifier: aws.String(d.Id())}

	log.Printf("[DEBUG] RDS Cluster Instance destroy configuration: %v", opts)
	if _, err := conn.DeleteDBInstance(&opts); err != nil {
		return err
	}

	log.Println("[INFO] Waiting for RDS Cluster Instance to be destroyed")
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"modifying", "deleting"},
		Target:     "",
		Refresh:    resourceAwsDbInstanceStateRefreshFunc(d, meta),
		Timeout:    40 * time.Minute,
		MinTimeout: 10 * time.Second,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return err
	}

	return nil

}

func resourceAwsRDSClusterInstanceStateRefreshFunc(
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		// conn := meta.(*AWSClient).rdsconn
		// return dbc, *dbc.Status, nil
		return 42, "", nil
	}
}
