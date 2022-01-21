<template>
  <div class="share-popover w-112 p-2 space-y-4">
    <h1 class="text-lg font-semibold">{{ $t("common.share") }}</h1>
    <NAlert type="info">
      {{ $t("sql-editor.share-a-link") }}
    </NAlert>
    <NInputGroup class="flex items-center justify-center">
      <n-input-group-label class="flex items-center">
        <heroicons-solid:link class="h-4 w-4" />
      </n-input-group-label>
      <n-input v-model:value="sharedTabLink" />
      <NButton
        class="w-20"
        :type="copied ? 'success' : 'primary'"
        @click="handleCopy"
      >
        <heroicons-solid:check v-if="copied" class="h-4 w-4" />
        {{ copied ? $t("common.copied") : $t("common.copy") }}
      </NButton>
    </NInputGroup>
  </div>
</template>

<script lang="ts" setup>
import { ref, toRaw } from "vue";
import { useClipboard } from "@vueuse/core";
import { useStore } from "vuex";
import {
  useNamespacedGetters,
  useNamespacedState,
} from "vuex-composition-helpers";
import { useI18n } from "vue-i18n";
import slug from "slug";
import { omit } from "lodash-es";

import { EditorSelectorGetters, SqlEditorState } from "../../../types";
import { utoa } from "../../../utils";

const { connectionContext } = useNamespacedState<SqlEditorState>("sqlEditor", [
  "connectionContext",
]);
const { currentTab } = useNamespacedGetters<EditorSelectorGetters>(
  "editorSelector",
  ["currentTab"]
);

const ctx = connectionContext.value;

const host = window.location.host;
const connectionSlug = [
  slug(ctx.instanceName as string),
  ctx.instanceId,
  slug(ctx.databaseName as string),
  ctx.databaseId,
].join("_");

const tabInfoSlug = utoa(
  JSON.stringify(omit(toRaw(currentTab.value), "queryResult"))
);
const sharedTabLink = ref(
  `${host}/sql-editor/${connectionSlug}/${tabInfoSlug}`
);

const store = useStore();
const { t } = useI18n();
const { copy, copied } = useClipboard({
  source: sharedTabLink.value,
});

const handleCopy = async () => {
  await copy();
  store.dispatch("notification/pushNotification", {
    module: "bytebase",
    style: "SUCCESS",
    title: t("sql-editor.copy-share-link"),
  });
};
</script>
