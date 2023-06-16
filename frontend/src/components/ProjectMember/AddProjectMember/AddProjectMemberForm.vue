<template>
  <div class="w-full flex flex-col justify-start items-start gap-y-4">
    <div class="w-full">
      <span>{{ $t("project.members.select-users") }}</span>
      <UserSelect
        v-model:users="state.userUidList"
        class="mt-2"
        style="width: 100%"
        :multiple="true"
        :include-all="false"
      />
    </div>
    <div class="w-full">
      <span>{{ $t("project.members.assign-role") }}</span>
      <ProjectMemberRoleSelect v-model:role="state.role" class="mt-2" />
    </div>
    <div class="w-full">
      <span>{{ $t("common.expiration") }}</span>
      <div class="w-full mt-2">
        <NRadioGroup
          v-model:value="state.expireDays"
          class="w-full !grid grid-cols-3 gap-2"
          name="radiogroup"
        >
          <div
            v-for="day in expireDaysOptions"
            :key="day.value"
            class="col-span-1 h-8 flex flex-row justify-start items-center"
          >
            <NRadio :value="day.value" :label="day.label" />
          </div>
          <div class="col-span-2 flex flex-row justify-start items-center">
            <NRadio :value="-1" :label="$t('issue.grant-request.customize')" />
            <NInputNumber
              v-model:value="state.customDays"
              class="!w-24 ml-2"
              :disabled="state.expireDays !== -1"
              :min="1"
              :show-button="false"
              :placeholder="''"
            >
              <template #suffix>{{ $t("common.date.days") }}</template>
            </NInputNumber>
          </div>
        </NRadioGroup>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */

import dayjs from "dayjs";
import { NRadio, NRadioGroup, NInputNumber } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useUserStore } from "@/store";
import { ComposedProject } from "@/types";
import { Binding } from "@/types/proto/v1/project_service";
import { Expr } from "@/types/proto/google/type/expr";
import ProjectMemberRoleSelect from "@/components/v2/Select/ProjectMemberRoleSelect.vue";

const props = defineProps<{
  project: ComposedProject;
  binding: Binding;
}>();

interface LocalState {
  userUidList: string[];
  role?: string;
  expireDays: number;
  customDays: number;
}

const { t } = useI18n();
const userStore = useUserStore();
const state = reactive<LocalState>({
  userUidList: [],
  expireDays: 0,
  customDays: 7,
});

const expireDaysOptions = computed(() => [
  {
    value: 7,
    label: t("common.date.days", { days: 7 }),
  },
  {
    value: 30,
    label: t("common.date.days", { days: 30 }),
  },
  {
    value: 60,
    label: t("common.date.days", { days: 60 }),
  },
  {
    value: 90,
    label: t("common.date.days", { days: 90 }),
  },
  {
    value: 180,
    label: t("common.date.days", { days: 180 }),
  },
  {
    value: 365,
    label: t("common.date.days", { days: 365 }),
  },
  {
    value: 0,
    label: t("project.members.never-expires"),
  },
]);

watch(
  () => state,
  () => {
    if (state.userUidList) {
      props.binding.members = state.userUidList.map((uid) => {
        const user = userStore.getUserById(uid);
        return `user:${user!.email}`;
      });
    }
    if (state.role) {
      props.binding.role = state.role;
    }
    if (state.expireDays === 0) {
      props.binding.condition = undefined;
    } else {
      let days = state.expireDays;
      if (state.expireDays === -1) {
        days = state.customDays;
      }
      props.binding.condition = Expr.create({
        expression: `request.time < timestamp("${dayjs()
          .add(days, "days")
          .toISOString()}")`,
      });
    }
  },
  {
    deep: true,
  }
);
</script>
