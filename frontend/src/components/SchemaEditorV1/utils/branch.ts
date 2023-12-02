import { markRaw } from "vue";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
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
    branch: markRaw(branch),
    schemaList: editableSchemas,
    originSchemaList: markRaw(originalSchemas),
  };
};

export const fetchBaselineMetadataOfBranch = async (
  branch: Branch
): Promise<DatabaseMetadata> => {
  // For personal branches, we use its parent branch's schema as the original schema in editing state.
  if (branch.parentBranch !== "") {
    const parentBranch = await useSchemaDesignStore().fetchSchemaDesignByName(
      branch.parentBranch,
      false /* !useCache */
    );
    return (
      parentBranch.baselineSchemaMetadata || DatabaseMetadata.fromPartial({})
    );
  }
  return branch.baselineSchemaMetadata || DatabaseMetadata.fromPartial({});
};
