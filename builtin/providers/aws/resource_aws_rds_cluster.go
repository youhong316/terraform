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

	resp, err := conn.CreateDBCluster(createOpts)
	if err != nil {
		log.Printf("[ERROR] Error creating RDS Cluster: %s", err)
		return err
	}

	log.Printf("[DEBUG]: Cluster create response: %s", resp)

	d.SetId(*resp.DBCluster.DBClusterIdentifier)

	log.Printf("\n\n-----\n ID set: %s", d.Id())

	// tags := tagsFromMapRDS(d.Get("tags").(map[string]interface{}))
	return resourceAwsRDSClusterRead(d, meta)
}

func resourceAwsRDSClusterRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsRDSClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	// conn := meta.(*AWSClient).rdsconn
	return resourceAwsRDSClusterRead(d, meta)
}

func resourceAwsRDSClusterDelete(d *schema.ResourceData, meta interface{}) error {
	// conn := meta.(*AWSClient).rdsconn

	// log.Printf("[DEBUG] DB Instance destroy: %v", d.Id())
	return nil
}
