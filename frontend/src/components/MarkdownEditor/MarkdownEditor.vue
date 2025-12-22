<template>
  <div>
    <NTabs
      v-if="mode === 'editor'"
      v-model:value="state.activeTab"
      type="line"
      :size="'small'"
      class="mb-2"
      @update:value="handleTabChange"
    >
      <NTab name="write" :tab="$t('issue.comment-editor.write')" />
      <NTab name="preview" :tab="$t('issue.comment-editor.preview')" />
      <template v-if="state.activeTab === 'write'" #suffix>
        <div class="flex items-center justify-end">
          <NTooltip
            v-for="(toolbar, i) in toolbarItems"
            :key="i"
            :show-arrow="true"
          >
            <template #trigger>
              <NButton quaternary size="small" @click="toolbar.action">
                <component :is="toolbar.icon" class="w-4 h-4" />
              </NButton>
            </template>
            <span class="w-56 text-sm">
              {{ toolbar.tooltip }}
            </span>
          </NTooltip>
        </div>
      </template>
    </NTabs>

    <iframe
      v-if="state.activeTab === 'preview' || state.showPreview"
      ref="contentPreviewArea"
      :srcdoc="renderedContent"
      class="rounded-md w-full overflow-hidden"
    />
    <div v-else-if="mode === 'editor'" class="relative">
      <textarea
        ref="contentTextArea"
        v-model="state.content"
        rows="4"
        class="block w-full px-4 py-3 resize-none whitespace-pre-wrap rounded-lg border border-gray-300 outline-hidden ring-0 text-sm"
        :placeholder="placeholder || $t('issue.leave-a-comment')"
        :maxlength="maxlength"
        @mousedown="clearIssuePanel"
        @input="(e: Event) => sizeToFit(e.target as HTMLTextAreaElement)"
        @keyup="(e: KeyboardEvent) => adjustIssuePanelWithPosition(e)"
        @keydown.enter="keyboardHandler"
        @keydown.down="handleArrowDown"
        @keydown.up="handleArrowUp"
      ></textarea>
      <div
        ref="issuePanel"
        class="border rounded-sm absolute hidden bg-white shadow-xs z-10"
      >
        <NScrollbar class="max-h-40">
          <ul class="text-sm rounded-sm divide-y divide-solid">
            <li
              v-for="(issue, index) in filterIssueList"
              :key="issue.name"
              :class="[
                'px-3 py-2 hover:bg-gray-100 cursor-pointer flex items-center gap-x-2',
                { 'bg-gray-100': state.activeIssueIndex === index },
              ]"
              @click="onIssueSelect(issue)"
              @mouseenter="() => (state.activeIssueIndex = index)"
            >
              <IssueStatusIcon
                :issue-status="issue.status"
                :task-status="issueTaskStatus(issue)"
              />
              <span class="opacity-60">#{{ extractIssueUID(issue.name) }}</span>
              <div class="whitespace-nowrap">
                <HighlightLabelText
                  :text="issue.title"
                  :keyword="issueSearchText"
                />
              </div>
            </li>
          </ul>
        </NScrollbar>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useDebounceFn } from "@vueuse/core";
import {
  BoldIcon,
  CodeIcon,
  HashIcon,
  HeadingIcon,
  LinkIcon,
} from "lucide-vue-next";
import { NButton, NScrollbar, NTab, NTabs, NTooltip } from "naive-ui";
import type { Component } from "vue";
import { nextTick, onMounted, reactive, ref, toRef, watch } from "vue";
import { useI18n } from "vue-i18n";
import { HighlightLabelText } from "@/components/v2";
import { useIssueV1Store } from "@/store";
import { type ComposedIssue, DEBOUNCE_SEARCH_DELAY } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  activeTaskInRollout,
  extractIssueUID,
  isDatabaseChangeRelatedIssue,
  sizeToFit,
} from "@/utils";
import IssueStatusIcon from "../IssueV1/components/IssueStatusIcon.vue";
import { useRenderMarkdown } from "./useRenderMarkdown";

interface LocalState {
  showPreview: boolean;
  content: string;
  activeTab: "write" | "preview";
  activeIssueIndex?: number;
}

interface Toolbar {
  icon: Component;
  tooltip: string;
  action: () => void;
}

type EditorMode = "editor" | "preview";

const props = withDefaults(
  defineProps<{
    content: string;
    mode: EditorMode;
    project: Project;
    placeholder?: string;
    autofocus?: boolean;
    maxlength?: number;
    maxHeight?: number;
  }>(),
  {
    maxHeight: 192,
  }
);

