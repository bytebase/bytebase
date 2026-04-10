import { cleanupEnvFile } from "./env";
import { stopServer } from "./mode-start-new-bytebase";

async function globalTeardown() {
  stopServer();
  cleanupEnvFile();
}

export default globalTeardown;
