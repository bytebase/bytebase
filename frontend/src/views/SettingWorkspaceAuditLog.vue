<template>
  <div>
    <div class="flex justify-end items-center mt-1">
      <MemberSelect
        class="w-72"
        :show-all="true"
        :show-system-bot="true"
        :selected-id="selectedPrincipalId"
        @select-principal-id="selectPrincipal"
      />
    </div>
    <PagedAuditLogTable
      :activity-find="{
        typePrefix: typePrefixList,
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

const dialog = ref<InstanceType<typeof BBDialog> | null>(null);
const state = reactive({
  showModal: false,
  modalContent: {},
});

const { t } = useI18n();
const router = useRouter();
const route = useRoute();

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
</script>
