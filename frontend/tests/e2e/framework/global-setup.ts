import { saveTestEnv } from "./env";
import { cleanupOrphans, startServer, stopServer } from "./mode-start-new-bytebase";

async function globalSetup() {
  cleanupOrphans();

  let server: Awaited<ReturnType<typeof startServer>>;
  try {
    server = await startServer();
  } catch (err) {
    // startServer spawns the server process and creates its temp data dir
    // before it can fail (e.g. a /healthz timeout when the boot is slow).
    // Playwright does NOT run globalTeardown when globalSetup throws, so tear
    // the half-started server down here — otherwise a failed boot orphans a
    // process group and temp dir that starve every subsequent run's boot.
    stopServer();
    throw err;
  }

  const { baseURL, adminEmail, adminPassword } = server;
  saveTestEnv({
    baseURL, adminEmail, adminPassword,
    project: "", instance: "", instanceId: "", database: "", databaseId: "",
  });
}

export default globalSetup;
