import { 
  DataSourceType as NewDataSourceType,
  DataSource_AuthenticationType as NewDataSource_AuthenticationType,
} from "@/types/proto-es/v1/instance_service_pb";

// ========== SCOPE VALUE CONVERSIONS (for AdvancedSearch components) ==========

// Convert scope value (string or number) to proto-es DataSourceType
export const convertScopeValueToDataSourceType = (value: string | number): NewDataSourceType => {
  switch (value) {
    case 0:
    case "DATA_SOURCE_UNSPECIFIED":
      return NewDataSourceType.DATA_SOURCE_UNSPECIFIED;
    case 1:
    case "ADMIN":
      return NewDataSourceType.ADMIN;
    case 2:
    case "READ_ONLY":
      return NewDataSourceType.READ_ONLY;
    case -1:
    case "UNRECOGNIZED":
    default:
      return NewDataSourceType.DATA_SOURCE_UNSPECIFIED;
  }
};

// Convert scope value (string or number) to proto-es DataSource_AuthenticationType
export const convertScopeValueToDataSourceAuthenticationType = (value: string | number): NewDataSource_AuthenticationType => {
  switch (value) {
    case 0:
    case "AUTHENTICATION_UNSPECIFIED":
      return NewDataSource_AuthenticationType.AUTHENTICATION_UNSPECIFIED;
    case 1:
    case "PASSWORD":
      return NewDataSource_AuthenticationType.PASSWORD;
    case 2:
    case "GOOGLE_CLOUD_SQL_IAM":
      return NewDataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM;
    case 3:
    case "AWS_RDS_IAM":
      return NewDataSource_AuthenticationType.AWS_RDS_IAM;
    case 4:
    case "AZURE_IAM":
      return NewDataSource_AuthenticationType.AZURE_IAM;
    case -1:
    case "UNRECOGNIZED":
    default:
      return NewDataSource_AuthenticationType.AUTHENTICATION_UNSPECIFIED;
  }
};
