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
        <router-link
          :to="action(anomaly).link"
          class="normal-link"
          exact-active-class=""
        >
          {{ action(anomaly).title }}
        </router-link>
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(anomaly.updatedTs) }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(anomaly.createdTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType } from "vue";
import { BBTableColumn } from "../bbkit/types";
import {
  Anomaly,
  AnomalyBackupMissingPayload,
  AnomalyBackupPolicyViolationPayload,
  AnomalyDatabaseConnectionPayload,
  AnomalyDatabaseSchemaDriftPayload,
  AnomalyType,
  Column,
} from "../types";
import { useStore } from "vuex";
import { humanizeTs, instanceSlug } from "../utils";

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
  link: string;
  title: string;
};

export default {
  name: "AnomalyTable",
  components: {},
  props: {
    columnList: {
      required: true,
      type: Object as PropType<Column[]>,
    },
    anomalyList: {
      required: true,
      type: Object as PropType<Anomaly[]>,
    },
  },
  setup(props, ctx) {
    const store = useStore();
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
          return "";
        }
      }
    };

    const action = (anomaly: Anomaly): Action => {
      switch (anomaly.type) {
        case "bb.anomaly.backup.policy-violation": {
          return {
            link: `#backup`,
            title: "Configure backup",
          };
        }
        case "bb.anomaly.backup.missing":
          return {
            link: `#backup`,
            title: "View backup",
          };
        case "bb.anomaly.database.connection":
          return {
            link: `/instance/${instanceSlug(anomaly.instance)}`,
            title: "Check instance",
          };
        case "bb.anomaly.database.schema.drift":
          return {
            link: `#migration-history`,
            title: "Check migration",
          };
      }
    };

    return {
      COLUMN_LIST,
      typeName,
      detail,
      action,
    };
  },
};
</script>
