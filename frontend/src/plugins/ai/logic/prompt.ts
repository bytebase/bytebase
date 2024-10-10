import type { Engine } from "@/types/proto/v1/common";
import type { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { engineNameV1 } from "@/utils";

export const databaseMetadataToText = (
  databaseMetadata: DatabaseMetadata | undefined,
  engine?: Engine
) => {
  const prompts: string[] = [];
  if (engine) {
    if (databaseMetadata) {
      prompts.push(
        `### ${engineNameV1(engine)} tables, with their properties:`
      );
    } else {
      prompts.push(`### ${engineNameV1(engine)} database`);
    }
  } else {
    if (databaseMetadata) {
      prompts.push(`### Giving a database`);
    }
  }
  if (databaseMetadata) {
    databaseMetadata.schemas.forEach((schema) => {
      schema.tables.forEach((table) => {
        const name = schema.name ? `${schema.name}.${table.name}` : table.name;
        const columns = table.columns
          .map((column) => {
            if (column.comment) {
              return `${column.name}(${column.comment})`;
            } else {
              return column.name;
            }
          })
          .join(", ");
        prompts.push(`# ${name}(${columns})`);
      });
    });
  }
  return prompts.join("\n");
};

export const declaration = (
  databaseMetadata?: DatabaseMetadata,
  engine?: Engine
) => {
  const prompts: string[] = [];
  if (engine) {
    prompts.push(`You are a ${engineNameV1(engine)} db and SQL expert.`);
  } else {
    prompts.push(`You are a db and SQL expert.`);
  }
  prompts.push(`When asked for your name, you must respond with "Bytebase".`);
  // prompts.push(`Your responses should be informative and terse.`);
  prompts.push(
    "Set the language to the markdown SQL block. e.g, `SELECT * FROM table`."
  );

  prompts.push(databaseMetadataToText(databaseMetadata, engine));
  prompts.push("Answer the following questions about this schema:");

  return prompts.join("\n");
};

export const findProblems = (statement: string, engine?: Engine) => {
  const prompts: string[] = [];
  prompts.push(
    "Find potential problems in the following SQL code. Explain and try to give the correct statement."
  );
  prompts.push(wrapStatementMarkdown(statement, engine));
  return prompts.join("\n");
};

export const explainCode = (statement: string, engine?: Engine) => {
  const prompts: string[] = [];
  prompts.push("Explain the following SQL code");
  prompts.push(wrapStatementMarkdown(statement, engine));
  return prompts.join("\n");
};

const wrapStatementMarkdown = (statement: string, engine?: Engine) => {
  let openTag = "```";
  if (engine) {
    openTag += engineNameV1(engine).toLowerCase();
  }
  const closeTag = "```";
  return [openTag, statement, closeTag].join("\n");
};
