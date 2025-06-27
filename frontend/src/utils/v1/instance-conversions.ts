import { fromJson, toJson } from "@bufbuild/protobuf";
import { 
  Instance as OldInstance,
  DataSource as OldDataSource,
  UpdateInstanceRequest as OldUpdateInstanceRequest,
  GetInstanceRequest as OldGetInstanceRequest,
  ListInstancesRequest as OldListInstancesRequest,
  ListInstancesResponse as OldListInstancesResponse,
  CreateInstanceRequest as OldCreateInstanceRequest,
  DeleteInstanceRequest as OldDeleteInstanceRequest,
  UndeleteInstanceRequest as OldUndeleteInstanceRequest,
  SyncInstanceRequest as OldSyncInstanceRequest,
  BatchSyncInstancesRequest as OldBatchSyncInstancesRequest,
  DataSourceType as OldDataSourceType,
  DataSource_AuthenticationType as OldDataSource_AuthenticationType
} from "@/types/proto/v1/instance_service";
import { 
  type Instance as NewInstance,
  type DataSource as NewDataSource,
  type UpdateInstanceRequest as NewUpdateInstanceRequest,
  type GetInstanceRequest as NewGetInstanceRequest,
  type ListInstancesRequest as NewListInstancesRequest,
  type ListInstancesResponse as NewListInstancesResponse,
  type CreateInstanceRequest as NewCreateInstanceRequest,
  type DeleteInstanceRequest as NewDeleteInstanceRequest,
  type UndeleteInstanceRequest as NewUndeleteInstanceRequest,
  type SyncInstanceRequest as NewSyncInstanceRequest,
  type BatchSyncInstancesRequest as NewBatchSyncInstancesRequest,
  DataSourceType as NewDataSourceType,
  DataSource_AuthenticationType as NewDataSource_AuthenticationType,
  InstanceSchema,
  BatchSyncInstancesRequestSchema,
  SyncInstanceRequestSchema,
  UndeleteInstanceRequestSchema,
  CreateInstanceRequestSchema,
  DataSourceSchema,
  UpdateInstanceRequestSchema,
  GetInstanceRequestSchema,
  ListInstancesRequestSchema,
  ListInstancesResponseSchema,
  DeleteInstanceRequestSchema,
} from "@/types/proto-es/v1/instance_service_pb";

// Convert old proto to proto-es
export const convertOldInstanceToNew = (oldInstance: OldInstance): NewInstance => {
  // Use toJSON to convert old proto to JSON, then fromJson to convert to proto-es
  const json = OldInstance.toJSON(oldInstance) as any; // Type assertion needed due to proto type incompatibility
  
  // Fix Duration fields: proto-es expects string format, old proto outputs object
  if (json.syncInterval && typeof json.syncInterval === 'object') {
    const duration = json.syncInterval as { seconds?: string | number; nanos?: number };
    if (!duration.seconds && !duration.nanos) {
      delete json.syncInterval;
    } else {
      const seconds = Number(duration.seconds || 0) + (duration.nanos || 0) / 1e9;
      json.syncInterval = `${seconds}s`;
    }
  }
  
  return fromJson(InstanceSchema, json);
};

// Convert proto-es to old proto
export const convertNewInstanceToOld = (newInstance: NewInstance): OldInstance => {
  // Use toJson to convert proto-es to JSON, then fromJSON to convert to old proto
  const json = toJson(InstanceSchema, newInstance);
  return OldInstance.fromJSON(json);
};

// Convert old proto DataSource to proto-es
export const convertOldDataSourceToNew = (oldDataSource: OldDataSource): NewDataSource => {
  const json = OldDataSource.toJSON(oldDataSource) as any;
  return fromJson(DataSourceSchema, json);
};

// Convert proto-es DataSource to old proto
export const convertNewDataSourceToOld = (newDataSource: NewDataSource): OldDataSource => {
  const json = toJson(DataSourceSchema, newDataSource);
  return OldDataSource.fromJSON(json);
};

// Convert old proto UpdateInstanceRequest to proto-es
export const convertOldUpdateInstanceRequestToNew = (oldRequest: OldUpdateInstanceRequest): NewUpdateInstanceRequest => {
  const json = OldUpdateInstanceRequest.toJSON(oldRequest) as any;
  return fromJson(UpdateInstanceRequestSchema, json);
};

// Convert proto-es UpdateInstanceRequest to old proto
export const convertNewUpdateInstanceRequestToOld = (newRequest: NewUpdateInstanceRequest): OldUpdateInstanceRequest => {
  const json = toJson(UpdateInstanceRequestSchema, newRequest);
  return OldUpdateInstanceRequest.fromJSON(json);
};

