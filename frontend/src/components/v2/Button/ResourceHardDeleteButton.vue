<template>
  <BBButtonConfirm
    v-if="resource.state === State.DELETED"
    :type="'DELETE'"
    :button-text="$t('common.hard-delete.self')"
    :ok-text="$t('common.hard-delete.self')"
    :require-confirm="true"
    :confirm-title="$t('common.hard-delete.title', { name: resource.title })"
    :positive-button-props="{
      disabled: disabled,
    }"
    class="border-none!"
    @confirm="handleDelete"
  >
    <div class="mt-3">
      <div class="text-sm mb-3 flex flex-col gap-y-2">
        <div>
          {{
            $t("common.hard-delete.description", {
              resources: [$t("common.database"), $t("changelog.self")].join(
                ","
              ),
            })
          }}
        </div>
        <i18n-t tag="div" keypath="common.hard-delete.double-comfirm">
          <template #name>
            <span class="bg-gray-200 rounded-sm py-1 px-2">
              {{ resource.name }}
            </span>
          </template>
        </i18n-t>
      </div>
      <NInput v-model:value="inputText" />
    </div>
  </BBButtonConfirm>
</template>

<script
  lang="tsx"
  setup
  generic="T extends { name: string; title: string; state: State }"
>
import { NInput } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBButtonConfirm } from "@/bbkit";
import { pushNotification } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";

const props = defineProps<{
  resource: T;
}>();

const emit = defineEmits<{
  (event: "delete", resource: string): Promise<void>;
}>();

const inputText = ref<string>();
const { t } = useI18n();

const disabled = computed(() => inputText.value !== props.resource.name);

const handleDelete = async () => {
  await emit("delete", props.resource.name);
  pushNotification({
    module: "bytebase",
    style: "INFO",
    title: t("common.hard-delete.notification", { name: props.resource.title }),
  });
};
</script>
