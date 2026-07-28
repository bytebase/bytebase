import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import { engineNameV1 } from "@/utils";

// Max characters for schema text to stay within token limits.
// ~4 chars per token, leaving room for system prompt and response.
// Targeting ~10k tokens = ~40k chars for the schema portion.
const MAX_SCHEMA_CHARS = 40000;

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
    let currentLength = prompts.join("\n").length;

    for (const schema of schemas) {
      for (const table of schema.tables) {
        const tableNameParts = [table.name];
        if (schema.name && !schemaScoped) {
          tableNameParts.unshift(schema.name);
        }
        const columns = table.columns
          .map((column) => {
            if (column.comment) {
              return `${column.name} ${column.type} (${column.comment})`;
            } else {
              return `${column.name} ${column.type}`;
            }
          })
          .join(", ");
        const tableLine = `# ${tableNameParts.join(".")}(${columns})`;

        // Check if adding this table would exceed the limit
        if (currentLength + tableLine.length + 1 > MAX_SCHEMA_CHARS) {
          prompts.push("# ... (schema truncated due to size)");
          return prompts.join("\n");
        }

        prompts.push(tableLine);
        currentLength += tableLine.length + 1; // +1 for newline
      }
    }
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
