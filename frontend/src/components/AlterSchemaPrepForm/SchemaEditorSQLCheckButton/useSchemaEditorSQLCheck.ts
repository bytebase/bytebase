import { cloneDeep, isEqual } from "lodash-es";
import { Ref, computed } from "vue";
import {
  mergeSchemaEditToMetadata,
  validateDatabaseMetadata,
} from "@/components/SchemaEditorV1/utils";
import { schemaDesignServiceClient } from "@/grpcweb";
import { t } from "@/plugins/i18n";
import { useDBSchemaV1Store, useSchemaEditorV1Store } from "@/store";
import { ComposedDatabase } from "@/types";

export const useSchemaEditorSQLCheck = (params: {
  selectedTab: Ref<"raw-sql" | "schema-editor">;
  databaseList: Ref<ComposedDatabase[]>;
  editStatement: Ref<string>;
}) => {
  const { selectedTab, databaseList, editStatement } = params;
  const schemaEditorV1Store = useSchemaEditorV1Store();
  const dbSchemaV1Store = useDBSchemaV1Store();

  const show = computed(() => {
    // SQL Check is highly related to the databases' environments.
    // By now we cannot handle mixed environments correctly.
    // so we just support SQL Check when editing single database's schema.
    return databaseList.value.length === 1;
  });

  const database = computed(() => {
    return databaseList.value[0];
  });

  const watchKey = computed(() => {
    if (selectedTab.value === "raw-sql") {
      return editStatement.value;
    } else {
      const databaseSchema = schemaEditorV1Store.resourceMap["database"].get(
        database.value.name
      );
      return JSON.stringify(databaseSchema);
    }
  });

  const fetchDatabaseEditStatement = async (): Promise<{
    errors: string[];
    statement: string;
  }> => {
    const db = database.value;
    const databaseSchema = schemaEditorV1Store.resourceMap["database"].get(
      db.name
    );
    if (!databaseSchema) {
      return { errors: [], statement: "" };
    }
    if (isEqual(databaseSchema.schemaList, databaseSchema.originSchemaList)) {
      return {
        errors: [t("schema-editor.nothing-changed")],
        statement: "",
      };
    }
    const metadata = await dbSchemaV1Store.getOrFetchDatabaseMetadata({
      database: db.name,
      skipCache: false,
      silent: true,
    });
    const mergedMetadata = mergeSchemaEditToMetadata(
      databaseSchema.schemaList,
      cloneDeep(metadata)
    );
    if (isEqual(metadata, mergedMetadata)) {
      return {
        errors: [t("schema-editor.nothing-changed")],
        statement: "",
      };
    }
    const validationMessages = validateDatabaseMetadata(mergedMetadata);
    if (validationMessages.length > 0) {
      return {
        errors: validationMessages,
        statement: "",
      };
    }

    try {
      const { diff } = await schemaDesignServiceClient.diffMetadata(
        {
          sourceMetadata: metadata,
          targetMetadata: mergedMetadata,
          engine: db.instanceEntity.engine,
        },
        {
          silent: true,
        }
      );
      if (diff.length === 0) {
        return {
          errors: [t("schema-editor.nothing-changed")],
          statement: "",
        };
      }
      return {
        errors: [],
        statement: diff,
      };
    } catch {
      return {
        errors: [t("schema-editor.message.invalid-schema")],
        statement: "",
      };
    }
  };

  const getStatement = async () => {
    if (selectedTab.value === "raw-sql") {
      const statement = editStatement.value;
      if (statement.length === 0) {
        return {
          errors: [t("schema-editor.nothing-changed")],
          statement: "",
        };
      }
      return {
        errors: [],
        statement,
      };
    } else {
      return fetchDatabaseEditStatement();
    }
  };

  return { show, database, watchKey, getStatement };
};
