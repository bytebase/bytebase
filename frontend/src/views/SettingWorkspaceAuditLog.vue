<template>
  <div class="w-full mt-4 space-y-4">
    <FeatureAttention
      v-if="!hasAuditLogFeature"
      feature="bb.feature.audit-log"
      :description="$t('subscription.features.bb-feature-audit-log.desc')"
    />
    <div class="flex justify-end items-center mt-1">
      <MemberSelect
        class="w-52"
        :disabled="!hasAuditLogFeature"
        :show-all="true"
        :show-system-bot="true"
        :selected-id="selectedPrincipalId"
        @select-principal-id="selectPrincipal"
      />
      <div class="w-52 ml-2">
        <TypeSelect
          :disabled="!hasAuditLogFeature"
          :selected-type-list="selectedAuditTypeList"
          @update-selected-type-list="selectAuditType"
        />
      </div>
    </div>
    <PagedAuditLogTable
      v-if="hasAuditLogFeature"
      :activity-find="{
        typePrefix:
          selectedAuditTypeList.length > 0
            ? selectedAuditTypeList
            : typePrefixList,
        user: selectedPrincipalId > 0 ? selectedPrincipalId : undefined,
        order: 'DESC',
      }"
      session-key="settings-audit-log-table"
      :page-size="10"
    >
      <template #table="{ list }">
        <AuditLogTable :audit-log-list="list" @view-detail="handleViewDetail" />
      </template>
    </PagedAuditLogTable>
    <template v-else>
      <AuditLogTable :audit-log-list="[]" />
      <div class="w-full h-full flex flex-col items-center justify-center">
        <img src="../assets/illustration/no-data.webp" class="max-h-[30vh]" />
      </div>
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
                  {{
                    (key as string).includes("Ts")
                      ? dayjs
                          .unix(value as number)
                          .format("YYYY-MM-DD HH:mm:ss Z")
                      : value
                  }}
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
import { reactive, ref, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NGrid, NGi } from "naive-ui";
import { useI18n } from "vue-i18n";
import { BBDialog } from "@/bbkit";
import { AuditActivityType, PrincipalId, EMPTY_ID } from "@/types";
import { featureToRef } from "@/store";

const dialog = ref<InstanceType<typeof BBDialog> | null>(null);
const state = reactive({
  showModal: false,
  modalContent: {},
});

const { t } = useI18n();
const router = useRouter();
const route = useRoute();

const hasAuditLogFeature = featureToRef("bb.feature.audit-log");

const logKeyMap = {
  createdTs: t("audit-log.table.created-time"),
  level: t("audit-log.table.level"),
  type: t("audit-log.table.type"),
  creator: t("audit-log.table.creator"),
  comment: t("audit-log.table.comment"),
  payload: t("audit-log.table.payload"),
};

const typePrefixList = (
  Object.keys(AuditActivityType) as Array<keyof typeof AuditActivityType>
).map((key) => AuditActivityType[key]);

const selectedPrincipalId = computed((): PrincipalId => {
  const id = parseInt(route.query.user as string, 10);
  if (id >= 0) {
    return id;
  }
  return EMPTY_ID;
});

const selectedAuditTypeList = computed((): AuditActivityType[] => {
  const typeList = route.query.type as string;
  if (typeList) {
    if (typeList.includes(",")) {
      return typeList.split(",") as AuditActivityType[];
    } else {
      return [typeList as AuditActivityType];
    }
  }
  return [];
});

const handleViewDetail = (log: any) => {
  // Display detail fields in the same order as logKeyMap.
  state.modalContent = Object.fromEntries(
    Object.keys(logKeyMap).map((logKey) => [logKey, log[logKey]])
  );
  state.showModal = true;
  dialog.value!.open();
};

const selectPrincipal = (principalId: PrincipalId) => {
  router.replace({
    name: "setting.workspace.audit-log",
    query: {
      ...route.query,
      user: principalId,
    },
  });
};

const selectAuditType = (typeList: AuditActivityType[]) => {
  if (typeList.length === 0) {
    // Clear `type=` query string if no type selected.
    const query = Object.assign({}, route.query);
    delete query.type;

    router.replace({
      name: "setting.workspace.audit-log",
      query,
    });
  } else {
    router.replace({
      name: "setting.workspace.audit-log",
      query: {
        ...route.query,
        type: typeList.join(","),
      },
    });
  }
};
</script>
