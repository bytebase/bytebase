<template>
  <Drawer :show="show" @close="$emit('close')">
    <DrawerContent
      :title="$t('sql-review.select-review')"
      class="w-240 max-w-[100vw] relative"
    >
      <template #default>
        <div class="flex flex-col gap-y-4">
          <i18n-t
            keypath="sql-review.select-review-label"
            tag="p"
            class="textinfolabel"
          >
            <template #create>
              <NButton
                text
                type="primary"
                class="normal-link lowercase"
                @click="createPolicy"
                @disabled="!allowCreateSQLReviewPolicy"
              >
                {{ $t("sql-review.create-policy") }}
              </NButton>
            </template>
          </i18n-t>
          <SQLReviewPolicyDataTable
            :size="'small'"
            :review-list="sqlReviewStore.reviewPolicyList"
            :allow-edit="false"
            :custom-click="true"
            @row-click="emit('select', $event)"
          />
        </div>
      </template>
      <template #footer>
        <div class="flex items-center justify-end gap-x-3">
          <NButton @click="$emit('close')">
            {{ $t("common.cancel") }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed, watchEffect } from "vue";
import { useRouter } from "vue-router";
import { Drawer, DrawerContent } from "@/components/v2";
import { WORKSPACE_ROUTE_SQL_REVIEW_CREATE } from "@/router/dashboard/workspaceRoutes";
import { useSQLReviewStore } from "@/store";
import type { SQLReviewPolicy } from "@/types";
import { hasWorkspacePermissionV2 } from "@/utils";
import SQLReviewPolicyDataTable from "./SQLReviewPolicyDataTable.vue";

const props = defineProps<{
  show: boolean;
  resource: string;
}>();

const emit = defineEmits<{
  (event: "close"): void;
  (event: "select", review: SQLReviewPolicy): void;
}>();

const sqlReviewStore = useSQLReviewStore();
const router = useRouter();

watchEffect(() => {
  sqlReviewStore.fetchReviewPolicyList();
});

const allowCreateSQLReviewPolicy = computed(() => {
  return hasWorkspacePermissionV2("bb.reviewConfigs.create");
});

const createPolicy = () => {
  router.push({
    name: WORKSPACE_ROUTE_SQL_REVIEW_CREATE,
    query: {
      attachedResource: props.resource,
    },
  });
};
</script>
