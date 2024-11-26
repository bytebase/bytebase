import { useStyleTag } from "@vueuse/core";
import SchemaIconSVG from "lucide-static/icons/box.svg?raw";
import FunctionIconSVG from "lucide-static/icons/square-function.svg?raw";
import TableIconSVG from "lucide-static/icons/table.svg?raw";
import type monaco from "monaco-editor";
import type { MonacoModule } from "../types";

// Base on heroicons-outline:circle-stack
const DatabaseIconSVG = `<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
  <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 0v3.75m-16.5-3.75v3.75m16.5 0v3.75C20.25 16.153 16.556 18 12 18s-8.25-1.847-8.25-4.125v-3.75m16.5 0c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125" />
</svg>
`;

// Based on lucide:columns-3 and removing the second gap line
const ColumnIconSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-columns-3">
  <rect width="18" height="18" x="3" y="3" rx="2"/><path d="M9 3v18"/>
</svg>`;

// Combine lucide:table and lucide:glasses
// See <ViewIcon /> for more details
const ViewIconSVG = {
  normal: `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="none" stroke-linecap="round" stroke-linejoin="round">
  <g stroke="rgb(156 163 175)" stroke-width="2" >
    <path d="M12 3v18"/>
    <rect width="18" height="18" x="3" y="3" rx="2"/>
    <path d="M3 9h18M3 15h18"/>
  </g>
  <g transform="scale(.75)" transform-origin="100% 100%" stroke="rgb(79 70 229)" stroke-width="3.5">
    <circle cx="6" cy="15" r="4"/>
    <circle cx="18" cy="15" r="4"/>
    <path d="M14 15a2 2 0 0 0-2-2 2 2 0 0 0-2 2m-7.5-2L5 7c.7-1.3 1.4-2 3-2m13.5 8L19 7c-.7-1.3-1.5-2-3-2"/>
  </g>
</svg>`,
  active: `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="none" stroke-linecap="round" stroke-linejoin="round">
  <g stroke="#fff" stroke-width="2" >
    <path d="M12 3v18"/>
    <rect width="18" height="18" x="3" y="3" rx="2"/>
    <path d="M3 9h18M3 15h18"/>
  </g>
  <g transform="scale(.75)" transform-origin="100% 100%" stroke="rgb(79 70 229)" stroke-width="3.5">
    <circle cx="6" cy="15" r="4"/>
    <circle cx="18" cy="15" r="4"/>
    <path d="M14 15a2 2 0 0 0-2-2 2 2 0 0 0-2 2m-7.5-2L5 7c.7-1.3 1.4-2 3-2m13.5 8L19 7c-.7-1.3-1.5-2-3-2"/>
  </g>
</svg>`,
};

type MonochromeIconOverride = { css: string; url: string };
type ColoredIconOverride = {
  css: string;
  url: { normal: string; active: string };
};

const content2DataURL = (content: string, mimeType = "image/svg+xml") => {
  return `data:${mimeType};base64,${btoa(content)}`;
};

const MonochromeIconOverrides: MonochromeIconOverride[] = [
  { css: "module", url: content2DataURL(SchemaIconSVG) },
  { css: "class", url: content2DataURL(DatabaseIconSVG) },
  { css: "field", url: content2DataURL(TableIconSVG) },
  { css: "interface", url: content2DataURL(ColumnIconSVG) },
  { css: "function", url: content2DataURL(FunctionIconSVG) },
];
const ColoredIconOverrides: ColoredIconOverride[] = [
  {
    css: "variable",
    url: {
      normal: content2DataURL(ViewIconSVG.normal),
      active: content2DataURL(ViewIconSVG.active),
    },
  },
];

const createMonochromeCSS = (icon: MonochromeIconOverride) => {
  return `.monaco-editor .suggest-widget .monaco-list .monaco-list-row .suggest-icon.codicon.codicon-symbol-${icon.css}:before {
  content: " " !important;
  width: 16px;
  height: 16px;
  background-color: currentColor;
  mask-image: url(${icon.url});
  mask-size: 100% 100%;
}`;
};

const createColoredCSS = (icon: ColoredIconOverride) => {
  return `.monaco-editor .suggest-widget .monaco-list .monaco-list-row .suggest-icon.codicon.codicon-symbol-${icon.css}:before {
  content: " " !important;
  width: 16px;
  height: 16px;
  background-image: url(${icon.url.normal});
  background-size: 100% 100%;
}
.monaco-editor .suggest-widget .monaco-list .monaco-list-row.focused .suggest-icon.codicon.codicon-symbol-${icon.css}:before {
  background-image: url(${icon.url.active});
}`;
};

export const useOverrideSuggestIcons = (
  _monaco: MonacoModule,
  _editor: monaco.editor.IStandaloneCodeEditor
) => {
  const style = [
    ...MonochromeIconOverrides.map(createMonochromeCSS),
    ...ColoredIconOverrides.map(createColoredCSS),
  ].join("\n");
  useStyleTag(style);
};
