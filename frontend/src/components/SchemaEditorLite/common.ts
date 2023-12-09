import { isEqual } from "lodash-es";
import { branchServiceClient } from "@/grpcweb";
import { t } from "@/plugins/i18n";
import { ComposedDatabase } from "@/types";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { TinyTimer } from "@/utils";
import { extractGrpcErrorMessage } from "@/utils/grpcweb";
import { validateDatabaseMetadata } from "./utils";

export const _generateDiffDDLTimer = new TinyTimer<"generateDiffDDL">(
  "GenerateDiffDDL"
);

export type GenerateDiffDDLResult = {
  statement: string;
  errors: string[];
  fatal: boolean;
};

export const generateDiffDDL = async (
  database: ComposedDatabase,
  source: DatabaseMetadata,
  target: DatabaseMetadata
): Promise<GenerateDiffDDLResult> => {
  const finish = (statement: string, errors: string[], fatal: boolean) => {
    _generateDiffDDLTimer.end("generateDiffDDL");
    return {
      statement,
      errors,
      fatal,
    };
  };

  if (isEqual(source, target)) {
    return finish("", [t("schema-editor.nothing-changed")], true);
  }

  const validationMessages = validateDatabaseMetadata(target);
  if (validationMessages.length > 0) {
    return finish(
      "",
      [t("schema-editor.message.invalid-schema"), ...validationMessages],
      false
    );
  }
  try {
    const diffResponse = await branchServiceClient.diffMetadata(
      {
        sourceMetadata: source,
        targetMetadata: target,
        engine: database.instanceEntity.engine,
      },
      {
        silent: true,
      }
    );
    const { diff } = diffResponse;
    if (diff.length === 0) {
      if (!isEqual(source.schemaConfigs, target.schemaConfigs)) {
        return finish(
          "",
          [t("schema-editor.message.cannot-change-config")],
          true
        );
      }
      return finish("", [t("schema-editor.nothing-changed")], true);
    }
    return finish(diff, [], false);
  } catch (ex) {
    console.warn("[generateDiffDDL]", ex);
    return finish("", [extractGrpcErrorMessage(ex)], true);
  }
};
