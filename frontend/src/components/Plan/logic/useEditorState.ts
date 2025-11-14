import { type Ref, ref } from "vue";

export interface EditorState {
  isEditing: Ref<boolean>;
}

const editorState: EditorState = {
  isEditing: ref(false),
};

export const useEditorState = () => {
  const setEditingState = (editing: boolean) => {
    editorState.isEditing.value = editing;
  };

  return {
    isEditing: editorState.isEditing,
    setEditingState,
  };
};
