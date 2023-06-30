<template>
  <div>
    <div v-if="mode === 'editor'" class="flex gap-x-3 mb-2 text-sm">
      <div
        :class="[
          'p-2 rounded cursor-pointer',
          state.showPreview ? '' : 'bg-gray-100 font-bold',
        ]"
        @click="state.showPreview = false"
      >
        {{ $t("issue.comment-editor.editor") }}
      </div>
      <div
        :class="[
          'p-2 rounded cursor-pointer',
          state.showPreview ? 'bg-gray-100 font-bold' : '',
        ]"
        @click="state.showPreview = true"
      >
        {{ $t("issue.comment-editor.preview") }}
      </div>
      <div
        v-if="!state.showPreview"
        class="flex-1 flex items-center justify-end"
      >
        <div v-for="(toolbar, i) in toolbarItems" :key="i">
          <button class="hover:bg-gray-100 p-2" @click="toolbar.action">
            <template v-if="toolbar.text">
              <span class="font-bold">{{ toolbar.text }}</span>
            </template>
            <template v-else-if="toolbar.icon">
              <heroicons-outline:code
                v-if="toolbar.icon === 'code'"
                class="w-4 h-4"
              />
              <heroicons-outline:link
                v-else-if="toolbar.icon === 'link'"
                class="w-4 h-4"
              />
            </template>
          </button>
        </div>
      </div>
    </div>
    <iframe
      v-if="state.showPreview"
      ref="contentPreviewArea"
      :srcdoc="markdownContent"
      class="rounded-md w-full overflow-hidden"
      @load="adjustIframe"
    />
    <textarea
      v-else-if="mode === 'editor'"
      ref="contentTextArea"
      v-model="state.content"
      rows="3"
      class="textarea block w-full resize-none whitespace-pre-wrap bg-gray-100"
      :placeholder="$t('issue.leave-a-comment')"
      @input="(e: any) => sizeToFit(e.target)"
      @keydown.enter="keyboardHandler"
      @keydown.esc="
        () => {
          $emit('cancel');
          state.content = props.content;
        }
      "
    ></textarea>
  </div>
</template>

<script lang="ts" setup>
import { computed, nextTick, ref, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import DOMPurify from "dompurify";
import hljs from "highlight.js/lib/core";
import MarkdownIt from "markdown-it";
import { sizeToFit } from "@/utils";
import codeStyle from "highlight.js/styles/github.css";
import markdownStyle from "../assets/css/github-markdown-style.css";
import "../assets/css/tailwind.css";

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

interface LocalState {
  showPreview: boolean;
  content: string;
}

interface Toolbar {
  icon?: string;
  text?: string;
  action: () => void;
}

type EditorMode = "editor" | "preview";

const props = defineProps<{
  content: string;
  mode: EditorMode;
}>();
const emit = defineEmits<{
  (event: "change", value: string): void;
  (event: "submit"): void;
  (event: "cancel"): void;
}>();

const state = reactive<LocalState>({
  showPreview: props.mode === "preview",
  content: props.content,
});
const { t } = useI18n();

watch(
  () => props.mode,
  (mode) => (state.showPreview = mode === "preview")
);

const markdownPlaceholder = t("issue.comment-editor.nothing-to-preview");
const markdownContent = computed(() => {
  return DOMPurify.sanitize(md.render(state.content || markdownPlaceholder));
});
const contentTextArea = ref<HTMLTextAreaElement>();
const contentPreviewArea = ref<HTMLIFrameElement>();

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

const adjustIframe = () => {
  if (!contentPreviewArea.value) return;
  if (contentPreviewArea.value.contentWindow) {
    contentPreviewArea.value.contentWindow.document.body.style.overflow =
      "hidden";
  }

  if (contentPreviewArea.value.contentDocument) {
    const cssLink = document.createElement("style");
    cssLink.append(codeStyle, markdownStyle);
    contentPreviewArea.value.contentDocument.head.append(cssLink);
    contentPreviewArea.value.contentDocument.body.className = "markdown-body";
  }

  nextTick(() => {
    if (!contentPreviewArea.value) return;
    const height =
      contentPreviewArea.value.contentDocument?.documentElement.offsetHeight ??
      0;
    contentPreviewArea.value.style.height = `${height + 2}px`;
  });
};

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
    if (autoComplete(state.content)) {
      e.stopPropagation();
      e.preventDefault();
    }
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

  if (/^\s{0,}([0-9]{1,}\.|-)\s{1,}$/.test(currentLine)) {
    // /^\s{0,}([0-9]{1,}\.|-)\s{1,}$/ matches "- ", " - " or "1. ", " 1. ", etc.
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
  } else if (/^\s{0,}([0-9]{1,}\.|-)\s/.test(currentLine)) {
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
    if (/^\s{0,}[0-9]{1,}\.\s/.test(currentLine)) {
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
    text: "H",
    action: () => {
      insertWithCursorPosition("### ", 4);
    },
  },
  {
    text: "B",
    action: () => {
      insertWithCursorPosition("****", 2);
    },
  },
  {
    icon: "code",
    action: () => {
      insertWithCursorPosition("\n```sql\n\n```\n", 8);
    },
  },
  {
    icon: "link",
    action: () => {
      insertWithCursorPosition("[](herf)", 1);
    },
  },
];

// insertWithCursorPosition will insert the template, and put selected text (or current cursor position) in the template with specific position.
// Support templates:
// \n```\nsql{text}\n```\n
// **{text}**
// [{text}](herf)
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
  });
};
</script>
