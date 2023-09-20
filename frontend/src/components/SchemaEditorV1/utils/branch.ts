import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import {
  SchemaDesign,
  SchemaDesign_Type,
} from "@/types/proto/v1/schema_design_service";
import {
  BranchSchema,
  convertSchemaMetadataList,
} from "@/types/v1/schemaEditor";
import { rebuildEditableSchemas } from "./metadataV1";

export const convertBranchToBranchSchema = (
  branch: SchemaDesign
): BranchSchema => {
  let originalSchemas = [];
  // For personal branches, we use its parent branch's schema as the original schema in editing state.
  if (branch.type === SchemaDesign_Type.PERSONAL_DRAFT) {
    const parentBranch = useSchemaDesignStore().getSchemaDesignByName(
      branch.baselineSheetName
    );
    originalSchemas = convertSchemaMetadataList(
      parentBranch?.baselineSchemaMetadata?.schemas || []
    );
  } else {
    originalSchemas = convertSchemaMetadataList(
      branch.baselineSchemaMetadata?.schemas || []
    );
  }
  const editableSchemas = rebuildEditableSchemas(
    originalSchemas,
    branch.schemaMetadata?.schemas || []
  );

  return {
    branch,
    schemaList: editableSchemas,
    originSchemaList: originalSchemas,
  };
};
