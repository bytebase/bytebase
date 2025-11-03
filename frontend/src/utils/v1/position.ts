import type * as monaco from "monaco-editor/esm/vs/editor/editor.api.js";
import type { Position } from "@/types/proto-es/v1/common_pb";

export function batchConvertPositionToMonacoPosition(
  positions: Position[],
  text: string,
  _text_encoding: string = "utf-8",
  _position_encoding: string = "utf-16"
): monaco.IPosition[] {
  const lines = text.split("\n");
  const result: monaco.IPosition[] = [];
  for (const position of positions) {
    if (position.line < 1) {
      result.push({
        lineNumber: 1,
        column: 1,
      });
      continue;
    }
    // Assuming the text is in utf-8 encoding.
    const lineNumber = position.line;
    if (lineNumber > lines.length) {
      // Out of bounds, return the first character of the last line.
      result.push({
        lineNumber: lines.length,
        column: 1,
      });
      continue;
    }

    const line = lines[lineNumber - 1];
    let column = 1;
    let pushed = false;
    for (let i = 0; i < line.length; i++) {
      if (position.column <= i + 1) {
        result.push({
          lineNumber: lineNumber,
          column: column,
        });
        pushed = true;
        break;
      }
      const codePoint = line.codePointAt(i);
      if (codePoint === undefined) {
        break;
      }
      const codeUnitCount = codePoint > 0xffff ? 2 : 1;
      column += codeUnitCount;
    }
    if (!pushed) {
      result.push({
        lineNumber: lineNumber,
        column: column,
      });
    }
  }

  return result;
}

export function convertPositionLineToMonacoLine(line: number) {
  if (line < 1) {
    return 1;
  }
  return line;
}
