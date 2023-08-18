<template>
  <BBTable
    :column-list="columnList"
    :section-data-source="anomalySectionList"
    :compact-section="compactSection"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
  >
    <template #header>
      <BBTableHeaderCell
        :left-padding="4"
        class="w-4"
        :title="columnList[0].title"
      />
      <BBTableHeaderCell :title="columnList[1].title" />
      <BBTableHeaderCell :title="columnList[2].title" />
      <BBTableHeaderCell :title="columnList[3].title" />
      <BBTableHeaderCell :title="columnList[4].title" />
    </template>
    <template #body="{ rowData: anomaly }">
      <BBTableCell :left-padding="4">
        <heroicons-outline:information-circle
          v-if="anomaly.severity == Anomaly_AnomalySeverity.MEDIUM"
          class="w-6 h-6 text-info"
        />
        <heroicons-outline:exclamation
          v-else-if="anomaly.severity == Anomaly_AnomalySeverity.HIGH"
          class="w-6 h-6 text-warning"
        />
        <heroicons-outline:exclamation-circle
          v-else-if="anomaly.severity == Anomaly_AnomalySeverity.CRITICAL"
          class="w-6 h-6 text-error"
        />
      </BBTableCell>
      <BBTableCell>
        {{ typeName(anomaly.type) }}
      </BBTableCell>
      <BBTableCell>
        {{ detail(anomaly) }}
        <span class="normal-link" @click.prevent="action(anomaly).onClick">
          {{ action(anomaly).title }}
        </span>
      </BBTableCell>
      <BBTableCell>
        {{ humanizeTs((anomaly.updateTime?.getTime() ?? 0) / 1000) }}
      </BBTableCell>
      <BBTableCell>
        {{ humanizeTs((anomaly.createTime?.getTime() ?? 0) / 1000) }}
      </BBTableCell>
    </template>
  </BBTable>
  <BBModal
    v-if="schemaDriftDetail"
    class="!max-w-[calc(100%-40px)] overflow-auto"
    :title="`'${schemaDriftDetail.database.databaseName}' schema drift - ${schemaDriftDetail.payload?.recordVersion} vs Actual`"
    @close="dismissModal"
  >
    <div class="space-y-4">
      <code-diff
        class="w-full"
        :old-string="schemaDriftDetail.payload?.expectedSchema"
        :new-string="schemaDriftDetail.payload?.actualSchema"
        :file-name="`${schemaDriftDetail.payload?.recordVersion} (left) vs Actual (right)`"
        output-format="side-by-side"
      />
      <div class="flex justify-end px-4">
        <button type="button" class="btn-primary" @click.prevent="dismissModal">
          {{ $t("common.close") }}
        </button>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { CodeDiff } from "v-code-diff";
import { computed, PropType, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useDatabaseV1Store, useInstanceV1Store } from "@/store";
import { useEnvironmentV1Store } from "@/store";
import {
  Anomaly,
  Anomaly_AnomalyType,
  Anomaly_AnomalySeverity,
} from "@/types/proto/v1/anomaly_service";
import {
  databaseV1Slug,
  instanceV1Slug,
  humanizeTs,
  extractDatabaseResourceName,
} from "@/utils";
import { BBTableSectionDataSource } from "../bbkit/types";
import { UNKNOWN_ENVIRONMENT_NAME } from "../types";

type Action = {
  onClick: () => void;
  title: string;
};

interface LocalState {
  selectedAnomaly?: Anomaly;
}

defineProps({
  anomalySectionList: {
    required: true,
    type: Object as PropType<BBTableSectionDataSource<Anomaly>[]>,
  },
  compactSection: {
    default: true,
    type: Boolean,
  },
});

const router = useRouter();
const { t } = useI18n();

const state = reactive<LocalState>({
});

const columnList = computed(() => [
  {
    title: "",
  },
  {
    title: t("common.type"),
  },
  {
    title: t("common.detail"),
  },
  {
    title: t("anomaly.last-seen"),
  },
  {
    title: t("anomaly.first-seen"),
  },
]);

const typeName = (type: Anomaly_AnomalyType): string => {
  switch (type) {
    case Anomaly_AnomalyType.INSTANCE_CONNECTION:
      return t("anomaly.types.connection-failure");
    case Anomaly_AnomalyType.MIGRATION_SCHEMA:
      return t("anomaly.types.missing-migration-schema");
    case Anomaly_AnomalyType.DATABASE_BACKUP_POLICY_VIOLATION:
      return t("anomaly.types.backup-enforcement-violation");
    case Anomaly_AnomalyType.DATABASE_BACKUP_MISSING:
      return t("anomaly.types.missing-backup");
    case Anomaly_AnomalyType.DATABASE_CONNECTION:
      return t("anomaly.types.connection-failure");
    case Anomaly_AnomalyType.DATABASE_SCHEMA_DRIFT:
      return t("anomaly.types.schema-drift");
    default:
      return "";
  }
};

