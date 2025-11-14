<template>
  <div class="flex items-start md:items-center flex-col md:flex-row gap-4">
    <div class="flex items-center gap-4">
      <span class="text-sm">
        {{ $t("export-data.password-optional") }}
      </span>
      <NSwitch
        v-model:value="encryptEnabled"
        :disabled="!editable"
        size="small"
      />
    </div>
    <div v-if="editable && encryptEnabled" class="flex items-center gap-4">
      <span class="text-sm">
        {{ $t("common.password") }}
        <RequiredStar />
      </span>
      <NInput
        :value="password"
        class="w-auto!"
        size="small"
        type="password"
        :input-props="{ autocomplete: 'new-password' }"
        :placeholder="$t('common.password')"
        @update:value="(val) => $emit('update:password', val)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { NInput, NSwitch } from "naive-ui";
import { ref, watch } from "vue";
import RequiredStar from "@/components/RequiredStar.vue";

const props = defineProps<{
  password?: string;
  editable?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:password", value: string): void;
}>();

const encryptEnabled = ref(Boolean(props.password));

watch(
  () => encryptEnabled.value,
  (encryptEnabled) => {
    if (!encryptEnabled) {
      emit("update:password", "");
    }
  }
);
</script>
