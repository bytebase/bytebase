<template>
  <Drawer :show="show" @close="$emit('dismiss')">
    <DrawerContent
      :title="
        $t('settings.sensitive-data.column-detail.masking-setting-for-column', {
          column: column.maskData.column,
        })
      "
    >
      <div class="divide-block-border divide-y space-y-8 w-[50rem] h-full">
        <div>
          <div class="w-full">
            <p class="mb-2">
              {{ $t("settings.sensitive-data.masking-level.self") }}
            </p>
            <MaskingLevelSelect
              :level-list="MASKING_LEVELS"
              :selected="column.maskData.maskingLevel"
              @update="column.maskData.maskingLevel = $event"
            />
          </div>
        </div>
        <div class="pt-8 space-y-5">
          <div class="flex justify-between">
            <div>
              <h1>
                {{
                  $t("settings.sensitive-data.column-detail.access-user-list")
                }}
              </h1>
              <span class="textinfolabel">{{
                $t(
                  "settings.sensitive-data.column-detail.access-user-list-desc"
                )
              }}</span>
            </div>
            <NButton type="primary" :disabled="true" @click="">
              {{ $t("settings.sensitive-data.grant-access") }}
            </NButton>
          </div>
          <BBTable
            ref="tableRef"
            :column-list="tableHeaderList"
            :data-source="accessUserList"
            :show-header="true"
            :left-bordered="true"
            :right-bordered="true"
            :top-bordered="true"
            :bottom-bordered="true"
            :compact-section="true"
            :row-clickable="false"
            @click-row=""
          >
            <template
              #body="{
                rowData: item,
                row,
              }: {
                rowData: AccessUser,
                row: number,
              }"
            >
            </template>
          </BBTable>
        </div>
      </div>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { groupBy } from "lodash-es";
import { NButton, NDatePicker } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import type { BBTableColumn, BBTableSectionDataSource } from "@/bbkit/types";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  usePolicyByParentAndType,
  useUserStore,
  pushNotification,
  useCurrentUserV1,
} from "@/store";
import { ComposedDatabase } from "@/types";
import { Expr } from "@/types/proto/google/type/expr";
import {
  LoginRequest,
  LoginResponse,
  User,
  UserType,
} from "@/types/proto/v1/auth_service";
import { MaskingLevel, maskingLevelToJSON } from "@/types/proto/v1/common";
import {
  Policy,
  PolicyType,
  PolicyResourceType,
  MaskingExceptionPolicy_MaskingException,
  MaskingExceptionPolicy_MaskingException_Action,
  maskingExceptionPolicy_MaskingException_ActionToJSON,
} from "@/types/proto/v1/org_policy_service";
import { extractInstanceResourceName } from "@/utils";
import { hasWorkspacePermissionV1 } from "@/utils";
import { SensitiveColumn } from "./types";

interface AccessUser {
  user: User;
  supportActions: Set<MaskingExceptionPolicy_MaskingException_Action>;
  maskingLevel: MaskingLevel;
}

const props = defineProps<{
  show: boolean;
  column: SensitiveColumn;
}>();
const emit = defineEmits(["dismiss"]);

const MASKING_LEVELS = [
  MaskingLevel.FULL,
  MaskingLevel.PARTIAL,
  MaskingLevel.NONE,
];

const { t } = useI18n();
const userStore = useUserStore();
const currentUserV1 = useCurrentUserV1();

const policy = usePolicyByParentAndType({
  parentPath: props.column.database.name,
  policyType: PolicyType.MASKING,
});

const allowAdmin = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sensitive-data",
    currentUserV1.value.userRole
  );
});

const accessUserList = computed((): AccessUser[] => {
  if (!policy.value || !policy.value.maskingExceptionPolicy) {
    return [];
  }
  return [];
});

const tableHeaderList = computed(() => {
  const list: BBTableColumn[] = [
    {
      title: t("common.user"),
    },
    {
      title: t("settings.sensitive-data.action.export"),
    },
    {
      title: t("settings.sensitive-data.action.query"),
    },
    {
      title: t("settings.sensitive-data.masking-level.self"),
    },
    {
      title: t("common.expiration"),
    },
  ];
  if (allowAdmin.value) {
    list.push({
      title: "",
    });
  }
  return list;
});
</script>
