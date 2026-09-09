import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

// Several tests import generated sources that are gitignored and copied from
// the backend -- src/modules/agent/logic/tools/gen/openapi-index.ts and the
// permission schema behind src/types/iam/permission.ts. `pnpm test` and
// `pnpm dev` regenerate them, but a focused `pnpm vitest run <path>` after
// switching revisions would otherwise read a stale copy, or fail module
// resolution if it was never generated. Regenerating here keeps every vitest
// entry point self-sufficient without another package.json script.
export default function setup() {
  const cwd = fileURLToPath(new URL("./", import.meta.url));
  execFileSync("sh", ["./scripts/copy_config_files.sh"], { cwd, stdio: "inherit" });
  execFileSync("node", ["scripts/generate_openapi_index.js"], { cwd, stdio: "inherit" });
}
