import { isEqual } from "lodash-es";

import { createContextValues } from "@connectrpc/connect";
import { sqlServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";

import { t } from "@/plugins/i18n";
import { useSettingV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import type { DatabaseCatalog } from "@/types/proto-es/v1/database_catalog_service_pb";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import { extractGrpcErrorMessage } from "@/utils/grpcweb";
import { validateDatabaseMetadata } from "./utils";
import { create } from "@bufbuild/protobuf";
import { DiffMetadataRequestSchema } from "@/types/proto-es/v1/sql_service_pb";

export type GenerateDiffDDLResult = {
  statement: string;
  errors: string[];
};

export const generateDiffDDL = async ({
  database,
  sourceMetadata,
  targetMetadata,
  sourceCatalog,
  targetCatalog,
  allowEmptyDiffDDLWithConfigChange = true,
}: {
  database: ComposedDatabase;
  sourceMetadata: DatabaseMetadata;
  targetMetadata: DatabaseMetadata;
  sourceCatalog: DatabaseCatalog;
  targetCatalog: DatabaseCatalog;
  allowEmptyDiffDDLWithConfigChange?: boolean;
}): Promise<GenerateDiffDDLResult> => {
  const finish = (statement: string, errors: string[]) => {
    return {
      statement,
      errors,
    };
  };

  if (
    isEqual(sourceMetadata, targetMetadata) &&
    isEqual(sourceCatalog, targetCatalog)
  ) {
    return finish("", []);
  }

  const validationMessages = validateDatabaseMetadata(targetMetadata);
  if (validationMessages.length > 0) {
    return finish("", [
      t("schema-editor.message.invalid-schema"),
      ...validationMessages,
    ]);
  }
  try {
    const classificationConfig = useSettingV1Store().getProjectClassification(
      database.projectEntity.dataClassificationConfigId
    );

    const newRequest = create(DiffMetadataRequestSchema,{
      sourceMetadata: sourceMetadata,
      targetMetadata: targetMetadata,
      sourceCatalog,
      targetCatalog,
      engine: database.instanceResource.engine,
      classificationFromConfig:
        classificationConfig?.classificationFromConfig ?? false,
    });
    const diffResponse = await sqlServiceClientConnect.diffMetadata(newRequest, {
      contextValues: createContextValues().set(silentContextKey, true),
    });
    const { diff } = diffResponse;
    if (diff.length === 0) {
      if (
        !allowEmptyDiffDDLWithConfigChange &&
        !isEqual(sourceCatalog, targetCatalog)
      ) {
        return finish("", [t("schema-editor.message.cannot-change-config")]);
      }
      return finish("", []);
    }
    return finish(diff, []);
  } catch (ex) {
    console.warn("[generateDiffDDL]", ex);
    return finish("", [extractGrpcErrorMessage(ex)]);
  }
};
