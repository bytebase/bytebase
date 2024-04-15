import { TablePartitionMetadata_Type } from "@/types/proto/v1/database_service";

export const PartitionTypesSupportSubPartition: readonly TablePartitionMetadata_Type[] =
  [
    TablePartitionMetadata_Type.RANGE,
    TablePartitionMetadata_Type.RANGE_COLUMNS,
    TablePartitionMetadata_Type.LIST,
    TablePartitionMetadata_Type.LIST_COLUMNS,
  ] as const;
