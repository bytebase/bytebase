import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { isEqual } from "lodash-es";
import { sqlServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { t } from "@/plugins/i18n";
import type { ComposedDatabase } from "@/types";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import { DiffMetadataRequestSchema } from "@/types/proto-es/v1/sql_service_pb";
import { extractGrpcErrorMessage } from "@/utils/connect";
import { validateDatabaseMetadata } from "./utils";

export type GenerateDiffDDLResult = {
  statement: string;
  errors: string[];
};

export const generateDiffDDL = async ({
  database,
  sourceMetadata,
  targetMetadata,
}: {
  database: ComposedDatabase;
  sourceMetadata: DatabaseMetadata;
  targetMetadata: DatabaseMetadata;
}): Promise<GenerateDiffDDLResult> => {
  const finish = (statement: string, errors: string[]) => {
    return {
      statement,
      errors,
    };
  };

  if (isEqual(sourceMetadata, targetMetadata)) {
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
    const newRequest = create(DiffMetadataRequestSchema, {
      sourceMetadata: sourceMetadata,
      targetMetadata: targetMetadata,
      engine: database.instanceResource.engine,
    });
    const diffResponse = await sqlServiceClientConnect.diffMetadata(
      newRequest,
      {
        contextValues: createContextValues().set(silentContextKey, true),
      }
    );
    const { diff } = diffResponse;
    if (diff.length === 0) {
      return finish("", []);
    }
    return finish(diff, []);
  } catch (ex) {
    console.warn("[generateDiffDDL]", ex);
    return finish("", [extractGrpcErrorMessage(ex)]);
  }
};
