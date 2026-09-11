import { DataSource_AuthenticationType } from "@/types/proto-es/v1/instance_service_pb";

export const isIAMAuthentication = (type: DataSource_AuthenticationType) =>
  type === DataSource_AuthenticationType.AWS_RDS_IAM ||
  type === DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM ||
  type === DataSource_AuthenticationType.AZURE_IAM;
