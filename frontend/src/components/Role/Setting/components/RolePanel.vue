<template>
  <NDrawer
    :show="role !== undefined"
    :auto-focus="false"
    width="auto"
    @update:show="(show) => !show && $emit('close')"
  >
    <NDrawerContent
      :title="mode === 'ADD' ? $t('role.setting.add') : $t('role.setting.edit')"
      :closable="true"
      class="w-[30rem] max-w-[100vw] relative"
    >
      <div
        class="flex flex-col gap-y-4"
        style="grid-template-columns: auto 1fr"
      >
        <div class="flex flex-col gap-y-2">
          <div class="textlabel">
            {{ $t("common.name") }}
            <span class="ml-0.5 text-error">*</span>
          </div>
          <div class="flex flex-col gap-y-2">
            <NInput
              v-model:value="readableName"
              :placeholder="$t('role.setting.name-placeholder')"
              :disabled="mode === 'EDIT'"
              :status="errors.name?.length ? 'error' : undefined"
            />
            <div
              v-if="mode === 'ADD' && !readableName"
              class="text-sm text-warning"
            >
              {{ $t("resource-id.cannot-be-changed-later") }}
            </div>
            <div
              v-for="(err, i) in errors.name"
              :key="i"
              class="text-sm text-error"
            >
              {{ err }}
            </div>
          </div>
        </div>

        <div class="flex flex-col gap-y-2">
          <div class="textlabel">{{ $t("common.description") }}</div>
          <div>
            <NInput
              v-model:value="state.role.description"
              type="textarea"
              :autosize="{ minRows: 3, maxRows: 10 }"
              :placeholder="$t('role.setting.description-placeholder')"
            />
          </div>
        </div>
      </div>

      <div
        v-if="state.loading"
        class="absolute inset-0 z-10 bg-white/50 flex flex-col items-center justify-center"
      >
        <BBSpin />
      </div>

      <template #footer>
        <div class="flex items-center justify-end gap-x-2">
          <NButton @click="$emit('close')">{{ $t("common.cancel") }}</NButton>
          <NButton type="primary" :disabled="!allowSave" @click="handleSave">
            {{ mode === "ADD" ? $t("common.add") : $t("common.update") }}
          </NButton>
        </div>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<script setup lang="ts">
import { computed, reactive, watch, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import { cloneDeep } from "lodash-es";
import { NButton, NDrawer, NDrawerContent, NInput } from "naive-ui";

import { Role } from "@/types/proto/v1/role_service";
import { extractRoleResourceName } from "@/utils";
import { pushNotification, useRoleStore } from "@/store";

type ValidationErrors<T> = Partial<{
  [K in keyof T]: string[];
}>;

type LocalState = {
  role: Role;
  dirty: boolean;
  loading: boolean;
};

const props = defineProps<{
  role: Role | undefined;
  mode: "ADD" | "EDIT";
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const RESOURCE_ID_PATTERN = /^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$/;
const { t } = useI18n();
const store = useRoleStore();
const state = reactive<LocalState>({
  role: Role.fromJSON({}),
  dirty: false,
  loading: false,
});

const readableName = computed({
  get() {
    return extractRoleResourceName(state.role.name);
  },
  set(name) {
    state.role.name = `roles/${name}`;
  },
});

const errors = computed(() => {
  const errors: ValidationErrors<Role> = {
    name: [],
  };
  const { role, dirty } = state;
  if (dirty) {
    if (!readableName.value) {
      errors.name?.push(
        t("role.setting.validation.required", {
          resource: t("common.name"),
        })
      );
    } else {
      if (role.name !== props.role?.name) {
        const roleList = store.roleList;
        if (roleList.findIndex((r) => r.name === role.name) >= 0) {
          errors.name?.push(
            t("role.setting.validation.duplicated", {
              resource: t("common.name"),
            })
          );
        }
        if (readableName.value.length > 64) {
          errors.name?.push(
            t("role.setting.validation.max-length", {
              length: 64,
              resource: t("common.name"),
            })
          );
        }
        if (!RESOURCE_ID_PATTERN.test(readableName.value)) {
          errors.name?.push(
            t("role.setting.validation.pattern", {
              resource: t("common.name"),
            })
          );
        }
      }
    }
  }

  return errors;
});

const allowSave = computed(() => {
  if (!state.dirty) return false;
  const keys = Object.keys(errors.value) as (keyof Role)[];
  if (
    keys.some((key) => {
      return errors.value[key]?.length ?? 0 > 0;
    })
  ) {
    return false;
  }
  return true;
});

const handleSave = async () => {
  state.loading = true;
  try {
    await store.upsertRole(state.role);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: props.mode === "ADD" ? t("common.added") : t("common.updated"),
    });
    emit("close");
  } finally {
    state.loading = false;
  }
};

watch(
  () => props.role,
  () => {
    if (props.role) {
      state.role = cloneDeep(props.role);
    }
    nextTick(() => {
      state.dirty = false;
    });
  },
  {
    immediate: true,
  }
);

watch(
  () => state.role,
  () => {
    state.dirty = true;
  },
  { deep: true }
);
</script>
