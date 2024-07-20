/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import { Range } from '../../../common/core/range.js';
import { applyEdits } from './utils.js';
export class GhostText {
    constructor(lineNumber, parts) {
        this.lineNumber = lineNumber;
        this.parts = parts;
    }
    equals(other) {
        return this.lineNumber === other.lineNumber &&
            this.parts.length === other.parts.length &&
            this.parts.every((part, index) => part.equals(other.parts[index]));
    }
    /**
     * Only used for testing/debugging.
    */
    render(documentText, debug = false) {
        const l = this.lineNumber;
        return applyEdits(documentText, [
            ...this.parts.map(p => ({
                range: { startLineNumber: l, endLineNumber: l, startColumn: p.column, endColumn: p.column },
                text: debug ? `[${p.lines.join('\n')}]` : p.lines.join('\n')
            })),
        ]);
    }
    renderForScreenReader(lineText) {
        if (this.parts.length === 0) {
            return '';
        }
        const lastPart = this.parts[this.parts.length - 1];
        const cappedLineText = lineText.substr(0, lastPart.column - 1);
        const text = applyEdits(cappedLineText, this.parts.map(p => ({
            range: { startLineNumber: 1, endLineNumber: 1, startColumn: p.column, endColumn: p.column },
            text: p.lines.join('\n')
        })));
        return text.substring(this.parts[0].column - 1);
    }
    isEmpty() {
        return this.parts.every(p => p.lines.length === 0);
    }
    get lineCount() {
        return 1 + this.parts.reduce((r, p) => r + p.lines.length - 1, 0);
    }
}
export class GhostTextPart {
    constructor(column, lines, 
    /**
     * Indicates if this part is a preview of an inline suggestion when a suggestion is previewed.
    */
    preview) {
        this.column = column;
        this.lines = lines;
        this.preview = preview;
    }
    equals(other) {
        return this.column === other.column &&
            this.lines.length === other.lines.length &&
            this.lines.every((line, index) => line === other.lines[index]);
    }
}
export class GhostTextReplacement {
    constructor(lineNumber, columnRange, newLines, additionalReservedLineCount = 0) {
        this.lineNumber = lineNumber;
        this.columnRange = columnRange;
        this.newLines = newLines;
        this.additionalReservedLineCount = additionalReservedLineCount;
        this.parts = [
            new GhostTextPart(this.columnRange.endColumnExclusive, this.newLines, false),
        ];
    }
    renderForScreenReader(_lineText) {
        return this.newLines.join('\n');
    }
    render(documentText, debug = false) {
        const replaceRange = this.columnRange.toRange(this.lineNumber);
        if (debug) {
            return applyEdits(documentText, [
                { range: Range.fromPositions(replaceRange.getStartPosition()), text: `(` },
                { range: Range.fromPositions(replaceRange.getEndPosition()), text: `)[${this.newLines.join('\n')}]` }
            ]);
        }
        else {
            return applyEdits(documentText, [
                { range: replaceRange, text: this.newLines.join('\n') }
            ]);
        }
    }
    get lineCount() {
        return this.newLines.length;
    }
    isEmpty() {
        return this.parts.every(p => p.lines.length === 0);
    }
    equals(other) {
        return this.lineNumber === other.lineNumber &&
            this.columnRange.equals(other.columnRange) &&
            this.newLines.length === other.newLines.length &&
            this.newLines.every((line, index) => line === other.newLines[index]) &&
            this.additionalReservedLineCount === other.additionalReservedLineCount;
    }
}
export function ghostTextOrReplacementEquals(a, b) {
    if (a === b) {
        return true;
    }
    if (!a || !b) {
        return false;
    }
    if (a instanceof GhostText && b instanceof GhostText) {
        return a.equals(b);
    }
    if (a instanceof GhostTextReplacement && b instanceof GhostTextReplacement) {
        return a.equals(b);
    }
    return false;
}
