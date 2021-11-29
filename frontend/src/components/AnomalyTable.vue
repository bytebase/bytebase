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
        <svg
          v-if="anomaly.severity == 'MEDIUM'"
          class="w-6 h-6 text-info"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
          ></path>
        </svg>
        <svg
          v-else-if="anomaly.severity == 'HIGH'"
          class="w-6 h-6 text-warning"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
          ></path>
        </svg>
        <svg
          v-else-if="anomaly.severity == 'CRITICAL'"
          class="w-6 h-6 text-error"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
          ></path>
        </svg>
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
          Close
        </button>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts">
import { PropType, reactive } from "vue";
import { CodeDiff } from "v-code-diff";
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
import { useStore } from "vuex";
import { databaseSlug, humanizeTs, instanceSlug } from "../utils";
import { useRouter } from "vue-router";

const COLUMN_LIST: BBTableColumn[] = [
  {
    title: "",
  },
  {
    title: "Type",
  },
  {
    title: "Detail",
  },
  {
    title: "Last seen",
  },
  {
    title: "First seen",
  },
];

type Action = {
  onClick: () => void;
  title: string;
};

interface LocalState {
  showModal: boolean;
  selectedAnomaly?: Anomaly;
}

export default {
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
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      showModal: false,
    });

    const typeName = (type: AnomalyType): string => {
      switch (type) {
        case "bb.anomaly.instance.connection":
          return "Connection failure";
        case "bb.anomaly.instance.migration-schema":
          return "Missing migration schema";
        case "bb.anomaly.database.backup.policy-violation":
          return "Backup enforcement violation";
        case "bb.anomaly.database.backup.missing":
          return "Missing backup";
        case "bb.anomaly.database.connection":
          return "Connection failure";
        case "bb.anomaly.database.schema.drift":
          return "Schema drift";
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
          const environment = store.getters["environment/environmentByID"](
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
            title: "Check instance",
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
            title: "Check instance",
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
            title: "Configure backup",
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
            title: "View backup",
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
            title: "Check instance",
          };
        case "bb.anomaly.database.schema.drift":
          return {
            onClick: () => {
              state.selectedAnomaly = anomaly;
              state.showModal = true;
            },
            title: "View diff",
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
};
</script>
