import { detectMode, cleanupEnvFile } from "./env";
import { stopServer } from "./mode-start-new-bytebase";

async function globalTeardown() {
  const mode = detectMode();
  if (mode === "new") {
    stopServer();
  }
  cleanupEnvFile();
}

export default globalTeardown;
