import { isEqual } from "lodash-es";
import { sqlServiceClient } from "@/grpcweb";
import { t } from "@/plugins/i18n";
import { useSettingV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import type { DatabaseCatalog } from "@/types/proto/api/v1alpha/database_catalog_service";
import type { DatabaseMetadata } from "@/types/proto/api/v1alpha/database_service";
import { TinyTimer } from "@/utils";
import { extractGrpcErrorMessage } from "@/utils/grpcweb";
import { validateDatabaseMetadata } from "./utils";

export const _generateDiffDDLTimer = new TinyTimer<"generateDiffDDL">(
  "GenerateDiffDDL"
);

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
    _generateDiffDDLTimer.end("generateDiffDDL");
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

    const diffResponse = await sqlServiceClient.diffMetadata(
      {
        sourceMetadata,
        targetMetadata,
        sourceCatalog,
        targetCatalog,
        engine: database.instanceResource.engine,
        classificationFromConfig:
          classificationConfig?.classificationFromConfig ?? false,
      },
      {
        silent: true,
      }
    );
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