// Convert old proto GetInstanceRequest to proto-es
export const convertOldGetInstanceRequestToNew = (oldRequest: OldGetInstanceRequest): NewGetInstanceRequest => {
  const json = OldGetInstanceRequest.toJSON(oldRequest) as any;
  return fromJson(GetInstanceRequestSchema, json);
};

// Convert proto-es GetInstanceRequest to old proto
export const convertNewGetInstanceRequestToOld = (newRequest: NewGetInstanceRequest): OldGetInstanceRequest => {
  const json = toJson(GetInstanceRequestSchema, newRequest);
  return OldGetInstanceRequest.fromJSON(json);
};

// Convert old proto ListInstancesRequest to proto-es
export const convertOldListInstancesRequestToNew = (oldRequest: OldListInstancesRequest): NewListInstancesRequest => {
  const json = OldListInstancesRequest.toJSON(oldRequest) as any;
  return fromJson(ListInstancesRequestSchema, json);
};

// Convert proto-es ListInstancesRequest to old proto
export const convertNewListInstancesRequestToOld = (newRequest: NewListInstancesRequest): OldListInstancesRequest => {
  const json = toJson(ListInstancesRequestSchema, newRequest);
  return OldListInstancesRequest.fromJSON(json);
};

// Convert old proto ListInstancesResponse to proto-es
export const convertOldListInstancesResponseToNew = (oldResponse: OldListInstancesResponse): NewListInstancesResponse => {
  const json = OldListInstancesResponse.toJSON(oldResponse) as any;
  return fromJson(ListInstancesResponseSchema, json);
};

// Convert proto-es ListInstancesResponse to old proto
export const convertNewListInstancesResponseToOld = (newResponse: NewListInstancesResponse): OldListInstancesResponse => {
  const json = toJson(ListInstancesResponseSchema, newResponse);
  return OldListInstancesResponse.fromJSON(json);
};

// Convert old proto CreateInstanceRequest to proto-es
export const convertOldCreateInstanceRequestToNew = (oldRequest: OldCreateInstanceRequest): NewCreateInstanceRequest => {
  const json = OldCreateInstanceRequest.toJSON(oldRequest) as any;
  return fromJson(CreateInstanceRequestSchema, json);
};

// Convert proto-es CreateInstanceRequest to old proto
export const convertNewCreateInstanceRequestToOld = (newRequest: NewCreateInstanceRequest): OldCreateInstanceRequest => {
  const json = toJson(CreateInstanceRequestSchema, newRequest);
  return OldCreateInstanceRequest.fromJSON(json);
};

// Convert old proto DeleteInstanceRequest to proto-es
export const convertOldDeleteInstanceRequestToNew = (oldRequest: OldDeleteInstanceRequest): NewDeleteInstanceRequest => {
  const json = OldDeleteInstanceRequest.toJSON(oldRequest) as any;
  return fromJson(DeleteInstanceRequestSchema, json);
};

// Convert proto-es DeleteInstanceRequest to old proto
export const convertNewDeleteInstanceRequestToOld = (newRequest: NewDeleteInstanceRequest): OldDeleteInstanceRequest => {
  const json = toJson(DeleteInstanceRequestSchema, newRequest);
  return OldDeleteInstanceRequest.fromJSON(json);
};

// Convert old proto UndeleteInstanceRequest to proto-es
export const convertOldUndeleteInstanceRequestToNew = (oldRequest: OldUndeleteInstanceRequest): NewUndeleteInstanceRequest => {
  const json = OldUndeleteInstanceRequest.toJSON(oldRequest) as any;
  return fromJson(UndeleteInstanceRequestSchema, json);
};

// Convert proto-es UndeleteInstanceRequest to old proto
export const convertNewUndeleteInstanceRequestToOld = (newRequest: NewUndeleteInstanceRequest): OldUndeleteInstanceRequest => {
  const json = toJson(UndeleteInstanceRequestSchema, newRequest);
  return OldUndeleteInstanceRequest.fromJSON(json);
};

// Convert old proto SyncInstanceRequest to proto-es
export const convertOldSyncInstanceRequestToNew = (oldRequest: OldSyncInstanceRequest): NewSyncInstanceRequest => {
  const json = OldSyncInstanceRequest.toJSON(oldRequest) as any;
  return fromJson(SyncInstanceRequestSchema, json);
};

// Convert proto-es SyncInstanceRequest to old proto
export const convertNewSyncInstanceRequestToOld = (newRequest: NewSyncInstanceRequest): OldSyncInstanceRequest => {
  const json = toJson(SyncInstanceRequestSchema, newRequest);
  return OldSyncInstanceRequest.fromJSON(json);
};

