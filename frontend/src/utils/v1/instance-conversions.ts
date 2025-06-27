import { fromJson, toJson } from "@bufbuild/protobuf";
import type { 
  Instance as OldInstance,
  DataSource as OldDataSource,
  UpdateInstanceRequest as OldUpdateInstanceRequest
} from "@/types/proto/v1/instance_service";
import { 
  Instance as OldInstanceProto,
  DataSource as OldDataSourceProto,
  UpdateInstanceRequest as OldUpdateInstanceRequestProto
} from "@/types/proto/v1/instance_service";
import type { 
  Instance as NewInstance,
  DataSource as NewDataSource,
  UpdateInstanceRequest as NewUpdateInstanceRequest
} from "@/types/proto-es/v1/instance_service_pb";
import { 
  InstanceSchema,
  DataSourceSchema,
  UpdateInstanceRequestSchema
} from "@/types/proto-es/v1/instance_service_pb";

// Convert old proto to proto-es
export const convertOldInstanceToNew = (oldInstance: OldInstance): NewInstance => {
  // Use toJSON to convert old proto to JSON, then fromJson to convert to proto-es
  const json = OldInstanceProto.toJSON(oldInstance) as any; // Type assertion needed due to proto type incompatibility
  
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
  return OldInstanceProto.fromJSON(json);
};

// Convert old proto DataSource to proto-es
export const convertOldDataSourceToNew = (oldDataSource: OldDataSource): NewDataSource => {
  const json = OldDataSourceProto.toJSON(oldDataSource) as any;
  return fromJson(DataSourceSchema, json);
};

// Convert proto-es DataSource to old proto
export const convertNewDataSourceToOld = (newDataSource: NewDataSource): OldDataSource => {
  const json = toJson(DataSourceSchema, newDataSource);
  return OldDataSourceProto.fromJSON(json);
};

// Convert old proto UpdateInstanceRequest to proto-es
export const convertOldUpdateInstanceRequestToNew = (oldRequest: OldUpdateInstanceRequest): NewUpdateInstanceRequest => {
  const json = OldUpdateInstanceRequestProto.toJSON(oldRequest) as any;
  return fromJson(UpdateInstanceRequestSchema, json);
};