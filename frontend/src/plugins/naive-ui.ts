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
  NTabs,
  NTabPane,
  NTag,
  NTooltip,
  NTree,
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
    NTabs,
    NTabPane,
    NTag,
    NTooltip,
    NTree,
    NUpload,
    NUploadDragger,
  ],
});

const install = (app: App) => app.use(naive);

export default install;
