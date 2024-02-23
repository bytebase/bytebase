<template>
  <Drawer
    :show="role !== undefined"
    width="auto"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <DrawerContent
      :title="mode === 'ADD' ? $t('role.setting.add') : $t('role.setting.edit')"
      :closable="true"
      class="w-[44rem] max-w-[100vw] relative"
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
              :autosize="{ minRows: 2, maxRows: 4 }"
              :placeholder="$t('role.setting.description-placeholder')"
            />
          </div>
        </div>

        <div class="flex flex-col gap-y-2">
          <div class="w-full flex flex-row justify-between items-center">
            <div class="textlabel">
              {{ $t("common.permissions") }}
              <span class="ml-0.5 text-error">*</span>
            </div>
            <NButton
              size="small"
              @click="state.showImportPermissionFromRoleModal = true"
            >
              <PlusIcon class="w-4 h-auto mr-1" />
              <span>{{ $t("role.import-from-role") }}</span>
            </NButton>
          </div>
          <NTransfer
            v-model:value="state.role.permissions"
            class="!h-[32rem]"
            source-filterable
            source-filter-placeholder="Search"
            :options="permissionOptions"
          />
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
    </DrawerContent>
  </Drawer>

  <ImportPermissionFromRoleModal
    v-if="state.showImportPermissionFromRoleModal"
    @cancel="state.showImportPermissionFromRoleModal = false"
    @import="handleImportPermissions"
  />
</template>

<script setup lang="ts">
import { cloneDeep, uniq } from "lodash-es";
import { PlusIcon } from "lucide-vue-next";
import { NButton, NInput, NTransfer } from "naive-ui";
import { computed, reactive, watch, nextTick, ref } from "vue";
import { useI18n } from "vue-i18n";
import { Drawer, DrawerContent, ResourceIdField } from "@/components/v2";
import { pushNotification, useRoleStore } from "@/store";
import {
  PROJECT_PERMISSIONS,
  ValidatedMessage,
  WORKSPACE_PERMISSIONS,
} from "@/types";
import { Role } from "@/types/proto/v1/role_service";
import { extractRoleResourceName, isDev } from "@/utils";
import { useCustomRoleSettingContext } from "../context";
import ImportPermissionFromRoleModal from "./ImportPermissionFromRoleModal.vue";

type LocalState = {
  role: Role;
  dirty: boolean;
  loading: boolean;
  showImportPermissionFromRoleModal: boolean;
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
const roleStore = useRoleStore();
const { hasCustomRoleFeature, showFeatureModal } =
  useCustomRoleSettingContext();
const state = reactive<LocalState>({
  role: Role.fromJSON({}),
  dirty: false,
  loading: false,
  showImportPermissionFromRoleModal: false,
});

const resourceId = computed({
  get() {
    return extractRoleResourceName(state.role.name);
  },
  set(value) {
    state.role.name = `roles/${value}`;
  },
});

const permissionOptions = computed(() => {
  return [...WORKSPACE_PERMISSIONS, ...PROJECT_PERMISSIONS].sort().map((p) => ({
    label: p,
    value: p,
  }));
});

const allowSave = computed(() => {
  if (!state.dirty) return false;
  if (state.role.title?.length === 0) return false;
  if (resourceIdField.value) {
    if (!resourceIdField.value.resourceId) return false;
    if (!resourceIdField.value.isValidated) return false;
  }
  if (state.role.permissions.length === 0) {
    return false;
  }
  return true;
});

const handleImportPermissions = (permissions: string[]) => {
  state.role.permissions = uniq([...state.role.permissions, ...permissions]);
  state.showImportPermissionFromRoleModal = false;
};

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
    state.role = await roleStore.upsertRole(state.role);
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
  if (roleStore.roleList.find((r) => r.name === `roles/${name}`)) {
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
  () => [state.role],
  () => {
    state.dirty = true;
  },
  { deep: true }
);
</script>
