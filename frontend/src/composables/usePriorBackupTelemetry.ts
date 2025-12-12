import { head } from "lodash-es";
import {
  useActuatorV1Store,
  useCurrentUserV1,
  useDatabaseV1Store,
  useSettingV1Store,
} from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { isValidDatabaseName } from "@/types/v1/database";
import { isDev } from "@/utils";

interface PriorBackupTelemetryPayload {
  enabled: boolean;
  environment: string;
  engine: string;
  isExplicitlySet: boolean;
  projectSetting: {
    autoEnableBackup: boolean;
    skipBackupErrors: boolean;
  };
}

/**
 * Get engine from task target (database name).
 */
async function getEngineFromTaskTarget(target: string): Promise<Engine> {
  if (!isValidDatabaseName(target)) {
    return Engine.ENGINE_UNSPECIFIED;
  }
  try {
    const db = await useDatabaseV1Store().getOrFetchDatabaseByName(
      target,
      true /* silent */
    );
    return db.instanceResource.engine;
  } catch {
    return Engine.ENGINE_UNSPECIFIED;
  }
}

/**
 * Track prior backup telemetry when tasks are run.
 * Reports to hub.bytebase.com when tasks with prior backup enabled are executed.
 */
export async function trackPriorBackupOnTaskRun(
  tasks: Task[],
  plan: Plan,
  project: Project,
  environmentName: string
): Promise<void> {
  // Skip in dev mode
  if (isDev()) {
    return;
  }

  // Respect user's metric collection preference
  const settingStore = useSettingV1Store();
  if (settingStore.workspaceProfileSetting?.enableMetricCollection === false) {
    return;
  }

  try {
    const actuatorStore = useActuatorV1Store();
    const currentUser = useCurrentUserV1();

    const workspaceId = actuatorStore.info?.workspaceId;
    const email = currentUser.value.email;
    const version = actuatorStore.version;
    const commit = actuatorStore.gitCommitBE;

    if (!workspaceId || !email) {
      return;
    }

    // Find specs with prior backup enabled that correspond to the running tasks
    for (const spec of plan.specs) {
      if (spec.config?.case !== "changeDatabaseConfig") {
        continue;
      }

      const config = spec.config.value;
      if (!config.enablePriorBackup) {
        continue;
      }

      // Check if any of the running tasks belong to this spec by specId
      const matchingTasks = tasks.filter((task) => task.specId === spec.id);

      if (matchingTasks.length === 0) {
        continue;
      }

      // Get engine from first matching task
      const firstTask = head(matchingTasks);
      const engine = firstTask
        ? await getEngineFromTaskTarget(firstTask.target)
        : Engine.ENGINE_UNSPECIFIED;

      // Determine if user explicitly set this or used project default
      // If enablePriorBackup differs from project's autoEnableBackup, user explicitly changed it
      const isExplicitlySet =
        config.enablePriorBackup !== project.autoEnableBackup;

      const payload: PriorBackupTelemetryPayload = {
        enabled: true,
        environment: environmentName,
        engine: Engine[engine],
        isExplicitlySet,
        projectSetting: {
          autoEnableBackup: project.autoEnableBackup,
          skipBackupErrors: project.skipBackupErrors,
        },
      };

      // Send event to hub.bytebase.com
      await fetch("https://hub.bytebase.com/v1/events", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          workspaceId,
          email,
          version,
          commit,
          priorBackupEnabled: payload,
        }),
      });

      // Only report once per task run action (not per spec)
      break;
    }
  } catch {
    // Silently fail if tracking fails - don't block user action
  }
}
