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
      <div class="flex flex-col gap-y-4">
        <div class="flex flex-col gap-y-2">
          <div class="textlabel">
            {{ $t("role.title") }}
            <span class="ml-0.5 text-error">*</span>
          </div>
          <div>
            <NInput
              v-model:value="state.role.title"
              :placeholder="$t('role.setting.title-placeholder')"
              :status="state.role.title?.length === 0 ? 'error' : undefined"
            />
          </div>
        </div>

        <ResourceIdField
          ref="resourceIdField"
          v-model:value="resourceId"
          resource-type="role"
          :resource-title="state.role.title"
          :suffix="true"
          :readonly="mode === 'EDIT'"
          :validate="validateResourceId"
          class="flex flex-col gap-y-2"
        />

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
import { cloneDeep } from "lodash-es";
import { NButton, NDrawer, NDrawerContent, NInput } from "naive-ui";
import { computed, reactive, watch, nextTick, ref } from "vue";
import { useI18n } from "vue-i18n";
import { ResourceIdField } from "@/components/v2";
import { pushNotification, useRoleStore } from "@/store";
import { ValidatedMessage } from "@/types";
import { Role } from "@/types/proto/v1/role_service";
import { extractRoleResourceName } from "@/utils";
import { useCustomRoleSettingContext } from "../context";

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

const { t } = useI18n();
const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();
const store = useRoleStore();
const { hasCustomRoleFeature, showFeatureModal } =
  useCustomRoleSettingContext();
const state = reactive<LocalState>({
  role: Role.fromJSON({}),
  dirty: false,
  loading: false,
});

const resourceId = computed({
  get() {
    return extractRoleResourceName(state.role.name);
  },
  set(value) {
    state.role.name = `roles/${value}`;
  },
});

const allowSave = computed(() => {
  if (!state.dirty) return false;
  if (state.role.title?.length === 0) return false;
  if (resourceIdField.value) {
    if (!resourceIdField.value.resourceId) return false;
    if (!resourceIdField.value.isValidated) return false;
  }
  return true;
});

const handleSave = async () => {
  if (!hasCustomRoleFeature.value) {
    showFeatureModal.value = true;

    // Getting crazy to adjust the z-indexes of the modal and the panel drawer
    // so just close the panel drawer here.
    emit("close");
    return;
  }

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

const validateResourceId = async (
  name: string
): Promise<ValidatedMessage[]> => {
  if (store.roleList.find((r) => r.name === `roles/${name}`)) {
    return [
      {
        type: "error",
        message: t("resource-id.validation.duplicated", {
          resource: t("role.self"),
        }),
      },
    ];
  }
  return [];
};

watch(
  () => props.role,
  () => {
    if (props.role) {
      state.role = cloneDeep(props.role);
      if (!state.role.title) {
        state.role.title = extractRoleResourceName(state.role.name);
      }
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
