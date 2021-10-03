<template>
  <BBTable
    :columnList="COLUMN_LIST"
    :dataSource="anomalyList"
    :showHeader="true"
    :leftBordered="true"
    :rightBordered="true"
  >
    <template v-slot:body="{ rowData: anomaly }">
      <BBTableCell :leftPadding="4" class="w-4">
        {{ typeName(anomaly.type) }}
      </BBTableCell>
      <BBTableCell class="w-48">
        {{ detail(anomaly) }}
        <span class="normal-link" @click.prevent="action(anomaly).onClick">
          {{ action(anomaly).title }}
        </span>
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(anomaly.updatedTs) }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(anomaly.createdTs) }}
      </BBTableCell>
    </template>
  </BBTable>
  <BBModal
    v-if="state.showModal"
    :title="`'${state.selectedAnomaly.database.name}' schema drift - ${state.selectedAnomaly.payload.version} vs Actual`"
    @close="dimissModal"
  >
    <div class="space-y-4">
      <code-diff
        class="w-full"
        :old-string="state.selectedAnomaly.payload.expect"
        :new-string="state.selectedAnomaly.payload.actual"
        :fileName="`${state.selectedAnomaly.payload.version} (left) vs Actual (right)`"
        output-format="side-by-side"
      />
      <div class="flex justify-end px-4">
        <button type="button" class="btn-primary" @click.prevent="dimissModal">
          Close
        </button>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts">
import { PropType, reactive } from "vue";
import { CodeDiff } from "v-code-diff";
import { BBTableColumn } from "../bbkit/types";
import {
  Anomaly,
  AnomalyBackupMissingPayload,
  AnomalyBackupPolicyViolationPayload,
  AnomalyDatabaseConnectionPayload,
  AnomalyDatabaseSchemaDriftPayload,
  AnomalyType,
} from "../types";
import { useStore } from "vuex";
import { databaseSlug, humanizeTs, instanceSlug } from "../utils";
import { useRouter } from "vue-router";

const COLUMN_LIST: BBTableColumn[] = [
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
    anomalyList: {
      required: true,
      type: Object as PropType<Anomaly[]>,
    },
  },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      showModal: false,
    });

    const typeName = (type: AnomalyType) => {
      switch (type) {
        case "bb.anomaly.backup.policy-violation":
          return "Backup enforcement violation";
        case "bb.anomaly.backup.missing":
          return "Missing backup";
        case "bb.anomaly.database.connection":
          return "Connection failure";
        case "bb.anomaly.database.schema.drift":
          return "Schema drift";
      }
    };

    const detail = (anomaly: Anomaly) => {
      switch (anomaly.type) {
        case "bb.anomaly.backup.policy-violation": {
          const environment = store.getters["environment/environmentById"](
            anomaly.instance.environment.id
          );
          const payload =
            anomaly.payload as AnomalyBackupPolicyViolationPayload;
          return `'${environment.name}' environment requires ${payload.expectedSchedule} auto-backup.`;
        }
        case "bb.anomaly.backup.missing": {
          const payload = anomaly.payload as AnomalyBackupMissingPayload;
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
          return `The recorded latest schema version ${payload.version} is different from the actual schema.`;
        }
      }
    };

    const action = (anomaly: Anomaly): Action => {
      switch (anomaly.type) {
        case "bb.anomaly.backup.policy-violation": {
          return {
            onClick: () => {
              router.push({
                name: "workspace.database.detail",
                params: {
                  databaseSlug: databaseSlug(anomaly.database),
                },
                hash: "#backup",
              });
            },
            title: "Configure backup",
          };
        }
        case "bb.anomaly.backup.missing":
          return {
            onClick: () => {
              router.push({
                name: "workspace.database.detail",
                params: {
                  databaseSlug: databaseSlug(anomaly.database),
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

    const dimissModal = () => {
      state.showModal = false;
      state.selectedAnomaly = undefined;
    };

    return {
      COLUMN_LIST,
      state,
      typeName,
      detail,
      action,
      dimissModal,
    };
  },
};
</script>
