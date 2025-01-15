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
        {{ humanizeTs(getTimeForPbTimestamp(anomaly.updateTime, 0) / 1000) }}
      </BBTableCell>
      <BBTableCell>
        {{ humanizeTs(getTimeForPbTimestamp(anomaly.createTime, 0) / 1000) }}
      </BBTableCell>
    </template>
  </BBTable>

  <DatabaseSchemaDriftAnomalyModal
    v-if="state.selectedAnomaly"
    :anomaly="state.selectedAnomaly"
    @close="state.selectedAnomaly = undefined"
  />
</template>

<script lang="ts" setup>
import type { PropType } from "vue";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBTable, BBTableCell, BBTableHeaderCell } from "@/bbkit";
import type { BBTableSectionDataSource } from "@/bbkit/types";
import { INSTANCE_ROUTE_DETAIL } from "@/router/dashboard/instance";
import { getTimeForPbTimestamp } from "@/types";
import type { Anomaly } from "@/types/proto/v1/anomaly_service";
import {
  Anomaly_AnomalyType,
  Anomaly_AnomalySeverity,
} from "@/types/proto/v1/anomaly_service";
import {
  humanizeTs,
  extractDatabaseResourceName,
  extractInstanceResourceName,
} from "@/utils";
import DatabaseSchemaDriftAnomalyModal from "./DatabaseSchemaDriftAnomalyModal.vue";

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

const state = reactive<LocalState>({});

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
    case Anomaly_AnomalyType.DATABASE_CONNECTION: {
      return `Failed to connect to the database.`;
    }
    case Anomaly_AnomalyType.DATABASE_SCHEMA_DRIFT: {
      return `Latest recorded schema is different from the actual schema.`;
    }
    default:
      // Should not reach here.
      return "UNKOWN ANOMALY DETAIL";
  }
};

const action = (anomaly: Anomaly): Action => {
  switch (anomaly.type) {
    case Anomaly_AnomalyType.DATABASE_CONNECTION: {
      return {
        onClick: () => {
          router.push({
            name: INSTANCE_ROUTE_DETAIL,
            params: {
              instanceId: extractInstanceResourceName(
                extractDatabaseResourceName(anomaly.resource).instance
              ),
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
        },
        title: t("anomaly.action.view-diff"),
      };
    default:
      return {
        onClick: () => {},
        title: "",
      };
  }
};
</script>
