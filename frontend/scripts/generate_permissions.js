#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const yaml = require('js-yaml');

const YAML_FILE = path.join(__dirname, '../src/types/iam/permission.yaml');
const OUTPUT_FILE = path.join(__dirname, '../src/types/iam/permission-generated.ts');

// Read and parse YAML file
const yamlContent = fs.readFileSync(YAML_FILE, 'utf8');
const data = yaml.load(yamlContent);

if (!data.permissions || !Array.isArray(data.permissions)) {
  console.error('Error: permission.yaml must contain a "permissions" array');
  process.exit(1);
}

const permissions = data.permissions;

// Generate TypeScript content
const tsContent = `// This file is auto-generated from permission.yaml. DO NOT EDIT manually.
// Run 'pnpm run generate:permissions' or 'sh scripts/copy_config_files.sh' to regenerate.

export type Permission =
${permissions.map(p => `  | "${p}"`).join('\n')};
`;

// Write the generated TypeScript file
fs.writeFileSync(OUTPUT_FILE, tsContent, 'utf8');

console.log(`✅ Generated ${OUTPUT_FILE} with ${permissions.length} permissions`);
