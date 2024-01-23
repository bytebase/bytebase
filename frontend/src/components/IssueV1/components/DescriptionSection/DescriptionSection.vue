<template>
  <div class="px-4 py-2 flex flex-col gap-y-2">
    <div class="flex items-center justify-between">
      <span class="textlabel">
        {{ title }}
      </span>

      <div v-if="!isCreating && allowEdit" class="flex items-center gap-x-2">
        <NButton v-if="!state.isEditing" size="tiny" @click.prevent="beginEdit">
          {{ $t("common.edit") }}
        </NButton>
        <NButton
          v-if="state.isEditing"
          size="tiny"
          :disabled="state.description === issue.description"
          :loading="state.isUpdating"
          @click.prevent="saveEdit"
        >
          {{ $t("common.save") }}
        </NButton>
        <NButton
          v-if="state.isEditing"
          size="tiny"
          quaternary
          @click.prevent="cancelEdit"
        >
          {{ $t("common.cancel") }}
        </NButton>
      </div>
    </div>

    <div class="text-sm">
      <NInput
        v-if="isCreating || state.isEditing"
        ref="inputRef"
        v-model:value="state.description"
        :placeholder="$t('issue.add-some-description')"
        :autosize="{ minRows: 3, maxRows: 10 }"
        :disabled="state.isUpdating"
        :loading="state.isUpdating"
        style="
          width: 100%;
          --n-placeholder-color: rgb(var(--color-control-placeholder));
        "
        type="textarea"
        size="small"
        @update:value="onDescriptionChange"
      />
      <div
        v-else
        class="min-h-[3rem] max-h-[12rem] whitespace-pre-wrap px-[10px] py-[4.5px] text-sm"
      >
        <template v-if="issue.description">
          <iframe
            v-if="issue.description"
            ref="contentPreviewArea"
            :srcdoc="renderedContent"
            class="rounded-md w-full overflow-hidden"
            @load="adjustIframe"
          />
        </template>
        <span v-else class="text-control-placeholder">
          {{ $t("issue.add-some-description") }}
        </span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NInput, NButton } from "naive-ui";
import { computed, nextTick, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { issueServiceClient } from "@/grpcweb";
import { emitWindowEvent } from "@/plugins";
import { pushNotification, useCurrentUserV1 } from "@/store";
import { Issue, IssueStatus } from "@/types/proto/v1/issue_service";
import {
  extractProjectResourceName,
  extractUserResourceName,
  hasProjectPermissionV2,
  isGrantRequestIssue,
  minmax,
} from "@/utils";
import { useIssueContext } from "../../logic";

type LocalState = {
  isEditing: boolean;
  isUpdating: boolean;
  description: string;
};
const [
  { default: hljs },
  { default: codeStyle },
  { default: markdownStyle },
  { default: MarkdownIt },
  { default: DOMPurify },
] = await Promise.all([
  import("highlight.js/lib/core"),
  import("highlight.js/styles/github.css?raw"),
  import("@/assets/css/github-markdown-style.css?raw"),
  import("markdown-it"),
  import("dompurify"),
]);

const md = new MarkdownIt({
  html: true,
  linkify: true,
  highlight: function (code, lang) {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return hljs.highlight(code, { language: lang }).value;
      } catch {
        return "";
      }
    }

    return ""; // use external default escaping
  },
});

const { t } = useI18n();
const { isCreating, issue } = useIssueContext();
const currentUser = useCurrentUserV1();
const contentPreviewArea = ref<HTMLIFrameElement>();

const state = reactive<LocalState>({
  isEditing: false,
  isUpdating: false,
  description: issue.value.description,
});

const inputRef = ref<InstanceType<typeof NInput>>();

const title = computed(() => {
  return isGrantRequestIssue(issue.value)
    ? t("common.reason")
    : t("common.description");
});

const allowEdit = computed(() => {
  if (isCreating.value) {
    return true;
  }
  if (issue.value.status !== IssueStatus.OPEN) {
    return false;
  }

  if (
    extractUserResourceName(issue.value.assignee) === currentUser.value.email ||
    extractUserResourceName(issue.value.creator) === currentUser.value.email
  ) {
    // Allowed if current user is the assignee or creator.
    return true;
  }

  if (
    hasProjectPermissionV2(
      issue.value.projectEntity,
      currentUser.value,
      "bb.issues.update"
    )
  ) {
    // Allowed if current has issue update permission in the project
    return true;
  }
  return false;
});

const onDescriptionChange = (description: string) => {
  if (isCreating.value) {
    issue.value.description = description;
  }
};

const beginEdit = () => {
  state.description = issue.value.description;
  state.isEditing = true;
  nextTick(() => {
    inputRef.value?.focus();
  });
};

const saveEdit = async () => {
  try {
    state.isUpdating = true;
    const issuePatch = Issue.fromJSON({
      ...issue.value,
      description: state.description,
    });
    const updated = await issueServiceClient.updateIssue({
      issue: issuePatch,
      updateMask: ["description"],
    });
    Object.assign(issue.value, updated);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
    emitWindowEvent("bb.issue-field-update");
    state.isEditing = false;
  } finally {
    state.isUpdating = false;
  }
};

const cancelEdit = () => {
  state.description = issue.value.description;
  state.isEditing = false;
};

const renderedContent = computed(() => {
  // we met a valid #{issue_id} in which issue_id is an integer and >= 0
  // render a link to the issue
  const format = issue.value.description
    .split(/(#\d+)\b/)
    .map((part) => {
      if (!part.startsWith("#")) {
        return part;
      }
      const id = parseInt(part.slice(1), 10);
      if (!Number.isNaN(id) && id > 0) {
        // Here we assume that the referenced issue and the current issue are always
        // in the same project
        const path = `projects/${extractProjectResourceName(
          issue.value.projectEntity.name
        )}/issues/${id}`;
        const url = `${window.location.origin}/${path}`;
        return `[${t("common.issue")} #${id}](${url})`;
      }
      return part;
    })
    .join("");
  return DOMPurify.sanitize(md.render(format));
});

const adjustIframe = () => {
  if (!contentPreviewArea.value) return;
  if (contentPreviewArea.value.contentWindow) {
    contentPreviewArea.value.contentWindow.document.body.style.overflow =
      "auto";
  }

  if (contentPreviewArea.value.contentDocument) {
    const cssLink = document.createElement("style");
    cssLink.append(codeStyle, markdownStyle);
    contentPreviewArea.value.contentDocument.head.append(cssLink);
    contentPreviewArea.value.contentDocument.body.className = "markdown-body";

    const links =
      contentPreviewArea.value.contentDocument.querySelectorAll("a");
    for (let i = 0; i < links.length; i++) {
      links[i].setAttribute("target", "_blank");
    }
  }

  nextTick(() => {
    if (!contentPreviewArea.value) return;
    const height =
      contentPreviewArea.value.contentDocument?.documentElement.offsetHeight ??
      0;
    const normalizedHeight = minmax(height, 48 /* 3rem */, 192 /* 12rem */);
    contentPreviewArea.value.style.height = `${normalizedHeight + 2}px`;
  });
};

// Reset the edit state after creating the issue.
watch(isCreating, (curr, prev) => {
  if (!curr && prev) {
    state.isEditing = false;
  }
});

watch(
  () => issue.value,
  (issue) => {
    if (state.isEditing) return;
    state.description = issue.description;
  },
  { immediate: true }
);
</script>
