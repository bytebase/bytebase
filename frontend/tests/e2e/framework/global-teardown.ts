import { cleanupEnvFile } from "./env";
import { stopServer } from "./mode-start-new-bytebase";

async function globalTeardown() {
  await stopServer();
  cleanupEnvFile();
}

export default globalTeardown;
