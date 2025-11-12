import {
  StarIcon,
  MoreHorizontalIcon,
  FolderSyncIcon,
  FolderCodeIcon,
  FolderMinusIcon,
  FileCodeIcon,
  FolderOpenIcon,
  UsersIcon,
} from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import { defineComponent, type PropType } from "vue";
import { t } from "@/plugins/i18n";
import { useUserStore } from "@/store";
import {
  Worksheet_Visibility,
  type Worksheet,
} from "@/types/proto-es/v1/worksheet_service_pb";
import type { WorsheetFolderNode } from "@/views/sql-editor/Sheet";

// Prefix icon component
export const TreeNodePrefix = defineComponent({
  name: "TreeNodePrefix",
  props: {
    node: {
      type: Object as PropType<WorsheetFolderNode>,
      required: true,
    },
    expandedKeys: {
      type: Object as PropType<Set<string>>,
      required: true,
    },
    rootPath: {
      type: String,
      required: true,
    },
    view: {
      type: String as PropType<"my" | "shared">,
      required: true,
    },
  },
  setup(props) {
    return () => {
      if (props.node.worksheet) {
        // the node is file
        return <FileCodeIcon class="w-4 h-auto text-gray-600" />;
      }
      if (props.expandedKeys.has(props.node.key)) {
        // is opened folder
        return <FolderOpenIcon class="w-4 h-auto text-gray-600" />;
      }
      if (props.node.key === props.rootPath) {
        // root folder icon
        if (props.view === "shared") {
          return <FolderSyncIcon class="w-4 h-auto text-gray-600" />;
        }
        return <FolderCodeIcon class="w-4 h-auto text-gray-600" />;
      }
      if (props.node.empty) {
        // empty folder icon
        return <FolderMinusIcon class="w-4 h-auto text-gray-600" />;
      }
      // fallback to normal folder icon
      return <FolderCodeIcon class="w-4 h-auto text-gray-600" />;
    };
  },
});

// Suffix icons component
export const TreeNodeSuffix = defineComponent({
  name: "TreeNodeSuffix",
  props: {
    node: {
      type: Object as PropType<WorsheetFolderNode>,
      required: true,
    },
    isWorksheetCreator: {
      type: Function as PropType<(worksheet: Worksheet) => boolean>,
      required: true,
    },
  },
  emits: ["contextMenuShow", "sharePanelShow", "toggleStar"],
  setup(props, { emit }) {
    const userStore = useUserStore();

    const visibilityDisplayName = (visibility: Worksheet_Visibility) => {
      switch (visibility) {
        case Worksheet_Visibility.PRIVATE:
          return t("sql-editor.private");
        case Worksheet_Visibility.PROJECT_READ:
          return t("sql-editor.project-read");
        case Worksheet_Visibility.PROJECT_WRITE:
          return t("sql-editor.project-write");
        default:
          return "";
      }
    };

    const creatorForSheet = (sheet: Worksheet) => {
      return (
        userStore.getUserByIdentifier(sheet.creator)?.title ?? sheet.creator
      );
    };

    const renderShareIcon = (worksheet: Worksheet) => {
      if (
        worksheet.visibility !== Worksheet_Visibility.PROJECT_READ &&
        worksheet.visibility !== Worksheet_Visibility.PROJECT_WRITE
      ) {
        return null;
      }

      return (
        <NTooltip>
          {{
            trigger: () => (
              <UsersIcon
                class="w-4 text-gray-400"
                onClick={(e) => emit("sharePanelShow", e, props.node)}
              />
            ),
            default: () => {
              return (
                <div>
                  <div>
                    {t("common.visibility")}
                    {": "}
                    {visibilityDisplayName(worksheet.visibility)}
                  </div>
                  {props.isWorksheetCreator(worksheet) ? null : (
                    <div>
                      {t("common.creator")}
                      {": "}
                      {creatorForSheet(worksheet)}
                    </div>
                  )}
                </div>
              );
            },
          }}
        </NTooltip>
      );
    };

    return () => {
      if (!props.node.worksheet) {
        return (
          <MoreHorizontalIcon
            class="w-4 h-auto text-gray-600"
            onClick={(e: MouseEvent) => emit("contextMenuShow", e, props.node)}
          />
        );
      }

      return (
        <div class="inline-flex gap-1">
          {renderShareIcon(props.node.worksheet)}
          <StarIcon
            class={`w-4 h-auto text-gray-400 ${props.node.worksheet.starred ? "text-yellow-400" : ""}`}
            onClick={() =>
              props.node.worksheet && emit("toggleStar", props.node.worksheet)
            }
          />
          <MoreHorizontalIcon
            class="w-4 h-auto text-gray-600"
            onClick={(e: MouseEvent) => emit("contextMenuShow", e, props.node)}
          />
        </div>
      );
    };
  },
});
