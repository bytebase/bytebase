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
  NMessageProvider,
  NModal,
  NPopover,
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
    NMessageProvider,
    NModal,
    NPopover,
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
