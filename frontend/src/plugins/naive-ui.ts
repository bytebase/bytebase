import type { App } from "vue";
import {
  create,
  NButton,
  NButtonGroup,
  NCascader,
  NDataTable,
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
    NButton,
    NButtonGroup,
    NCascader,
    NDataTable,
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
