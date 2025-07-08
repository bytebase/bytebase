import { TablePartitionMetadata_Type } from "@/types/proto-es/v1/database_service_pb";

export const PartitionTypesSupportSubPartition: readonly TablePartitionMetadata_Type[] =
  [
    TablePartitionMetadata_Type.RANGE,
    TablePartitionMetadata_Type.RANGE_COLUMNS,
    TablePartitionMetadata_Type.LIST,
    TablePartitionMetadata_Type.LIST_COLUMNS,
  ] as const;
