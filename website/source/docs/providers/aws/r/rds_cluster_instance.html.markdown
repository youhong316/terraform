---
layout: "aws"
page_title: "AWS: aws_rds_cluster"
sidebar_current: "docs-aws-resource-rds-cluster-instance"
description: |-
  Provides an RDS Cluster Resource Instance
---

# aws\_rds\_cluster\_instance

Provides an RDS Cluster Resource Instance. A Cluster Instance Resource defines 
attributes that are specific to a single instance in a [RDS Cluster][3],
specifically running Amazon Aurora.

Unlike other RDS resources that support replication, with Amazon Aurora you do
not designate a primary and subsequent replicas. Instead, you simply add RDS
Instances and Aurora manages the replication.

For more information on Amazon Aurora, see [Aurora on Amazon RDS][2] in the Amazon RDS User Guide.

## Example Usage

```
resource "aws_rds_cluster_instance" "cluster_instances" {
  count = 2
  instance_identifier = "aurora-cluster-demo"
  cluster_identifer = "${aws_rds_cluster.default.id}"
  instance_class = "db.r3.large"
}

resource "aws_rds_cluster" "default" {
  cluster_identifier = "aurora-cluster-demo"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  database_name = "mydb"
  master_username = "foo"
  master_password = "bar"
}
```

## Argument Reference

For more detailed documentation about each argument, refer to
the [AWS official documentation](http://docs.aws.amazon.com/AmazonRDS/latest/CommandLineReference/CLIReference-cmd-ModifyDBInstance.html).

The following arguments are supported:

* `instance_identifier` - (Required) The Instance Identifer. Must be a lower case
string.
* `cluster_identifier` - (Required) The Cluster Identifer. Must be a lower case
string.
* `instance_class` - (Required) The instance class to use. Aurora currently 
  supports the following instance classes:  
  - db.r3.large
  - db.r3.xlarge
  - db.r3.2xlarge
  - db.r3.4xlarge
  - db.r3.8xlarge

## Attributes Reference

The following attributes are exported:

* `cluster_identifer` - The RDS Cluster Identifer
* `instance_identifer` - The Instance identifer
* `id` - The Instance identifer
* `writer` – Boolean indicating if this instance is writable. `False` indicates
this instance is a read replica
* `address` - The address of the RDS instance.
* `allocated_storage` - The amount of allocated storage
* `availability_zones` - The availability zone of the instance
* `endpoint` - The IP address for this instance. May not be writable
* `engine` - The database engine
* `engine_version` - The database engine version
* `database_name` - The database name
* `port` - The database port
* `status` - The RDS instance status

[2]: http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_Aurora.html
[3]: /docs/providers/aws/r/rds_cluster.html