const emit = defineEmits<{
  (event: "change", value: string): void;
  (event: "submit"): void;
}>();

const state = reactive<LocalState>({
  showPreview: props.mode === "preview",
  content: props.content,
  activeTab: props.mode === "preview" ? "preview" : "write",
});

const { t } = useI18n();
const issueV1Store = useIssueV1Store();
const issueSearchText = ref<string>("");

watch(
  () => props.mode,
  (mode) => {
    state.showPreview = mode === "preview";
    state.activeTab = mode === "preview" ? "preview" : "write";
  }
);

const contentTextArea = ref<HTMLTextAreaElement>();
const contentPreviewArea = ref<HTMLIFrameElement>();
const issuePanel = ref<HTMLDivElement>();
const filterIssueList = ref<ComposedIssue[]>([]);

watch(
  () => filterIssueList.value.length,
  (length) => {
    if (length === 0) {
      if (issuePanel.value) {
        issuePanel.value.style.display = "none";
      }
      state.activeIssueIndex = undefined;
    } else if (contentTextArea.value && issuePanel.value) {
      const position = getIssuePanelPosition(contentTextArea.value);
      issuePanel.value.style.display = "block";
      issuePanel.value.style.left = `${position.x}px`;
      issuePanel.value.style.top = `${position.y + 25}px`;
      state.activeIssueIndex = 0;
    }
  }
);

const { renderedContent } = useRenderMarkdown(
  toRef(state, "content"),
  contentPreviewArea,
  toRef(props, "project"),
  {
    placeholder: `<span>${t("issue.comment-editor.nothing-to-preview")}</span>`,
    maxHeight: props.maxHeight,
  }
);

watch(
  () => state.content,
  (val) => emit("change", val)
);

watch(
  () => props.content,
  (val) => {
    if (val !== state.content) {
      state.content = val;
      nextTick(() => sizeToFit(contentTextArea.value));
    }
  }
);

const handleTabChange = (value: string) => {
  state.showPreview = value === "preview";
  state.activeTab = value as "write" | "preview";
};

watch(
  () => state.showPreview,
  (preview) => {
    if (!preview) {
      nextTick(() => {
        sizeToFit(contentTextArea.value);
        contentTextArea.value?.focus();
      });
    }
  }
);

onMounted(() => {
  if (props.autofocus && !state.showPreview) {
    nextTick(() => {
      contentTextArea.value?.focus();
    });
  }
});

const keyboardHandler = (e: KeyboardEvent) => {
  if (!contentTextArea.value) {
    return;
  }
  if (contentTextArea.value !== document.activeElement) {
    return;
  }

  if (e.code !== "Enter") {
    // For now we only trigger by the Enter event.
    return;
  }

  if (e.metaKey) {
    emit("submit");
  } else {
    if (
      filterIssueList.value.length > 0 &&
      state.activeIssueIndex !== undefined
    ) {
      const selectedIssue = filterIssueList.value[state.activeIssueIndex];
      if (selectedIssue) {
        onIssueSelect(selectedIssue);
        e.stopPropagation();
        e.preventDefault();
        return;
      }
    }

    if (autoComplete(state.content)) {
      e.stopPropagation();
      e.preventDefault();
    }
  }
};

const handleArrowDown = (e: KeyboardEvent) => {
  if (filterIssueList.value.length === 0) {
    return;
  }

  e.preventDefault();
  e.stopPropagation();

  if (state.activeIssueIndex === undefined) {
    state.activeIssueIndex = 0;
  } else if (state.activeIssueIndex < filterIssueList.value.length - 1) {
    state.activeIssueIndex++;
  }
};

const handleArrowUp = (e: KeyboardEvent) => {
  if (filterIssueList.value.length === 0) {
    return;
  }

  e.preventDefault();
  e.stopPropagation();

  if (state.activeIssueIndex === undefined) {
    state.activeIssueIndex = 0;
  } else if (state.activeIssueIndex > 0) {
    state.activeIssueIndex--;
  }
};

