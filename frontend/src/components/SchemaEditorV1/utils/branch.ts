import { markRaw } from "vue";
import { Branch } from "@/types/proto/v1/branch_service";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import {
  BranchSchema,
  convertSchemaMetadataList,
} from "@/types/v1/schemaEditor";
import { rebuildEditableSchemas } from "./metadata";

export const convertBranchToBranchSchema = async (
  branch: Branch
): Promise<BranchSchema> => {
  const baselineMetadata =
    branch.baselineSchemaMetadata || DatabaseMetadata.fromPartial({});
  const originalSchemas = convertSchemaMetadataList(
    baselineMetadata.schemas || [],
    baselineMetadata.schemaConfigs || []
  );

  const editableSchemas = rebuildEditableSchemas(
    originalSchemas,
    branch.schemaMetadata?.schemas || [],
    branch.schemaMetadata?.schemaConfigs || []
  );

  return {
    branch: markRaw(branch),
    schemaList: editableSchemas,
    originSchemaList: markRaw(originalSchemas),
  };
};
