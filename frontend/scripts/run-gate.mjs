// The frontend gate, and exactly what CI runs: `pnpm test`.
//
// The four stages below are independent, so they run concurrently rather than
// in a serial `&&` chain. vitest saturates most of the box on its own while the
// other three are largely single-core, so overlapping them costs little and
// takes the gate from ~92s to ~70s.
//
// Each stage's output is buffered and printed in a fixed order once every stage
// has finished, so a failure reads the same as it did serially -- interleaved
// live output from four concurrent stages would not.
import { spawn } from "node:child_process";

const GUARDS = [
  "check-frontend-structure",
  "check-ui-guideline",
  "check-no-crypto-randomuuid",
  "check-react-i18n",
  "check-react-layering",
];

const BIG_HEAP = { NODE_OPTIONS: "--max_old_space_size=8000" };

// Ordered cheapest-first: how the results are reported, not how they are run.
const STAGES = [
  { name: "biome", argv: ["pnpm", "exec", "biome", "ci", "."] },
  {
    name: "guards",
    argv: [
      "sh",
      "-c",
      [...GUARDS.map((g) => `node scripts/${g}.mjs`), "node scripts/sort_i18n_keys.mjs --check"].join(" && "),
    ],
  },
  { name: "tsc", argv: ["pnpm", "exec", "tsc", "--build", "--force"], env: BIG_HEAP },
  {
    name: "bundle",
    argv: ["pnpm", "exec", "vite", "build", "--mode", "release", "--outDir=dist-check", "--emptyOutDir"],
    // The legacy browser pass doubles the build and proves nothing about
    // whether the app compiles; see the plugin comment in vite.config.ts.
    env: { ...BIG_HEAP, BB_SKIP_LEGACY: "1" },
  },
  { name: "vitest", argv: ["pnpm", "exec", "vitest", "run", "--passWithNoTests"] },
];

function run({ name, argv, env }) {
  return new Promise((resolve) => {
    const started = Date.now();
    const child = spawn(argv[0], argv.slice(1), {
      env: { ...process.env, ...env },
      stdio: ["ignore", "pipe", "pipe"],
    });
    const chunks = [];
    child.stdout.on("data", (c) => chunks.push(c));
    child.stderr.on("data", (c) => chunks.push(c));
    child.on("error", (error) => {
      chunks.push(Buffer.from(`failed to spawn ${argv[0]}: ${error.message}\n`));
      resolve({ name, code: 1, seconds: 0, output: Buffer.concat(chunks).toString() });
    });
    child.on("close", (code) => {
      const seconds = (Date.now() - started) / 1000;
      process.stderr.write(`${code === 0 ? "ok  " : "FAIL"} ${name} (${seconds.toFixed(1)}s)\n`);
      resolve({ name, code: code ?? 1, seconds, output: Buffer.concat(chunks).toString() });
    });
  });
}

// Every stage reads the generated sources copied from the backend, so this one
// runs first and on its own.
const prepare = await run({ name: "prepare", argv: ["pnpm", "run", "prepare"] });
if (prepare.code !== 0) {
  process.stdout.write(prepare.output);
  process.exit(prepare.code);
}

const results = await Promise.all(STAGES.map(run));

for (const { name, output } of results) {
  process.stdout.write(`\n${"=".repeat(60)}\n${name}\n${"=".repeat(60)}\n${output}`);
}

// Each stage already reported itself as it finished, so only restate things
// when something failed and the list is worth having next to the exit.
const failed = results.filter((r) => r.code !== 0);
if (failed.length > 0) {
  process.stdout.write(`\n${failed.length} stage(s) failed: ${failed.map((r) => r.name).join(", ")}\n`);
  process.exit(1);
}