const autoComplete = (text: string) => {
  if (!contentTextArea.value) {
    return false;
  }
  const start = contentTextArea.value.selectionStart;
  const end = contentTextArea.value.selectionEnd;
  if (start !== end) {
    return false;
  }

  const lines = text.split("\n");
  if (lines.length === 0) {
    return false;
  }

  const currentLineIndex = getActiveLineIndex(text, start);
  const currentLine = lines[currentLineIndex];

  if (/^\s{0,}(\d{1,}\.|-)\s{1,}$/.test(currentLine)) {
    // /^\s{0,}(\d{1,}\.|-)\s{1,}$/ matches "- ", " - " or "1. ", " 1. ", etc.
    // if current line only contains "-" or number list like "1.", we will clear the line just like the GitHub.
    lines[currentLineIndex] = "";
    state.content = lines.join("\n");
    nextTick(() => {
      if (!contentTextArea.value) {
        return;
      }
      const newPosition = getCursorPosition(lines.slice(0, currentLineIndex));
      contentTextArea.value.setSelectionRange(newPosition, newPosition);
    });
    return true;
  } else if (/^\s{0,}(\d{1,}\.|-)\s/.test(currentLine)) {
    // else if current line also contains other text, we will auto-complete the markdown list.
    // for example, the "- 12|3"(| is the cursor position) should be "- 12\n- 3"
    const indent = new Array(
      currentLine.length - currentLine.trimStart().length + 1
    ).join(" ");
    const indexInCurrentLine =
      start - getCursorPosition(lines.slice(0, currentLineIndex));
    const trimEnd = currentLine.slice(indexInCurrentLine);
    lines[currentLineIndex] = currentLine.slice(0, indexInCurrentLine);

    let nextListStart = "-";
    if (/^\s{0,}\d{1,}\.\s/.test(currentLine)) {
      const guessListNumber = Number(currentLine.match(/\d+/)![0]) + 1;
      nextListStart = `${guessListNumber}.`;
    }
    lines.splice(
      currentLineIndex + 1,
      0,
      `${indent}${nextListStart} ${trimEnd}`
    );
    state.content = lines.join("\n");

    nextTick(() => {
      if (!contentTextArea.value) {
        return;
      }
      const newPosition =
        getCursorPosition(lines.slice(0, currentLineIndex + 2)) - 1;
      contentTextArea.value.setSelectionRange(newPosition, newPosition);
    });

    return true;
  }

  return false;
};

// getActiveLineIndex returns the current line index for active cursor.
const getActiveLineIndex = (
  content: string,
  cursorPosition: number
): number => {
  const lines = content.split("\n");

  let n = 0;
  for (let i = 0; i < lines.length; i++) {
    n += lines[i].length;
    if (n >= cursorPosition) {
      return i;
    }
    n++;
  }
  return lines.length - 1;
};

// getCursorPosition returns the index for active cursor in current line.
const getCursorPosition = (lines: string[]): number => {
  let n = 0;
  for (const line of lines) {
    n += line.length;
    n++;
  }
  return n;
};

const toolbarItems: Toolbar[] = [
  {
    icon: HeadingIcon,
    tooltip: t("issue.comment-editor.toolbar.header"),
    action: () => {
      insertWithCursorPosition("### ", 4);
    },
  },
  {
    icon: BoldIcon,
    tooltip: t("issue.comment-editor.toolbar.bold"),
    action: () => {
      insertWithCursorPosition("****", 2);
    },
  },
  {
    icon: CodeIcon,
    tooltip: t("issue.comment-editor.toolbar.code"),
    action: () => {
      insertWithCursorPosition("```sql\n\n```", 7);
    },
  },
  {
    icon: LinkIcon,
    tooltip: t("issue.comment-editor.toolbar.link"),
    action: () => {
      insertWithCursorPosition("[](url)", 1);
    },
  },
  {
    icon: HashIcon,
    tooltip: t("issue.comment-editor.toolbar.hashtag"),
    action: () => {
      insertWithCursorPosition("#", 1);
    },
  },
];

// insertWithCursorPosition will insert the template, and put selected text (or current cursor position) in the template with specific position.
// Support templates:
// \n```\nsql{text}\n```\n
// **{text}**
// [{text}](url)
// ### {text}
const insertWithCursorPosition = (template: string, position: number) => {
  if (!contentTextArea.value) {
    return false;
  }
  const start = contentTextArea.value.selectionStart;
  const end = contentTextArea.value.selectionEnd;

  const pendingInsert = `${template.slice(0, position)}${state.content.slice(
    start,
    end
  )}${template.slice(position)}`;
  const newContent = `${state.content.slice(
    0,
    start
  )}${pendingInsert}${state.content.slice(end)}`;

  state.content = newContent;

  nextTick(() => {
    if (!contentTextArea.value) {
      return;
    }
    contentTextArea.value.setSelectionRange(start + position, end + position);
    contentTextArea.value.focus();

    if (template === "#") {
      adjustIssuePanelWithPosition();
    }
  });
};

const clearIssuePanel = () => {
  filterIssueList.value = [];
};