// Convert old proto BatchSyncInstancesRequest to proto-es
export const convertOldBatchSyncInstancesRequestToNew = (oldRequest: OldBatchSyncInstancesRequest): NewBatchSyncInstancesRequest => {
  const json = OldBatchSyncInstancesRequest.toJSON(oldRequest) as any;
  return fromJson(BatchSyncInstancesRequestSchema, json);
};

// Convert proto-es BatchSyncInstancesRequest to old proto
export const convertNewBatchSyncInstancesRequestToOld = (newRequest: NewBatchSyncInstancesRequest): OldBatchSyncInstancesRequest => {
  const json = toJson(BatchSyncInstancesRequestSchema, newRequest);
  return OldBatchSyncInstancesRequest.fromJSON(json);
};

// ========== ENUM CONVERSIONS ==========

// Convert proto-es DataSourceType to old proto DataSourceType
export const convertDataSourceTypeToOld = (dataSourceType: NewDataSourceType): OldDataSourceType => {
  switch (dataSourceType) {
    case NewDataSourceType.DATA_SOURCE_UNSPECIFIED:
      return OldDataSourceType.DATA_SOURCE_UNSPECIFIED;
    case NewDataSourceType.ADMIN:
      return OldDataSourceType.ADMIN;
    case NewDataSourceType.READ_ONLY:
      return OldDataSourceType.READ_ONLY;
    default:
      return OldDataSourceType.DATA_SOURCE_UNSPECIFIED;
  }
};

// Convert old proto DataSourceType to proto-es DataSourceType
export const convertDataSourceTypeToNew = (dataSourceType: OldDataSourceType): NewDataSourceType => {
  switch (dataSourceType) {
    case OldDataSourceType.DATA_SOURCE_UNSPECIFIED:
      return NewDataSourceType.DATA_SOURCE_UNSPECIFIED;
    case OldDataSourceType.ADMIN:
      return NewDataSourceType.ADMIN;
    case OldDataSourceType.READ_ONLY:
      return NewDataSourceType.READ_ONLY;
    default:
      return NewDataSourceType.DATA_SOURCE_UNSPECIFIED;
  }
};

// Convert proto-es DataSource_AuthenticationType to old proto DataSource_AuthenticationType
export const convertDataSourceAuthenticationTypeToOld = (authType: NewDataSource_AuthenticationType): OldDataSource_AuthenticationType => {
  switch (authType) {
    case NewDataSource_AuthenticationType.AUTHENTICATION_UNSPECIFIED:
      return OldDataSource_AuthenticationType.AUTHENTICATION_UNSPECIFIED;
    case NewDataSource_AuthenticationType.PASSWORD:
      return OldDataSource_AuthenticationType.PASSWORD;
    case NewDataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM:
      return OldDataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM;
    case NewDataSource_AuthenticationType.AWS_RDS_IAM:
      return OldDataSource_AuthenticationType.AWS_RDS_IAM;
    case NewDataSource_AuthenticationType.AZURE_IAM:
      return OldDataSource_AuthenticationType.AZURE_IAM;
    default:
      return OldDataSource_AuthenticationType.AUTHENTICATION_UNSPECIFIED;
  }
};

// Convert old proto DataSource_AuthenticationType to proto-es DataSource_AuthenticationType
export const convertDataSourceAuthenticationTypeToNew = (authType: OldDataSource_AuthenticationType): NewDataSource_AuthenticationType => {
  switch (authType) {
    case OldDataSource_AuthenticationType.AUTHENTICATION_UNSPECIFIED:
      return NewDataSource_AuthenticationType.AUTHENTICATION_UNSPECIFIED;
    case OldDataSource_AuthenticationType.PASSWORD:
      return NewDataSource_AuthenticationType.PASSWORD;
    case OldDataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM:
      return NewDataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM;
    case OldDataSource_AuthenticationType.AWS_RDS_IAM:
      return NewDataSource_AuthenticationType.AWS_RDS_IAM;
    case OldDataSource_AuthenticationType.AZURE_IAM:
      return NewDataSource_AuthenticationType.AZURE_IAM;
    default:
      return NewDataSource_AuthenticationType.AUTHENTICATION_UNSPECIFIED;
  }
};

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

// ========== COMPOSED INSTANCE ADAPTERS ==========

import type { ComposedInstance as NewComposedInstance } from "@/types/v1/instance";

export const adaptComposedInstance = {
  fromLegacy: (legacy: any): NewComposedInstance => {
    // Since ComposedInstance is already using proto-es types, just return it
    return legacy as NewComposedInstance;
  },
  toLegacy: (current: NewComposedInstance): any => {
    // Since ComposedInstance is already using proto-es types, just return it
    return current;
  }
};