<template>
  <BBModal :title="$t('role.import-from-role')" @close="$emit('cancel')">
    <div class="w-96 mb-2 space-y-2">
      <div>
        <p class="textlabel mb-1">{{ $t("role.select-role") }}</p>
        <NSelect
          v-model:value="state.selectedRole"
          :options="availableRoleOptions"
          :placeholder="$t('role.select-role')"
        />
      </div>
      <template v-if="selectedRole">
        <p class="textinfolabel">
          {{ displayRoleDescription(selectedRole.name) }}
        </p>
        <div>
          <p class="textlabel mb-1">
            {{ $t("common.permissions") }} ({{
              selectedRole.permissions.length
            }})
          </p>
          <div class="max-h-[10em] overflow-auto">
            <p
              v-for="permission in selectedRole.permissions"
              :key="permission"
              class="text-sm"
            >
              {{ permission }}
            </p>
          </div>
        </div>
      </template>
      <div class="!mt-4 w-full flex flex-row justify-end items-center gap-2">
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
import { NSelect, NButton } from "naive-ui";
import { computed, reactive } from "vue";
import { useRoleStore } from "@/store";
import { displayRoleTitle, displayRoleDescription } from "@/utils";

interface LocalState {
  selectedRole?: string;
}

const emit = defineEmits<{
  (event: "cancel"): void;
  (event: "import", permissions: string[]): void;
}>();

const roleStore = useRoleStore();
const state = reactive<LocalState>({});

const availableRoleOptions = computed(() => {
  const roles = roleStore.roleList.map((role) => role.name);
  return roles.map((role) => ({
    label: displayRoleTitle(role),
    value: role,
  }));
});

const selectedRole = computed(() => {
  return roleStore.roleList.find((role) => role.name === state.selectedRole);
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
