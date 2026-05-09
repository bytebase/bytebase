import { describe, expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { getPlanOptionVisibility } from "./options";

const databaseWithEngine = (engine: Engine, engineVersion = "") =>
  ({
    instanceResource: {
      engine,
      engineVersion,
    },
  }) as Database;

describe("getPlanOptionVisibility", () => {
  test("shows MySQL sheet-backed change options", () => {
    expect(
      getPlanOptionVisibility({
        databases: [databaseWithEngine(Engine.MYSQL, "8.0.0")],
        isChangeDatabaseConfig: true,
        isSheetBasedDatabaseChange: true,
      })
    ).toMatchObject({
      shouldShow: true,
      showGhost: true,
      showIsolationLevel: true,
      showPreBackup: true,
      showTransactionMode: true,
    });
  });

  test("shows pre-backup for MariaDB change database configs", () => {
    expect(
      getPlanOptionVisibility({
        databases: [databaseWithEngine(Engine.MARIADB, "10.11.0")],
        isChangeDatabaseConfig: true,
        isSheetBasedDatabaseChange: true,
      })
    ).toMatchObject({
      showPreBackup: true,
    });
  });

  test("hides sheet-only options for release-backed specs", () => {
    expect(
      getPlanOptionVisibility({
        databases: [databaseWithEngine(Engine.POSTGRES, "15.0")],
        isChangeDatabaseConfig: true,
        isSheetBasedDatabaseChange: false,
      })
    ).toMatchObject({
      shouldShow: true,
      showGhost: false,
      showInstanceRole: false,
      showPreBackup: true,
      showTransactionMode: false,
    });
  });

  test("hides all options until target databases are loaded", () => {
    expect(
      getPlanOptionVisibility({
        databases: [],
        isChangeDatabaseConfig: true,
        isSheetBasedDatabaseChange: true,
      })
    ).toMatchObject({
      shouldShow: false,
      showGhost: false,
      showInstanceRole: false,
      showIsolationLevel: false,
      showPreBackup: false,
      showTransactionMode: false,
    });
  });
});
