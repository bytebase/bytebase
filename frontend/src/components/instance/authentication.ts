import { Engine } from "@/types/proto-es/v1/common_pb";
import { DataSource_AuthenticationType } from "@/types/proto-es/v1/instance_service_pb";

export const isIAMAuthentication = (type: DataSource_AuthenticationType) =>
  type === DataSource_AuthenticationType.AWS_RDS_IAM ||
  type === DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM ||
  type === DataSource_AuthenticationType.AZURE_IAM;

export const normalizeAuthenticationType = (
  engine: Engine,
  type: DataSource_AuthenticationType
): DataSource_AuthenticationType =>
  engine === Engine.SPANNER || engine === Engine.BIGQUERY
    ? DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
    : type;
