<template>
  <BBModal :title="$t('role.import-from-role')" @close="$emit('cancel')">
    <div class="w-96 mb-2 flex flex-col gap-y-2">
      <div>
        <p class="textlabel mb-1">{{ $t("role.select-role") }}</p>
        <RoleSelect v-model:value="state.selectedRole" :multiple="false" />
      </div>
      <template v-if="selectedRole">
        <p class="textinfolabel">
          {{ displayRoleDescription(selectedRole.name) }}
        </p>
        <div>
          <p class="textlabel mb-1">
            {{ $t("common.permissions") }} ({{
              filterDisplayPermissions.length
            }})
          </p>
          <div class="max-h-[10em] overflow-auto border rounded-sm p-2">
            <p
              v-for="permission in filterDisplayPermissions"
              :key="permission"
              class="text-sm leading-5"
            >
              {{ permission }}
            </p>
          </div>
        </div>
      </template>
      <div class="mt-4! w-full flex flex-row justify-end items-center gap-2">
        <NButton @click.prevent="$emit('cancel')">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton
          :disabled="!allowConfirm"
          type="primary"
          @click.prevent="handleConfirm"
        >
          {{ $t("common.confirm") }}
        </NButton>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed, reactive } from "vue";
import { BBModal } from "@/bbkit";
import { RoleSelect } from "@/components/v2/Select";
import { useRoleStore } from "@/store";
import { displayRoleDescription } from "@/utils";

interface LocalState {
  selectedRole?: string;
}

const emit = defineEmits<{
  (event: "cancel"): void;
  (event: "import", permissions: string[]): void;
}>();

const roleStore = useRoleStore();
const state = reactive<LocalState>({});

const selectedRole = computed(() => {
  return roleStore.roleList.find((role) => role.name === state.selectedRole);
});

const filterDisplayPermissions = computed(() => {
  return selectedRole.value?.permissions || [];
});

const allowConfirm = computed(() => {
  return !!selectedRole.value;
});

const handleConfirm = () => {
  if (!allowConfirm.value) {
    return;
  }

  emit("import", selectedRole.value?.permissions || []);
};
</script>
