<template>
  <BBTable
    :column-list="COLUMN_LIST"
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
        :title="COLUMN_LIST[0].title"
      />
      <BBTableHeaderCell :title="COLUMN_LIST[1].title" />
      <BBTableHeaderCell :title="COLUMN_LIST[2].title" />
      <BBTableHeaderCell :title="COLUMN_LIST[3].title" />
      <BBTableHeaderCell :title="COLUMN_LIST[4].title" />
    </template>
    <template #body="{ rowData: anomaly }">
      <BBTableCell :left-padding="4">
        <heroicons-outline:information-circle
          v-if="anomaly.severity == 'MEDIUM'"
          class="w-6 h-6 text-info"
        />
        <heroicons-outline:exclamation
          v-else-if="anomaly.severity == 'HIGH'"
          class="w-6 h-6 text-warning"
        />
        <heroicons-outline:exclamation-circle
          v-else-if="anomaly.severity == 'CRITICAL'"
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
        {{ humanizeTs(anomaly.updatedTs) }}
      </BBTableCell>
      <BBTableCell>
        {{ humanizeTs(anomaly.createdTs) }}
      </BBTableCell>
    </template>
  </BBTable>
  <BBModal
    v-if="state.showModal"
    :title="`'${state.selectedAnomaly.database.name}' schema drift - ${state.selectedAnomaly.payload.version} vs Actual`"
    @close="dismissModal"
  >
    <div class="space-y-4">
      <code-diff
        class="w-full"
        :old-string="state.selectedAnomaly.payload.expect"
        :new-string="state.selectedAnomaly.payload.actual"
        :file-name="`${state.selectedAnomaly.payload.version} (left) vs Actual (right)`"
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

<script lang="ts">
import { defineComponent, PropType, reactive } from "vue";
import { useRouter } from "vue-router";
import { CodeDiff } from "v-code-diff";
import { useI18n } from "vue-i18n";

import { BBTableColumn, BBTableSectionDataSource } from "../bbkit/types";
import {
  Anomaly,
  AnomalyDatabaseBackupMissingPayload,
  AnomalyDatabaseBackupPolicyViolationPayload,
  AnomalyDatabaseConnectionPayload,
  AnomalyDatabaseSchemaDriftPayload,
  AnomalyInstanceConnectionPayload,
  AnomalyType,
} from "../types";
import { databaseSlug, humanizeTs, instanceSlug } from "../utils";
import { useEnvironmentStore } from "@/store";

type Action = {
  onClick: () => void;
  title: string;
};

interface LocalState {
  showModal: boolean;
  selectedAnomaly?: Anomaly;
}

export default defineComponent({
  name: "AnomalyTable",
  components: { CodeDiff },
  props: {
    anomalySectionList: {
      required: true,
      type: Object as PropType<BBTableSectionDataSource<Anomaly>[]>,
    },
    compactSection: {
      default: true,
      type: Boolean,
    },
  },
  setup() {
    const router = useRouter();
    const { t } = useI18n();

    const state = reactive<LocalState>({
      showModal: false,
    });

    const COLUMN_LIST: BBTableColumn[] = reactive([
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

    const typeName = (type: AnomalyType): string => {
      switch (type) {
        case "bb.anomaly.instance.connection":
          return t("anomaly.types.connection-failure");
        case "bb.anomaly.instance.migration-schema":
          return t("anomaly.types.missing-migration-schema");
        case "bb.anomaly.database.backup.policy-violation":
          return t("anomaly.types.backup-enforcement-viloation");
        case "bb.anomaly.database.backup.missing":
          return t("anomaly.types.missing-backup");
        case "bb.anomaly.database.connection":
          return t("anomaly.types.connection-failure");
        case "bb.anomaly.database.schema.drift":
          return t("anomaly.types.schema-drift");
      }
    };

    const detail = (anomaly: Anomaly): string => {
      switch (anomaly.type) {
        case "bb.anomaly.instance.connection": {
          const payload = anomaly.payload as AnomalyInstanceConnectionPayload;
          return payload.detail;
        }
        case "bb.anomaly.instance.migration-schema":
          return "Please create migration schema on the instance first.";
        case "bb.anomaly.database.backup.policy-violation": {
          const environment = useEnvironmentStore().getEnvironmentById(
            anomaly.instance.environment.id
          );
          const payload =
            anomaly.payload as AnomalyDatabaseBackupPolicyViolationPayload;
          return `'${environment.name}' environment requires ${payload.expectedSchedule} auto-backup.`;
        }
        case "bb.anomaly.database.backup.missing": {
          const payload =
            anomaly.payload as AnomalyDatabaseBackupMissingPayload;
          const missingSentence = `Missing ${payload.expectedSchedule} backup, `;
          return (
            missingSentence +
            (payload.lastBackupTs
              ? `last successful backup taken on ${humanizeTs(
                  payload.lastBackupTs
                )}.`
              : "no successful backup taken.")
          );
        }
        case "bb.anomaly.database.connection": {
          const payload = anomaly.payload as AnomalyDatabaseConnectionPayload;
          return payload.detail;
        }
        case "bb.anomaly.database.schema.drift": {
          const payload = anomaly.payload as AnomalyDatabaseSchemaDriftPayload;
          return `Recorded latest schema version ${payload.version} is different from the actual schema.`;
        }
      }
    };

    const action = (anomaly: Anomaly): Action => {
      switch (anomaly.type) {
        case "bb.anomaly.instance.connection":
          return {
            onClick: () => {
              router.push({
                name: "workspace.instance.detail",
                params: {
                  instanceSlug: instanceSlug(anomaly.instance),
                },
              });
            },
            title: t("anomaly.action.check-instance"),
          };
        case "bb.anomaly.instance.migration-schema":
          return {
            onClick: () => {
              router.push({
                name: "workspace.instance.detail",
                params: {
                  instanceSlug: instanceSlug(anomaly.instance),
                },
              });
            },
            title: t("anomaly.action.check-instance"),
          };
        case "bb.anomaly.database.backup.policy-violation": {
          return {
            onClick: () => {
              router.push({
                name: "workspace.database.detail",
                params: {
                  databaseSlug: databaseSlug(anomaly.database!),
                },
                hash: "#backup",
              });
            },
            title: t("anomaly.action.configure-backup"),
          };
        }
        case "bb.anomaly.database.backup.missing":
          return {
            onClick: () => {
              router.push({
                name: "workspace.database.detail",
                params: {
                  databaseSlug: databaseSlug(anomaly.database!),
                },
                hash: "#backup",
              });
            },
            title: t("anomaly.action.view-backup"),
          };
        case "bb.anomaly.database.connection":
          return {
            onClick: () => {
              router.push({
                name: "workspace.instance.detail",
                params: {
                  instanceSlug: instanceSlug(anomaly.instance),
                },
              });
            },
            title: t("anomaly.action.check-instance"),
          };
        case "bb.anomaly.database.schema.drift":
          return {
            onClick: () => {
              state.selectedAnomaly = anomaly;
              state.showModal = true;
            },
            title: t("anomaly.action.view-diff"),
          };
      }
    };

    const dismissModal = () => {
      state.showModal = false;
      state.selectedAnomaly = undefined;
    };

    return {
      COLUMN_LIST,
      state,
      typeName,
      detail,
      action,
      dismissModal,
    };
  },
});
</script>
