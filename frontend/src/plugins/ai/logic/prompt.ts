import type { Engine } from "@/types/proto/v1/common";
import type { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { engineNameV1 } from "@/utils";

export const databaseMetadataToText = (
  databaseMetadata: DatabaseMetadata | undefined,
  engine?: Engine,
  schema?: string
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
    const schemas = schema
      ? databaseMetadata.schemas.filter((s) => s.name === schema)
      : databaseMetadata.schemas;

    const schemaScoped = !!schema;
    schemas.forEach((schema) => {
      schema.tables.forEach((table) => {
        const tableNameParts = [table.name];
        if (schema.name && !schemaScoped) {
          tableNameParts.unshift(schema.name);
        }
        const columns = table.columns
          .map((column) => {
            if (column.comment) {
              return `${column.name}(${column.comment})`;
            } else {
              return column.name;
            }
          })
          .join(", ");
        prompts.push(`# ${tableNameParts.join(".")}(${columns})`);
      });
    });
  }
  return prompts.join("\n");
};

export const declaration = (
  databaseMetadata?: DatabaseMetadata,
  engine?: Engine,
  schema?: string
) => {
  const prompts: string[] = [];
  if (engine) {
    prompts.push(`You are a ${engineNameV1(engine)} db and SQL expert.`);
  } else {
    prompts.push(`You are a db and SQL expert.`);
  }
  // prompts.push(`When asked for your name, you must respond with "Bytebase".`);
  prompts.push(`Your responses should be informative and terse.`);
  // prompts.push(
  //   "Set the language to the markdown SQL block. e.g, `SELECT * FROM table`."
  // );

  prompts.push(databaseMetadataToText(databaseMetadata, engine, schema));
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
  prompts.push("Explain the following SQL code：");
  prompts.push(wrapStatementMarkdown(statement, engine));
  return prompts.join("\n");
};

export const wrapStatementMarkdown = (statement: string, engine?: Engine) => {
  let openTag = "```";
  if (engine) {
    openTag += engineNameV1(engine).toLowerCase();
  } else {
    openTag += "SQL";
  }
  const closeTag = "```";
  return [openTag, statement, closeTag].join("\n");
};

export const dynamicSuggestions = (metadata: string, ignores?: Set<string>) => {
  const commands = [
    `You are an assistant who works as a Magic: The Suggestion card designer. Create cards that are in the following card schema and JSON format. OUTPUT MUST FOLLOW THIS CARD SCHEMA AND JSON FORMAT. DO NOT EXPLAIN THE CARD.`,
    `{"suggestion-1": "What is the average salary of employees in each department?", "suggestion-2": "What is the average salary of employees in each department?", "suggestion-3": "What is the average salary of employees in each department?"}`,
  ];
  const prompts = [
    metadata,
    "Create a suggestion card about interesting queries to try in this database.",
  ];
  if (ignores && ignores.size > 0) {
    prompts.push("queries below should be ignored");
    for (const sug of ignores.values()) {
      prompts.push(sug);
    }
  }
  return {
    command: commands.join("\n"),
    prompt: prompts.join("\n"),
  };
};
