module "redshift_loader_lambda" {

  source = "github.com/diogoaurelio/terraform-aws-lambda-module"
  version = "v0.0.1"

  aws_region     = "${var.aws_region}"
  environment    = "${var.environment}"
  project        = "${var.project}"

  lambda_unique_function_name = "${var.redshift_loader_lambda_unique_function_name}"
  runtime                     = "${var.redshift_loader_lambda_runtime}"
  handler                     = "${var.redshift_loader_lambda_handler}"
  lambda_iam_role_name        = "${var.redshift_loader_lambda_role_name}"

  main_lambda_file  = "${var.redshift_loader_main_lambda_file}"
  lambda_source_dir = "${local.redshift_loader_lambda_dir}/src"

  lambda_zip_file_location = "${local.redshift_loader_lambda_dir}/${var.redshift_loader_main_lambda_file}.zip"
  lambda_env_vars          = "${local.redshift_loader_lambda_env_vars}"

  additional_policy = "${data.aws_iam_policy_document.this.json}"
  attach_policy     = true

  # configure Lambda function inside a specific VPC
  security_group_ids = ["${aws_security_group.this.id}"]
  subnet_ids         = "${split(",", var.private_subnet_ids)}"

  # DLQ
  use_dead_letter_config_target_arn = true
  dead_letter_config_target_arn     = "${aws_sns_topic.lambda_sns_dql.arn}"
}

################################################################################
# Locals used for different Lambdas Environmental Variables
################################################################################

locals {

  redshift_loader_lambda_env_vars = {
    ENVIRONMENT = "${var.environment}"
    REGION      = "${var.aws_region}"
    IAM_ROLE    = "${var.redshift_data_loader_lambda_iam_role}"

    DB_HOST     = "${var.redshift_data_loader_lambda_db_host}"
    DB_PORT     = "${var.redshift_data_loader_lambda_db_port}"
    DB_NAME     = "${var.redshift_data_loader_lambda_db_name}"

    DB_USER     = "${var.redshift_data_loader_lambda_db_user}"
    DB_PW_PARAM = "${aws_ssm_parameter.redshift_lambda_db_password.name}"
    DB_SCHEMA   = "${var.redshift_data_loader_lambda_db_schema}"
    DB_TABLE    = "${var.redshift_data_loader_lambda_db_table}"
  }

  redshift_loader_lambda_dir = "${path.cwd}/../../../etl/lambda/redshift/"
}

################################################################################
# AWS SSM secret for Redshift user password
################################################################################

resource "aws_ssm_parameter" "redshift_lambda_db_password" {
  name        = "${var.environment}-${var.project}-redshift-lambda-password"
  description = "${var.environment} redshift lambda user password"
  type        = "SecureString"
  value       = "${var.redshift_data_loader_lambda_db_password}"
  key_id      = "${aws_kms_key.redshift_secrets_key.arn}"

  tags {
    Environment = "${var.environment}"
    Project     = "${var.project}"
    Name        = "redshift-lambda-password"
  }
}


################################################################################
# AWS Lambda IAM Policy document definitions
################################################################################
data "aws_iam_policy_document" "this" {
  statement {
    effect = "Allow"

    actions = [
      "s3:GetBucketLocation",
      "s3:ListAllMyBuckets",
    ]

    resources = [
      "*",
    ]
  }

  statement {
    effect = "Allow"

    actions = [
      "s3:Get*",
      "s3:List*",
      "s3:Describe*",
      "s3:RestoreObject",
    ]

    resources = [
      "*",
    ]
  }

  # Required if Lambda is created inside VPC
  statement {
    effect = "Allow"

    actions = [
      "ec2:DescribeNetworkInterfaces",
      "ec2:CreateNetworkInterface",
      "ec2:DeleteNetworkInterface",
    ]

    resources = [
      "*",
    ]
  }

  # Required if lambda requires specific Secrets from AWS SSM
  statement {
    effect = "Allow"

    actions = [
      "ssm:GetParameter",
    ]

    resources = [
      "${aws_ssm_parameter.redshift_lambda_db_password.arn}",
    ]
  }

  # Required if using encrypted data
  statement {
    effect = "Allow"

    actions = [
      "kms:ListKeys",
      "kms:Encrypt",
      "kms:Decrypt",
    ]

    resources = [
      "*",
    ]
  }

  statement {
    effect = "Allow"

    actions = [
      "sns:Publish",
    ]

    resources = [
      "${aws_sns_topic.lambda_sns_dql.arn}",
    ]
  }
  statement {
    effect = "Allow"

    actions = [
      "sqs:Publish",
    ]

    resources = [
      "${aws_sqs_message.lambda_sqs_dql.arn}",
      ]
  }
}  