const detail = (anomaly: Anomaly): string => {
  switch (anomaly.type) {
    case Anomaly_AnomalyType.INSTANCE_CONNECTION: {
      return anomaly.instanceConnectionDetail?.detail ?? "";
    }
    case Anomaly_AnomalyType.MIGRATION_SCHEMA:
      return "Please create migration schema on the instance first.";
    case Anomaly_AnomalyType.DATABASE_BACKUP_POLICY_VIOLATION: {
      const environment = useEnvironmentV1Store().getEnvironmentByName(
        anomaly.databaseBackupPolicyViolationDetail?.parent ??
          UNKNOWN_ENVIRONMENT_NAME
      );
      if (!environment) {
        return "";
      }
      return `'${environment.title}' environment requires ${anomaly.databaseBackupPolicyViolationDetail?.expectedSchedule} auto-backup.`;
    }
    case Anomaly_AnomalyType.DATABASE_BACKUP_MISSING: {
      const payload = anomaly.databaseBackupMissingDetail;
      const missingSentence = `Missing ${payload?.expectedSchedule} backup, `;
      return (
        missingSentence +
        (payload?.latestBackupTime
          ? `last successful backup taken on ${humanizeTs(
              (payload?.latestBackupTime.getTime() ?? 0) / 1000
            )}.`
          : "no successful backup taken.")
      );
    }
    case Anomaly_AnomalyType.DATABASE_CONNECTION: {
      return anomaly.databaseConnectionDetail?.detail ?? "";
    }
    case Anomaly_AnomalyType.DATABASE_SCHEMA_DRIFT: {
      return `Recorded latest schema version ${anomaly.databaseSchemaDriftDetail?.recordVersion} is different from the actual schema.`;
    }
    default:
      return "";
  }
};

const action = (anomaly: Anomaly): Action => {
  switch (anomaly.type) {
    case Anomaly_AnomalyType.INSTANCE_CONNECTION: {
      const instance = useInstanceV1Store().getInstanceByName(anomaly.resource);
      return {
        onClick: () => {
          router.push({
            name: "workspace.instance.detail",
            params: {
              instanceSlug: instanceV1Slug(instance),
            },
          });
        },
        title: t("anomaly.action.check-instance"),
      };
    }
    case Anomaly_AnomalyType.MIGRATION_SCHEMA: {
      const instance = useInstanceV1Store().getInstanceByName(anomaly.resource);
      return {
        onClick: () => {
          router.push({
            name: "workspace.instance.detail",
            params: {
              instanceSlug: instanceV1Slug(instance),
            },
          });
        },
        title: t("anomaly.action.check-instance"),
      };
    }
    case Anomaly_AnomalyType.DATABASE_BACKUP_POLICY_VIOLATION: {
      const database = useDatabaseV1Store().getDatabaseByName(anomaly.resource);
      return {
        onClick: () => {
          router.push({
            name: "workspace.database.detail",
            params: {
              databaseSlug: databaseV1Slug(database),
            },
            hash: "#backup-and-restore",
          });
        },
        title: t("anomaly.action.configure-backup"),
      };
    }
    case Anomaly_AnomalyType.DATABASE_BACKUP_MISSING: {
      const database = useDatabaseV1Store().getDatabaseByName(anomaly.resource);
      return {
        onClick: () => {
          router.push({
            name: "workspace.database.detail",
            params: {
              databaseSlug: databaseV1Slug(database),
            },
            hash: "#backup-and-restore",
          });
        },
        title: t("anomaly.action.view-backup"),
      };
    }
    case Anomaly_AnomalyType.DATABASE_CONNECTION: {
      const instance = useInstanceV1Store().getInstanceByName(
        `instances/${extractDatabaseResourceName(anomaly.resource).instance}`
      );
      return {
        onClick: () => {
          router.push({
            name: "workspace.instance.detail",
            params: {
              instanceSlug: instanceV1Slug(instance),
            },
          });
        },
        title: t("anomaly.action.check-instance"),
      };
    }
    case Anomaly_AnomalyType.DATABASE_SCHEMA_DRIFT:
      return {
        onClick: () => {
          state.selectedAnomaly = anomaly;
          useDatabaseV1Store().getOrFetchDatabaseByName(anomaly.resource);
        },
        title: t("anomaly.action.view-diff"),
      };
    default:
      return {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        onClick: () => {},
        title: "",
      };
  }
};

const schemaDriftDetail = computed(() => {
  if (state.selectedAnomaly) {
    const anomaly = state.selectedAnomaly;
    const database = useDatabaseV1Store().getDatabaseByName(anomaly.resource);
    return {
      anomaly,
      payload: anomaly.databaseSchemaDriftDetail,
      database,
    };
  }
  return undefined;
});

const dismissModal = () => {
  state.selectedAnomaly = undefined;
};
</script>
