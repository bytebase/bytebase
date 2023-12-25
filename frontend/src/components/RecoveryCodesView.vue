<template>
  <div class="w-full space-y-4 my-8">
    <BBAttention
      type="info"
      :title="
        $t('two-factor.setup-steps.download-recovery-codes.keep-safe.self')
      "
    >
      <template #default>
        <div class="text-sm">
          <p>
            {{ $t("two-factor.setup-steps.download-recovery-codes.tips") }}
          </p>
          <p>
            {{
              $t(
                "two-factor.setup-steps.download-recovery-codes.keep-safe.description"
              )
            }}
          </p>
        </div>
      </template>
    </BBAttention>
    <div
      class="mt-8 w-full mx-auto flex flex-col justify-start items-start"
      v-bind="$attrs"
    >
      <ul
        class="w-full grid grid-cols-2 list-disc list-inside mx-auto gap-4 gap-x-24 p-8 px-12 border rounded-md bg-gray-50"
      >
        <li v-for="code in props.recoveryCodes" :key="code">
          <code class="-ml-2">{{ code }}</code>
        </li>
      </ul>
    </div>
    <div class="w-full mx-auto flex flex-row justify-end items-center">
      <NButton type="success" @click="downloadRecoveryCodes">
        <template #icon>
          <heroicons-outline:download class="w-5 h-auto text-white" />
        </template>
        {{ $t("common.download") }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
const props = withDefaults(
  defineProps<{
    recoveryCodes: string[];
  }>(),
  {
    recoveryCodes: () => [],
  }
);

const emit = defineEmits(["download"]);

const downloadRecoveryCodes = () => {
  const content = props.recoveryCodes.join("\n");
  const blob = new Blob([content], { type: "text/plain" });
  const downloadLink = document.createElement("a");
  downloadLink.href = URL.createObjectURL(blob);
  downloadLink.download = "bytebase-recovery-codes.txt";
  document.body.appendChild(downloadLink);
  downloadLink.click();
  URL.revokeObjectURL(downloadLink.href);
  emit("download");
};
</script>
