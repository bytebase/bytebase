<template>
  <div class="w-full flex flex-col justify-start items-start gap-y-4">
    <div class="w-full">
      <div class="flex items-center justify-between">
        {{ $t("project.members.select-users") }}

        <NButton v-if="allowRemove" text @click="$emit('remove')">
          <template #icon>
            <heroicons:trash class="w-4 h-4" />
          </template>
        </NButton>
      </div>
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
      <ExpirationSelector
        class="mt-2"
        :options="expireDaysOptions"
        :value="state.expireDays"
        @update="state.expireDays = $event"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */

import dayjs from "dayjs";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useUserStore } from "@/store";
import { ComposedProject } from "@/types";
import { Binding } from "@/types/proto/v1/iam_policy";
import { Expr } from "@/types/proto/google/type/expr";
import ProjectMemberRoleSelect from "@/components/v2/Select/ProjectMemberRoleSelect.vue";
import ExpirationSelector from "@/components/ExpirationSelector.vue";

const props = defineProps<{
  project: ComposedProject;
  binding: Binding;
  allowRemove: boolean;
}>();

defineEmits<{
  (event: "remove"): void;
}>();

interface LocalState {
  userUidList: string[];
  role?: string;
  expireDays: number;
}

const { t } = useI18n();
const userStore = useUserStore();
const state = reactive<LocalState>({
  userUidList: [],
  expireDays: 7,
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
    label: t("common.date.months", { months: 6 }),
  },
  {
    value: 365,
    label: t("common.date.years", { years: 1 }),
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
      const days = state.expireDays;
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
