<template>
  <div class="w-full space-y-4">
    <FeatureAttention feature="bb.feature.audit-log" />
    <div class="flex justify-end items-center space-x-2">
      <UserSelect
        v-model:user="state.userUid"
        :multiple="false"
        :include-all="true"
      />
      <div class="w-52">
        <TypeSelect
          v-model:selected="state.selectedAuditTypeList"
          :disabled="!hasAuditLogFeature"
        />
      </div>
      <div class="hidden sm:block w-112">
        <NDatePicker
          v-model:value="selectedTimeRange"
          type="datetimerange"
          :on-confirm="confirmDatePicker"
          :on-clear="clearDatePicker"
          clearable
        >
        </NDatePicker>
      </div>
      <DataExportButton
        size="medium"
        :support-formats="[
          ExportFormat.CSV,
          ExportFormat.JSON,
          ExportFormat.XLSX,
        ]"
        :disabled="!hasAuditLogFeature"
        @export="handleExport"
      />
    </div>
    <PagedActivityTable
      v-if="hasAuditLogFeature"
      :activity-find="activityFind"
      session-key="bb.page-audit-log-table.settings-audit-log-table"
      :page-size="10"
    >
      <template #table="{ list }">
        <AuditLogTable :audit-log-list="list" @view-detail="handleViewDetail" />
      </template>
    </PagedActivityTable>
    <template v-else>
      <AuditLogTable :audit-log-list="[]" />
    </template>

    <BBDialog
      ref="dialog"
      :title="$t('audit-log.audit-log-detail')"
      data-label="bb-audit-log-detail-dialog"
      :closable="true"
      :show-negative-btn="false"
    >
      <div class="w-192 font-mono">
        <dl>
          <dd
            v-for="(value, key) in state.modalContent"
            :key="key"
            class="flex items-start text-sm md:mr-4 mb-1"
          >
            <NGrid x-gap="20" :cols="20">
              <NGi span="3">
                <span class="textlabel whitespace-nowrap">{{
                  logKeyMap[key]
                }}</span
                ><span class="mr-1">:</span>
              </NGi>
              <NGi span="17">
                <span v-if="value !== ''">
                  {{ value }}
                </span>
                <span v-else class="italic text-gray-500">
                  {{ $t("audit-log.table.empty") }}
                </span>
              </NGi>
            </NGrid>
          </dd>
        </dl>
      </div>
    </BBDialog>
  </div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { NGrid, NGi, NDatePicker } from "naive-ui";
import { BinaryLike } from "node:crypto";
import { reactive, ref, computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { BBDialog } from "@/bbkit";
import { featureToRef, useUserStore, useActivityV1Store } from "@/store";
import {
  AuditActivityTypeList,
  UNKNOWN_ID,
  AuditActivityTypeI18nNameMap,
  FindActivityMessage,
} from "@/types";
import { ExportFormat } from "@/types/proto/v1/common";
import {
  LogEntity,
  LogEntity_Action,
  logEntity_LevelToJSON,
} from "@/types/proto/v1/logging_service";

const dialog = ref<InstanceType<typeof BBDialog> | null>(null);
const state = reactive({
  modalContent: {},
  userUid: String(UNKNOWN_ID),
  selectedAuditTypeList: [] as LogEntity_Action[],
});

const { t } = useI18n();
const router = useRouter();
const route = useRoute();
const userStore = useUserStore();
const activityV1Store = useActivityV1Store();

const hasAuditLogFeature = featureToRef("bb.feature.audit-log");

const logKeyMap = {
  createdTs: t("audit-log.table.created-time"),
  level: t("audit-log.table.level"),
  action: t("audit-log.table.type"),
  creator: t("audit-log.table.actor"),
  comment: t("audit-log.table.comment"),
  payload: t("audit-log.table.payload"),
};

watch(
  () => state.userUid,
  (uid) => {
    router.replace({
      name: "setting.workspace.audit-log",
      query: {
        ...route.query,
        user: parseInt(uid, 10) > 0 ? uid : undefined,
      },
    });
  }
);

watch(
  () => state.selectedAuditTypeList,
  (list) => {
    router.replace({
      name: "setting.workspace.audit-log",
      query: {
        ...route.query,
        type: list.join(","),
      },
    });
  }
);

const selectedUserEmail = computed((): string => {
  const id = route.query.user as string;
  const selected = userStore.getUserById(id);
  return selected?.email ?? "";
});

const selectedTimeRange = computed((): [number, number] => {
  const defaultTimeRange = [0, Date.now()] as [number, number];
  const createdTsAfter = route.query.createdTsAfter as string;
  if (createdTsAfter) {
    defaultTimeRange[0] = parseInt(createdTsAfter, 10);
  }
  const createdTsBefore = route.query.createdTsBefore as string;
  if (createdTsBefore) {
    defaultTimeRange[1] = parseInt(createdTsBefore, 10);
  }
  return defaultTimeRange;
});

const handleViewDetail = (log: LogEntity) => {
  // Display detail fields in the same order as logKeyMap.
  state.modalContent = Object.fromEntries(
    Object.keys(logKeyMap)
      .map((logKey) => {
        switch (logKey) {
          case "createdTs":
            return [
              logKey,
              dayjs(log.createTime).format("YYYY-MM-DD HH:mm:ss Z"),
            ];
          case "level":
            return [logKey, logEntity_LevelToJSON(log.level)];
          case "action":
            return [
              logKey,
              t(AuditActivityTypeI18nNameMap.get(log.action) ?? ""),
            ];
          case "creator":
            return [logKey, log.creator];
          case "comment":
            return [logKey, log.comment];
          case "payload":
            return [logKey, log.payload];
        }
        return [];
      })
      .filter((arr) => arr.length === 2)
  );
  dialog.value!.open();
};

const confirmDatePicker = (value: [number, number]) => {
  router.replace({
    name: "setting.workspace.audit-log",
    query: {
      ...route.query,
      createdTsAfter: value[0],
      createdTsBefore: value[1],
    },
  });
};

const clearDatePicker = () => {
  router.replace({
    name: "setting.workspace.audit-log",
    query: {
      ...route.query,
      createdTsAfter: 0,
      createdTsBefore: Date.now(),
    },
  });
};

const activityFind = computed((): FindActivityMessage => {
  return {
    action:
      state.selectedAuditTypeList.length > 0
        ? state.selectedAuditTypeList
        : AuditActivityTypeList,
    creatorEmail: selectedUserEmail.value,
    order: "desc",
    createdTsAfter: selectedTimeRange.value
      ? selectedTimeRange.value[0]
      : undefined,
    createdTsBefore: selectedTimeRange.value
      ? selectedTimeRange.value[1]
      : undefined,
  };
});

const handleExport = async (
  format: ExportFormat,
  callback: (content: BinaryLike | Blob, format: ExportFormat) => void
) => {
  const content = await activityV1Store.exportData({
    find: activityFind.value,
    format,
  });
  callback(content, format);
};
</script>
