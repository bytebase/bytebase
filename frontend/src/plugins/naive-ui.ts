import type { App } from "vue";
import {
  create,
  NAlert,
  NButton,
  NButtonGroup,
  NCascader,
  NDataTable,
  NDropdown,
  NInput,
  NInputGroup,
  NInputGroupLabel,
  NMessageProvider,
  NModal,
  NPopover,
  NPopselect,
  NPopconfirm,
  NSpace,
  NTab,
  NTabs,
  NTabPane,
  NTag,
  NTooltip,
  NTree,
  NSelect,
  NUpload,
  NUploadDragger,
} from "naive-ui";

const naive = create({
  components: [
    NAlert,
    NButton,
    NButtonGroup,
    NCascader,
    NDataTable,
    NDropdown,
    NInput,
    NInputGroup,
    NInputGroupLabel,
    NMessageProvider,
    NModal,
    NPopover,
    NPopselect,
    NPopconfirm,
    NSpace,
    NTab,
    NTabs,
    NTabPane,
    NTag,
    NTooltip,
    NTree,
    NSelect,
    NUpload,
    NUploadDragger,
  ],
});

const install = (app: App) => {
  app.use(naive);
  reAppendMetaTag("naive-ui-style");
  reAppendMetaTag("vueuc-style");
};

export default install;

const reAppendMetaTag = (name: string) => {
  const meta = document.createElement("meta");
  meta.name = name;
  document.head.appendChild(meta);
};
