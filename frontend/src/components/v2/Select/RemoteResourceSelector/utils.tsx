import { NCheckbox, NTag } from "naive-ui";
import { type VNodeChild } from "vue";
import EllipsisText from "@/components/EllipsisText.vue";
import { HighlightLabelText } from "@/components/v2";
import type { ResourceSelectOption, SelectSize } from "./types";

export const getRenderLabelFunc =
  <T extends { name: string }>(params: {
    showResourceName?: boolean;
    multiple?: boolean;
    customLabel?: (resource: T, keyword: string) => VNodeChild;
  }) =>
  (option: ResourceSelectOption<T>, selected: boolean, searchText: string) => {
    const { resource, label } = option;
    const node = (
      <div class="py-1">
        {params.customLabel && resource ? (
          params.customLabel(resource, searchText)
        ) : (
          <HighlightLabelText keyword={searchText} text={label} />
        )}
        {params.showResourceName && resource && (
          <div>
            <EllipsisText class="opacity-60 textinfolabel">
              <HighlightLabelText keyword={searchText} text={resource.name} />
            </EllipsisText>
          </div>
        )}
      </div>
    );
    if (params.multiple) {
      return (
        <div class="flex items-center gap-x-2 py-2">
          <NCheckbox checked={selected} size="small" />
          {node}
        </div>
      );
    }

    return node;
  };

export const getRenderTagFunc =
  <T,>(params: {
    multiple?: boolean;
    size?: SelectSize;
    disabled?: boolean;
    customLabel?: (resource: T, keyword: string) => VNodeChild;
  }) =>
  ({
    option,
    handleClose,
  }: {
    option: ResourceSelectOption<T>;
    handleClose: () => void;
  }) => {
    const { resource, label } = option;
    const node =
      params.customLabel && resource ? params.customLabel(resource, "") : label;
    if (params.multiple) {
      return (
        <NTag
          size={params.size}
          closable={!params.disabled}
          onClose={handleClose}
        >
          {node}
        </NTag>
      );
    }
    return node;
  };
