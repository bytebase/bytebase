import { saveTestEnv } from "./env";
import { cleanupOrphans, startServer } from "./mode-start-new-bytebase";

async function globalSetup() {
  cleanupOrphans();
  const { baseURL, adminEmail, adminPassword } = await startServer();
  saveTestEnv({
    baseURL, adminEmail, adminPassword,
    project: "", instance: "", instanceId: "", database: "", databaseId: "",
  });
}

export default globalSetup;
