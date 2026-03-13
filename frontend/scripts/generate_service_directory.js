#!/usr/bin/env node

// Generates a service directory from the OpenAPI index using an LLM.
// Run: pnpm --dir frontend run generate:service-directory
//
// Requires ANTHROPIC_API_KEY environment variable.
// Uses Claude to group services by feature area with user-friendly descriptions.

const fs = require("fs");
const path = require("path");

const INDEX_FILE = path.join(
  __dirname,
  "../src/plugins/agent/logic/tools/gen/openapi-index.ts"
);
const OUTPUT_FILE = path.join(
  __dirname,
  "../src/plugins/agent/logic/tools/gen/service-directory.ts"
);

// Parse the generated index to extract service -> endpoint summaries.
function extractServices() {
  const content = fs.readFileSync(INDEX_FILE, "utf8");

  // Extract the endpoints array via regex (avoid importing TS)
  const match = content.match(
    /export const endpoints: EndpointInfo\[\] = (\[[\s\S]*?\]);/
  );
  if (!match) {
    throw new Error("Could not parse endpoints from openapi-index.ts");
  }

  const endpoints = JSON.parse(match[1]);
  const services = new Map();

  for (const ep of endpoints) {
    if (!services.has(ep.service)) {
      services.set(ep.service, []);
    }
    services.get(ep.service).push({
      method: ep.method,
      summary: ep.summary,
      description: ep.description,
    });
  }

  return services;
}

async function generateWithLLM(services) {
  const apiKey = process.env.ANTHROPIC_API_KEY;
  if (!apiKey) {
    console.error(
      "Error: ANTHROPIC_API_KEY environment variable is required.\n" +
        "Set it and retry: ANTHROPIC_API_KEY=sk-... pnpm --dir frontend run generate:service-directory"
    );
    process.exit(1);
  }

  // Build service summary for the prompt
  const serviceSummary = [];
  for (const [name, endpoints] of services) {
    const methods = endpoints.map((e) => `${e.method}: ${e.summary || e.description}`).join("\n    ");
    serviceSummary.push(`  ${name} (${endpoints.length} endpoints):\n    ${methods}`);
  }

  const prompt = `You are generating a compact service directory for an AI agent embedded in the Bytebase database management console. The agent uses this directory to know which API service to call for a given task.

Here are all the API services and their endpoints:

${serviceSummary.join("\n\n")}

Generate a service directory that:
1. Groups services by feature area (e.g. "Database Management", "SQL & Queries", "Change Management", "Access & Identity", etc.)
2. For each service, write a brief description of what it covers using user-facing feature names (not just API method names)
3. Include common synonyms users might use (e.g. "semantic types" for DatabaseCatalogService, "SSO/IdP" for IdentityProviderService)
4. Mark internal/system services as "(system use)" so the agent knows not to suggest them to users
5. Keep each service description to one line
6. Format as a flat text block, no markdown headers

Format exactly like this example:
Database Management:
- DatabaseService: CRUD databases, list schemas/tables/columns, metadata, secrets, changelogs, slow queries, backups
- DatabaseCatalogService: semantic types, column masking config, data classification

Output ONLY the directory text, nothing else. No preamble, no explanation.`;

  const response = await fetch("https://api.anthropic.com/v1/messages", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "x-api-key": apiKey,
      "anthropic-version": "2023-06-01",
    },
    body: JSON.stringify({
      model: "claude-sonnet-4-20250514",
      max_tokens: 2000,
      messages: [{ role: "user", content: prompt }],
    }),
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`Anthropic API error ${response.status}: ${text}`);
  }

  const data = await response.json();
  return data.content[0].text.trim();
}

function writeOutput(directory) {
  const content = `// LLM-generated service directory. Regenerate with:
//   pnpm --dir frontend run generate:service-directory
// See frontend/src/plugins/agent/AGENT.md for maintenance guide.

export const serviceDirectory = \`API Directory — use search_api(service="...") to browse endpoints, search_api(operationId="...") for schemas.

${directory}
\`;
`;

  fs.writeFileSync(OUTPUT_FILE, content, "utf8");
  console.log(`Generated ${OUTPUT_FILE}`);
}

async function main() {
  console.log("Reading OpenAPI index...");
  const services = extractServices();
  console.log(`Found ${services.size} services`);

  console.log("Generating directory with LLM...");
  const directory = await generateWithLLM(services);

  writeOutput(directory);
  console.log("Done.");
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
