import * as monaco from "monaco-editor";
import { ref } from "vue";

const DELIMITER = ";";

type FragmentRange = {
  startLineNumber: number;
  startColumn: number;
  endLineNumber: number;
  endColumn: number;
};

/**
 * Get the ranges of SQL fragments split by the delimiter
 * @param editor monaco.editor.IStandaloneCodeEditor
 * @returns fragmentRanges FragmentRange[]
 */
const getSQLFragmentRanges = (
  editor: monaco.editor.IStandaloneCodeEditor
): FragmentRange[] => {
  const model = editor.getModel() as monaco.editor.ITextModel;
  const linesContent = model.getLinesContent();
  const sqlFragmentRanges: FragmentRange[] = [];
  let startLineNumber = 0;

  for (let i = 0; i < linesContent.length; i++) {
    const content = linesContent[i];
    const range: FragmentRange = new monaco.Range(0, 0, 0, 0);

    const idx = content.indexOf(DELIMITER);
    // whether the content of line is a empty content
    const isEmptyContent = monaco.Range.isEmpty(
      new monaco.Range(i, 0, i, content.length)
    );

    // first non-empty line
    if (!isEmptyContent && startLineNumber === 0) {
      startLineNumber = i + 1;
    }

    // which contains the delimiter
    if (idx !== -1) {
      range.startLineNumber = startLineNumber;
      // startColumn is always first character of line
      range.startColumn = 1;
      // endLineNumber is the line number contains the delimiter
      range.endLineNumber = i + 1;
      // keep the last character of the line in the range
      range.endColumn = idx + 2;

      // jump to the next line of end line
      startLineNumber = range.endLineNumber + 1;

      sqlFragmentRanges.push(
        new monaco.Range(
          range.startLineNumber,
          range.startColumn,
          range.endLineNumber,
          range.endColumn
        )
      );
    }

    // continue count the empty content
    if (isEmptyContent && startLineNumber > 0) {
      startLineNumber += 1;
    }
  }
  return sqlFragmentRanges;
};

const lineDecorations = ref<string[]>([]);

const useLineDecorations = (
  editor: monaco.editor.IStandaloneCodeEditor,
  position: monaco.Position
) => {
  const defineLineDecorations = () => {
    const newDecorations: monaco.editor.IModelDeltaDecoration[] = [];

    const sqlFragmentRanges = getSQLFragmentRanges(editor);

    // console.log(sqlFragmentRanges);

    sqlFragmentRanges.forEach((range) => {
      // if the current position in the range, then highlight the range
      if (monaco.Range.containsPosition(range, position)) {
        newDecorations.push({
          range: range,
          options: {
            isWholeLine: true,
            linesDecorationsClassName: "sql-fragment",
          },
        });
      }
    });

    lineDecorations.value = editor.deltaDecorations([], newDecorations);

    return lineDecorations;
  };

  const disposeLineDecorations = () => {
    if (lineDecorations.value && lineDecorations.value.length > 0) {
      editor.deltaDecorations(lineDecorations.value, []);
      lineDecorations.value = [];
    }
  };

  return {
    lineDecorations,
    defineLineDecorations,
    disposeLineDecorations,
  };
};

export { useLineDecorations, getSQLFragmentRanges };
