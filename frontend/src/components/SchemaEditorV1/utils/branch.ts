import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import {
  SchemaDesign,
  SchemaDesign_Type,
} from "@/types/proto/v1/schema_design_service";
import {
  BranchSchema,
  convertSchemaMetadataList,
} from "@/types/v1/schemaEditor";
import { rebuildEditableSchemas } from "./metadata";

export const convertBranchToBranchSchema = async (
  branch: SchemaDesign
): Promise<BranchSchema> => {
  const baselineMetadata = await fetchBaselineMetadataOfBranch(branch);
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
    branch,
    schemaList: editableSchemas,
    originSchemaList: originalSchemas,
  };
};

export const fetchBaselineMetadataOfBranch = async (
  branch: SchemaDesign
): Promise<DatabaseMetadata> => {
  // For personal branches, we use its parent branch's schema as the original schema in editing state.
  if (
    branch.type === SchemaDesign_Type.PERSONAL_DRAFT &&
    branch.baselineSheetName
  ) {
    const parentBranch = await useSchemaDesignStore().fetchSchemaDesignByName(
      branch.baselineSheetName,
      false /* !useCache */
    );
    return (
      parentBranch.baselineSchemaMetadata || DatabaseMetadata.fromPartial({})
    );
  }
  return branch.baselineSchemaMetadata || DatabaseMetadata.fromPartial({});
};