// onIssueSelect will replace the input issue id with the selected issue id.
// For example, if the text is "#12|" (| is the cursor position), and select the issue with id 1234,
// we will replace the "#12|" with "#1234 |"
const onIssueSelect = (issue: ComposedIssue) => {
  if (!contentTextArea.value) {
    return false;
  }
  const start = contentTextArea.value.selectionStart;
  const end = contentTextArea.value.selectionEnd;
  if (start !== end) {
    return false;
  }

  let replaceStart = start - 1;
  while (replaceStart > 0) {
    if (state.content[replaceStart] === "#") {
      break;
    }
    replaceStart--;
  }
  replaceStart++;

  const content = state.content.split("");
  const issueId = `${extractIssueUID(issue.name)} `;
  content.splice(replaceStart, start - replaceStart, issueId);
  state.content = content.join("");

  clearIssuePanel();

  nextTick(() => {
    if (!contentTextArea.value) {
      return;
    }
    const selectionDiff = issueId.length - (start - replaceStart);
    contentTextArea.value.setSelectionRange(
      start + selectionDiff,
      end + selectionDiff
    );
    contentTextArea.value.focus();
  });

  return;
};

const issueTaskStatus = (issue: ComposedIssue) => {
  // For grant request issue, we always show the status as "NOT_STARTED" as task status.
  if (!isDatabaseChangeRelatedIssue(issue)) {
    return Task_Status.NOT_STARTED;
  }

  return activeTaskInRollout(issue.rolloutEntity)?.status;
};

const adjustIssuePanelWithPosition = useDebounceFn((e?: KeyboardEvent) => {
  if (
    e &&
    (e.key === "ArrowDown" || e.key === "ArrowUp") &&
    filterIssueList.value.length > 0
  ) {
    return;
  }

  if (!contentTextArea.value || !issuePanel.value) {
    return;
  }

  clearIssuePanel();

  const start = contentTextArea.value.selectionStart;
  const end = contentTextArea.value.selectionEnd;
  if (start !== end || start === 0) {
    return;
  }

  const text = `${state.content.slice(0, start)}${
    start === state.content.length ? " " : state.content[start]
  }`;
  const matches = text.match(/#\S{0,}\s$/);
  if (!matches) {
    return;
  }

  const query = matches[0].slice(1).trimEnd();
  const isQuery = Number.isNaN(parseInt(query));
  issueSearchText.value = isQuery ? query : "";

  issueV1Store
    .listIssues({
      find: {
        project: props.project.name,
        query: isQuery ? query : "",
      },
    })
    .then((resp) => {
      if (!isQuery && query) {
        filterIssueList.value = resp.issues.filter((issue) =>
          extractIssueUID(issue.name).startsWith(query)
        );
      } else {
        filterIssueList.value = resp.issues;
      }
    });
}, DEBOUNCE_SEARCH_DELAY);

const getIssuePanelPosition = (textArea: HTMLTextAreaElement) => {
  const start = textArea.selectionStart;
  const end = textArea.selectionEnd;
  const copy = createDivCopyForTextarea(textArea);

  const range = document.createRange();
  if (copy.firstChild) {
    range.setStart(copy.firstChild, start);
    range.setEnd(copy.firstChild, end);
  }

  const selection = document.getSelection();
  selection?.removeAllRanges();
  selection?.addRange(range);

  const rect = range.getBoundingClientRect();
  document.body.removeChild(copy);
  textArea.selectionStart = start;
  textArea.selectionEnd = end;
  textArea.focus();

  return {
    x: rect.left - textArea.scrollLeft,
    y: rect.top - textArea.scrollTop,
  };
};

const createDivCopyForTextarea = (textArea: HTMLTextAreaElement) => {
  const copy = document.createElement("div");
  copy.textContent = textArea.value;
  const style = getComputedStyle(textArea);

  const styleKeys = [
    "fontFamily",
    "fontSize",
    "fontWeight",
    "wordWrap",
    "whiteSpace",
    "borderLeftWidth",
    "borderTopWidth",
    "borderRightWidth",
    "borderBottomWidth",
  ] as const;
  styleKeys.forEach((key) => {
    copy.style[key] = style[key];
  });

  copy.style.overflow = "auto";
  copy.style.width = textArea.offsetWidth + "px";
  copy.style.height = textArea.offsetHeight + "px";
  copy.style.position = "absolute";
  copy.style.left = textArea.offsetLeft + "px";
  copy.style.top = textArea.offsetTop + "px";

  document.body.appendChild(copy);
  return copy;
};
</script>
