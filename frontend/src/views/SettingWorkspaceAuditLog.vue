<template>
  <div class="space-y-4">
    <div v-if="auditLogList.length > 0">
      <AuditLogTable
        :audit-log-list="auditLogList.sort((a, b) => b.createdTs - a.createdTs)"
        @view-detail="
          (log: any) => {
            console.log(log)
            state.modalContent = log
            state.showModal = true;
            dialog!.open();
          }
        "
      />
      <BBDialog
        ref="dialog"
        :title="$t('audit-log.audit-log-detail')"
        data-label="bb-audit-log-detail-dialog"
        :closable="true"
        :need-negative-btn="false"
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
    <AuditLogEmptyView v-else />
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref } from "vue";
import { NGrid, NGi } from "naive-ui";
import { useI18n } from "vue-i18n";
import { BBDialog } from "@/bbkit";
import { useAuditLogList } from "@/store";

const dialog = ref<InstanceType<typeof BBDialog> | null>(null);
const state = reactive({
  showModal: false,
  modalContent: {},
});

const { t } = useI18n();
const auditLogList = useAuditLogList();

const logKeyMap = {
  createdTs: t("audit-log.table.created-time"),
  updatedTs: t("audit-log.table.updated-time"),
  creator: t("audit-log.table.creator"),
  type: t("audit-log.table.type"),
  level: t("audit-log.table.level"),
  comment: t("audit-log.table.comment"),
  payload: t("audit-log.table.payload"),
};
</script>
