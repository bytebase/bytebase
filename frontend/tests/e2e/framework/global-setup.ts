import { detectMode, saveTestEnv, getBaseURL } from "./env";
import { cleanupOrphans, startServer } from "./mode-start-new-bytebase";
import { verifyReachable } from "./mode-use-local-bytebase";

async function globalSetup() {
  const mode = detectMode();

  if (mode === "new") {
    cleanupOrphans();
    const { baseURL, adminEmail, adminPassword } = await startServer();
    saveTestEnv({
      baseURL, adminEmail, adminPassword, mode,
      project: "", instance: "", instanceId: "", database: "", databaseId: "",
    });
  } else {
    const baseURL = getBaseURL();
    await verifyReachable(baseURL);
    saveTestEnv({
      baseURL, mode, adminEmail: process.env.BYTEBASE_USER ?? "",
      adminPassword: process.env.BYTEBASE_PASS,
      project: "", instance: "", instanceId: "", database: "", databaseId: "",
    });
  }
}

export default globalSetup;
