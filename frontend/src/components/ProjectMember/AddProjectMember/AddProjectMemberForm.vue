<template>
  <div class="w-full flex flex-col justify-start items-start gap-y-2">
    <span>Select user</span>
    <UserSelect
      v-model:user="state.userUid"
      :include-all="false"
      style="width: 100%"
    />
    <span>Assign role</span>
    <ProjectMemberRoleSelect v-model:role="state.role" />
    <span>Set expiration</span>
    <div class="w-full">
      <NRadioGroup
        v-model:value="state.expireDays"
        class="w-full !grid grid-cols-3 gap-4"
        name="radiogroup"
      >
        <div
          v-for="day in expireDaysOptions"
          :key="day.value"
          class="col-span-1 flex flex-row justify-start items-center"
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
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */

import { NRadio, NRadioGroup, NInputNumber } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { ComposedProject } from "@/types";
import { Binding } from "@/types/proto/v1/project_service";
import ProjectMemberRoleSelect from "@/components/v2/Select/ProjectMemberRoleSelect.vue";
import { useI18n } from "vue-i18n";
import { useUserStore } from "@/store";
import { Expr } from "@/types/proto/google/type/expr";

const props = defineProps<{
  project: ComposedProject;
  binding: Binding;
}>();

interface LocalState {
  userUid?: string;
  role?: string;
  expireDays: number;
  customDays: number;
}

const { t } = useI18n();
const userStore = useUserStore();
const state = reactive<LocalState>({
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
    label: "Never",
  },
]);

watch(
  () => state,
  () => {
    if (state.userUid) {
      const user = userStore.getUserById(state.userUid);
      if (user) {
        props.binding.members = [`user:${user.email}`];
      }
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
        expression: `request.time < timestamp("${new Date(
          Date.now() + days * 24 * 60 * 60 * 1000
        ).toISOString()}")`,
      });
    }
  },
  {
    deep: true,
  }
);
</script>
