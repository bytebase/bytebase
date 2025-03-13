<template>
  <AdvancedSearch
    class="flex-1"
    :params="params"
    :scope-options="scopeOptions"
    @update:params="$emit('update:params', $event)"
  />
</template>

<script lang="tsx" setup>
import { flatten, uniqBy } from "lodash-es";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import AdvancedSearch from "@/components/AdvancedSearch";
import type { ScopeOption } from "@/components/AdvancedSearch/types";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import { type SearchParams } from "@/utils";
import { useRolloutDetailContext } from "../../context";
import { databaseForTask } from "../../utils";

defineProps<{
  params: SearchParams;
}>();

defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

const { t } = useI18n();
const { rollout, project } = useRolloutDetailContext();

const databasesFromTasks = computed(() =>
  uniqBy(
    flatten(rollout.value.stages.map((stage) => stage.tasks)).map((task) =>
      databaseForTask(project.value, task)
    ),
    (database) => database.name
  )
);

// fullScopeOptions provides full search scopes and options.
const scopeOptions = computed((): ScopeOption[] => {
  const scopes: ScopeOption[] = [
    {
      id: "stage",
      title: t("common.stage"),
      options: rollout.value.stages.map((stage) => {
        return {
          value: stage.name,
          keywords: [stage.name, stage.environment],
          render: () => stage.environment,
        };
      }),
    },
    {
      id: "environment",
      title: t("common.environment"),
      options: uniqBy(
        databasesFromTasks.value.map(
          (database) => database.effectiveEnvironmentEntity
        ),
        (env) => env.name
      ).map((environment) => {
        return {
          value: environment.name,
          keywords: [environment.name, environment.title],
          render: () => environment.title,
        };
      }),
    },
    {
      id: "instance",
      title: t("common.instance"),
      options: uniqBy(
        databasesFromTasks.value.map((database) => database.instanceResource),
        (instance) => instance.name
      ).map((instanceResource) => {
        return {
          value: instanceResource.name,
          keywords: [instanceResource.name, instanceResource.title],
          render: () => instanceResource.title,
        };
      }),
    },
    {
      id: "database",
      title: t("common.database"),
      options: databasesFromTasks.value.map((database) => {
        return {
          value: database.name,
          keywords: [database.name, database.databaseName],
          render: () => database.databaseName,
        };
      }),
    },
    {
      id: "status",
      title: t("common.status"),
      options: [
        Task_Status.NOT_STARTED,
        Task_Status.PENDING,
        Task_Status.RUNNING,
        Task_Status.DONE,
        Task_Status.FAILED,
        Task_Status.CANCELED,
        Task_Status.SKIPPED,
      ].map((status) => {
        let statusTitle;
        switch (status) {
          case Task_Status.NOT_STARTED:
            statusTitle = t("task.status.not-started");
            break;
          case Task_Status.PENDING:
            statusTitle = t("task.status.pending");
            break;
          case Task_Status.RUNNING:
            statusTitle = t("task.status.running");
            break;
          case Task_Status.DONE:
            statusTitle = t("task.status.done");
            break;
          case Task_Status.FAILED:
            statusTitle = t("task.status.failed");
            break;
          case Task_Status.CANCELED:
            statusTitle = t("task.status.canceled");
            break;
          case Task_Status.SKIPPED:
            statusTitle = t("task.status.skipped");
            break;
          default:
            statusTitle = status;
        }
        return {
          value: status,
          keywords: [status],
          render: () => statusTitle,
        };
      }),
    },
  ];
  return scopes;
});
</script>
