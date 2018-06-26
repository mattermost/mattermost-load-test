variable "cluster_name" {
    default = "loadtest"
}
variable "app_instance_type" {}
variable "db_instance_type" {}
variable "app_instance_count" {
    default = 1
}
variable "db_instance_count" {
    default = 1
}
variable "loadtest_instance_count" {
    default = 1
}
variable "db_password" {}
variable "ssh_public_key" {}
variable "ssh_private_key" {}

provider "aws" {
    region = "us-east-1"
    profile = "ltops"
}

resource "aws_instance" "app_server" {
    tags {
        Name = "${var.cluster_name}-app-${count.index}"
    }
    ami = "ami-43a15f3e"
    instance_type = "${var.app_instance_type}"
    associate_public_ip_address = true
    vpc_security_group_ids = [
        "${aws_security_group.app.id}",
        "${aws_security_group.app_gossip.id}"
    ]
    key_name = "${aws_key_pair.key.id}"
    count = "${var.app_instance_count}"
    availability_zone = "us-east-1a"
}

resource "aws_key_pair" "key" {
    key_name = "Terraform-${var.cluster_name}"
    public_key = "${var.ssh_public_key}"
}

output "instanceIP" {
    value = "${aws_instance.app_server.*.public_dns}"
}

resource "aws_security_group" "app" {
    name = "${var.cluster_name}-app-security-group"
    description = "App security group for loadtest cluster ${var.cluster_name}"

    ingress {
        from_port = 22
        to_port = 22
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }
    ingress {
        from_port = 80
        to_port = 80
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }
    ingress {
        from_port = 8067
        to_port = 8067
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }
    egress {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

resource "aws_security_group" "app_gossip" {
    name = "${var.cluster_name}-app-security-group-gossip"
    description = "App security group for gossip loadtest cluster ${var.cluster_name}"
    ingress {
        from_port = 8074
        to_port = 8074
        protocol = "udp"
        security_groups = ["${aws_security_group.app.id}"]
    }
    ingress {
        from_port = 8074
        to_port = 8074
        protocol = "tcp"
        security_groups = ["${aws_security_group.app.id}"]
    }
    ingress {
        from_port = 8075
        to_port = 8075
        protocol = "udp"
        security_groups = ["${aws_security_group.app.id}"]
    }
    ingress {
        from_port = 8075
        to_port = 8075
        protocol = "tcp"
        security_groups = ["${aws_security_group.app.id}"]
    }
    egress {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

resource "aws_instance" "loadtest" {
    tags {
        Name = "${var.cluster_name}-loadtest-${count.index}"
    }
    ami = "ami-43a15f3e"
    instance_type = "m4.xlarge"
    associate_public_ip_address = true
    vpc_security_group_ids = [
        "${aws_security_group.app.id}"
    ]
    key_name = "${aws_key_pair.key.id}"
    count = "${var.loadtest_instance_count}"
    availability_zone = "us-east-1a"
}

output "loadtestInstanceIP" {
    value = "${aws_instance.loadtest.*.public_ip}"
}

resource "aws_security_group" "loadtest" {
    name = "${var.cluster_name}-loadtest-security-group"
    description = "Loadtest security group for loadtest cluster ${var.cluster_name}"

    ingress {
        from_port = 22
        to_port = 22
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }
    egress {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

resource "aws_rds_cluster_instance" "db_cluster_instances" {
    count = "${var.db_instance_count}"
    identifier = "${var.cluster_name}-db-${count.index}"
    cluster_identifier = "${aws_rds_cluster.db_cluster.id}"
    instance_class = "${var.db_instance_type}"
    publicly_accessible = true
    apply_immediately = true
    monitoring_interval = 10
    monitoring_role_arn = "${aws_iam_role.rds_enhanced_monitoring.arn}"
}

resource "aws_rds_cluster" "db_cluster" {
    cluster_identifier = "${var.cluster_name}-db"
    database_name = "mattermost"
    master_username = "mmuser"
    master_password = "${var.db_password}"
    skip_final_snapshot = true
    apply_immediately = true
    vpc_security_group_ids = ["${aws_security_group.db.id}"]
    availability_zones = ["us-east-1a"]
}

output "dbEndpoint" {
    value = "${aws_rds_cluster.db_cluster.endpoint}"
}

output "dbReaderEndpoint" {
    value = "${aws_rds_cluster.db_cluster.reader_endpoint}"
}

resource "aws_security_group" "db" {
    name = "${var.cluster_name}-db-security-group"

    ingress {
        from_port = 3306
        to_port = 3306
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

# These roles and policies are to enable enhanced monitoring for the DBs
resource "aws_iam_role" "rds_enhanced_monitoring" {
	name               = "${var.cluster_name}-rds-enhanced_monitoring-role"
	assume_role_policy = "${data.aws_iam_policy_document.rds_enhanced_monitoring.json}"
}

resource "aws_iam_role_policy_attachment" "rds_enhanced_monitoring" {
	role       = "${aws_iam_role.rds_enhanced_monitoring.name}"
	policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonRDSEnhancedMonitoringRole"
}

data "aws_iam_policy_document" "rds_enhanced_monitoring" {
	statement {
        actions = [
            "sts:AssumeRole",
        ]

        effect = "Allow"

        principals {
            type        = "Service"
            identifiers = ["monitoring.rds.amazonaws.com"]
        }
    }
}

resource "aws_instance" "proxy_server" {
    tags {
        Name = "${var.cluster_name}-proxy-${count.index}"
    }
    ami = "ami-43a15f3e"
    instance_type = "m4.xlarge"
    associate_public_ip_address = true
    vpc_security_group_ids = [
        "${aws_security_group.proxy.id}"
    ]
    key_name = "${aws_key_pair.key.id}"
    count = "${var.loadtest_instance_count}"
    availability_zone = "us-east-1a"
}

output "proxyIP" {
    value = "${aws_instance.proxy_server.*.public_dns}"
}

resource "aws_security_group" "proxy" {
    name = "${var.cluster_name}-proxy-security-group"
    description = "Proxy security group for loadtest cluster ${var.cluster_name}"

    ingress {
        from_port = 80
        to_port = 80
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

    ingress {
        from_port = 22
        to_port = 22
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

    egress {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

resource "aws_s3_bucket" "app" {
    bucket = "${var.cluster_name}.loadtestbucket"
    acl = "private"
    tags {
        Name = "${var.cluster_name}"
    }
    force_destroy = true
}

output "s3bucket" {
    value = "${aws_s3_bucket.app.id}"
}

output "s3bucketRegion" {
    value = "${aws_s3_bucket.app.region}"
}

resource "aws_iam_access_key" "s3" {
    user = "${aws_iam_user.s3.name}"
}

output "s3AccessKeyId" {
    value = "${aws_iam_access_key.s3.id}"
}

output "s3AccessKeySecret" {
    value = "${aws_iam_access_key.s3.secret}"
}

resource "aws_iam_user" "s3" {
    name = "${var.cluster_name}-s3"
}

resource "aws_iam_user_policy" "s3" {
    name = "${var.cluster_name}-s3-user-access"
    user = "${aws_iam_user.s3.name}"

    policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": [
                "s3:AbortMultipartUpload",
                "s3:DeleteObject",
                "s3:GetObject",
                "s3:GetObjectAcl",
                "s3:PutObject",
                "s3:PutObjectAcl"
            ],
            "Effect": "Allow",
            "Resource": "arn:aws:s3:::${aws_s3_bucket.app.id}/*"
        }
    ]
}
EOF
}

resource "aws_instance" "metrics" {
    tags {
        Name = "${var.cluster_name}-metrics"
    }
    ami = "ami-43a15f3e"
    instance_type = "t2.large"
    associate_public_ip_address = true
    vpc_security_group_ids = [
        "${aws_security_group.metrics.id}",
    ]
    key_name = "${aws_key_pair.key.id}"
    availability_zone = "us-east-1a"

    provisioner "file" {
        destination = "/home/ubuntu/prometheus.service"
        connection {
            user="ubuntu"
            private_key="${var.ssh_private_key}"
            agent=false
        }

        content = <<EOF
[Unit]
Description=Monitoring system and time series database
Documentation=https://prometheus.io/docs/introduction/overview/

[Service]
Restart=always
User=ubuntu
WorkingDirectory=/home/ubuntu/prometheus
ExecStart=/home/ubuntu/prometheus/prometheus $ARGS
ExecReload=/bin/kill -HUP $MAINPID
TimeoutStopSec=20s
SendSIGKILL=no

[Install]
WantedBy=multi-user.target
EOF
    }

    provisioner "remote-exec" {
        connection {
            user="ubuntu"
            private_key="${var.ssh_private_key}"
            agent=false
        }
        inline = [
            "sudo mv prometheus.service /lib/systemd/system/prometheus.service",
            "wget https://github.com/prometheus/prometheus/releases/download/v2.2.1/prometheus-2.2.1.linux-amd64.tar.gz",
            "tar -zxf *.tar.gz",
            "rm *.tar.gz",
            "mv prometheus* prometheus",
        ]
    }

    provisioner "file" {
        destination = "/home/ubuntu/prometheus/prometheus.yml"
        connection {
            user="ubuntu"
            private_key="${var.ssh_private_key}"
            agent=false
        }
        content = <<EOF
global:
  scrape_interval:     10s
  evaluation_interval: 10s
  external_labels:
      monitor: 'mattermost-monitor'

scrape_configs:
  - job_name: 'loadtest'
    static_configs:
      - targets: ["${join("\",\"", formatlist("%s:8067", aws_instance.app_server.*.public_dns))}"]
EOF
    }

    provisioner "remote-exec" {
        connection {
            user="ubuntu"
            private_key="${var.ssh_private_key}"
            agent=false
        }
        inline = [
            "sudo systemctl enable prometheus",
            "sudo systemctl start prometheus",
            "sudo apt-get update",
            "sudo apt-get install -y adduser libfontconfig",
            "wget https://s3-us-west-2.amazonaws.com/grafana-releases/release/grafana_5.0.4_amd64.deb",
            "sudo dpkg -i grafana_5.0.4_amd64.deb",
            "sudo systemctl daemon-reload",
            "sudo systemctl enable grafana-server",
            "sudo systemctl start grafana-server",
            "sudo iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 3000",
            "sleep 10"
        ]
    }
}

output "metricsIP" {
    value = "${aws_instance.metrics.public_ip}"
}

resource "aws_security_group" "metrics" {
    name = "${var.cluster_name}-metrics-security-group"
    description = "Metrics security group for loadtest cluster ${var.cluster_name}"

    ingress {
        from_port = 80
        to_port = 80
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

    ingress {
        from_port = 9090
        to_port = 9090
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

    ingress {
        from_port = 22
        to_port = 22
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

    egress {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }
}
/*
provider "grafana" {
    url = "http://${aws_instance.metrics.public_ip}"
    auth = "admin:admin"
}

resource "grafana_data_source" "prometheus" {
    type = "prometheus"
    name = "prometheus"
    url = "http://localhost:9090"
    depends_on = ["aws_instance.metrics"]
}*/
