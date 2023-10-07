import { postDatabaseEdit } from "@/components/AlterSchemaPrepForm/utils";
import { validateDatabaseMetadata } from "@/components/SchemaEditorV1/utils";
import { convertBranchToBranchSchema } from "@/components/SchemaEditorV1/utils/branch";
import { schemaDesignServiceClient } from "@/grpcweb";
import { t } from "@/plugins/i18n";
import { ComposedDatabase, DatabaseEdit } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import { BranchSchema } from "@/types/v1/schemaEditor";
import {
  checkHasSchemaChanges,
  diffSchema,
  mergeDiffResults,
} from "../schemaEditor/diffSchema";

const diffViaMetadata = async (
  branch: SchemaDesign,
  database: ComposedDatabase
) => {
  // const original = await useDBSchemaV1Store().getOrFetchDatabaseMetadata(
  //   database.name,
  //   false /* !skipCache */,
  //   true /* silent */
  // );
  // const mergedMetadata = mergeSchemaEditToMetadata(
  //   branchSchema.schemaList,
  //   cloneDeep(original)
  // );
  const sourceMetadata =
    branch.baselineSchemaMetadata ?? DatabaseMetadata.fromPartial({});
  const targetMetadata =
    branch.schemaMetadata ?? DatabaseMetadata.fromPartial({});
  const validationMessages = validateDatabaseMetadata(targetMetadata);
  if (validationMessages.length > 0) {
    return {
      errors: validationMessages,
      statement: "",
    };
  }
  try {
    const { diff } = await schemaDesignServiceClient.diffMetadata(
      {
        sourceMetadata,
        targetMetadata,
        engine: database.instanceEntity.engine,
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

const diffViaDatabaseEdit = async (
  branch: SchemaDesign,
  database: ComposedDatabase
) => {
  const branchSchema = convertBranchToBranchSchema(branch);

  const databaseEdit = calcDatabaseEditFromBranchSchema(branchSchema, database);

  const databaseEditResult = await postDatabaseEdit(
    databaseEdit,
    true /* silent */
  );
  if (databaseEditResult.validateResultList.length > 0) {
    return {
      errors: databaseEditResult.validateResultList.map((v) => v.message),
      statement: "",
    };
  }
  const { statement } = databaseEditResult;
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
};

const calcDatabaseEditFromBranchSchema = (
  branchSchema: BranchSchema,
  database: ComposedDatabase
) => {
  let databaseEdit: DatabaseEdit = {
    databaseId: Number(database.uid),
    createSchemaList: [],
    renameSchemaList: [],
    dropSchemaList: [],
    createTableList: [],
    alterTableList: [],
    renameTableList: [],
    dropTableList: [],
  };
  for (const schema of branchSchema.schemaList) {
    const originSchema = branchSchema.originSchemaList.find(
      (originSchema) => originSchema.id === schema.id
    );
    if (!originSchema) {
      continue;
    }

    const diffSchemaResult = diffSchema(database.name, originSchema, schema);
    if (checkHasSchemaChanges(diffSchemaResult)) {
      databaseEdit = {
        databaseId: Number(database.uid),
        ...mergeDiffResults([diffSchemaResult, databaseEdit]),
      };
    }
  }

  return databaseEdit;
};

export const generateDDLByBranchAndDatabase = async (
  branch: SchemaDesign,
  database: ComposedDatabase
) => {
  // Use `SchemaDesignService.DiffMetadata` for MySQL as we only support MySQL for now.
  if (database.instanceEntity.engine === Engine.MYSQL) {
    return await diffViaMetadata(branch, database);
  } else {
    // Use legacy `DatabaseEdit` for non-MySQL databases.
    return await diffViaDatabaseEdit(branch, database);
  }
};
