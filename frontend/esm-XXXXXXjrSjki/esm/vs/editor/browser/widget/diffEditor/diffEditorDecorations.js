/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import { h } from '../../../../base/browser/dom.js';
import { renderIcon } from '../../../../base/browser/ui/iconLabel/iconLabels.js';
import { Codicon } from '../../../../base/common/codicons.js';
import { Disposable, toDisposable } from '../../../../base/common/lifecycle.js';
import { autorunWithStore, derived } from '../../../../base/common/observable.js';
import { diffAddDecoration, diffAddDecorationEmpty, diffDeleteDecoration, diffDeleteDecorationEmpty, diffLineAddDecorationBackground, diffLineAddDecorationBackgroundWithIndicator, diffLineDeleteDecorationBackground, diffLineDeleteDecorationBackgroundWithIndicator, diffWholeLineAddDecoration, diffWholeLineDeleteDecoration } from './decorations.js';
import { MovedBlocksLinesPart } from './movedBlocksLines.js';
import { applyObservableDecorations } from './utils.js';
import { LineRange, LineRangeSet } from '../../../common/core/lineRange.js';
import { Range } from '../../../common/core/range.js';
import { GlyphMarginLane } from '../../../common/model.js';
import { localizeWithPath } from '../../../../nls.js';
export class DiffEditorDecorations extends Disposable {
    constructor(_editors, _diffModel, _options, widget) {
        super();
        this._editors = _editors;
        this._diffModel = _diffModel;
        this._options = _options;
        this._decorations = derived(this, (reader) => {
            const diff = this._diffModel.read(reader)?.diff.read(reader);
            if (!diff) {
                return null;
            }
            const movedTextToCompare = this._diffModel.read(reader).movedTextToCompare.read(reader);
            const renderIndicators = this._options.renderIndicators.read(reader);
            const showEmptyDecorations = this._options.showEmptyDecorations.read(reader);
            const originalDecorations = [];
            const modifiedDecorations = [];
            if (!movedTextToCompare) {
                for (const m of diff.mappings) {
                    if (!m.lineRangeMapping.original.isEmpty) {
                        originalDecorations.push({ range: m.lineRangeMapping.original.toInclusiveRange(), options: renderIndicators ? diffLineDeleteDecorationBackgroundWithIndicator : diffLineDeleteDecorationBackground });
                    }
                    if (!m.lineRangeMapping.modified.isEmpty) {
                        modifiedDecorations.push({ range: m.lineRangeMapping.modified.toInclusiveRange(), options: renderIndicators ? diffLineAddDecorationBackgroundWithIndicator : diffLineAddDecorationBackground });
                    }
                    if (m.lineRangeMapping.modified.isEmpty || m.lineRangeMapping.original.isEmpty) {
                        if (!m.lineRangeMapping.original.isEmpty) {
                            originalDecorations.push({ range: m.lineRangeMapping.original.toInclusiveRange(), options: diffWholeLineDeleteDecoration });
                        }
                        if (!m.lineRangeMapping.modified.isEmpty) {
                            modifiedDecorations.push({ range: m.lineRangeMapping.modified.toInclusiveRange(), options: diffWholeLineAddDecoration });
                        }
                    }
                    else {
                        for (const i of m.lineRangeMapping.innerChanges || []) {
                            // Don't show empty markers outside the line range
                            if (m.lineRangeMapping.original.contains(i.originalRange.startLineNumber)) {
                                originalDecorations.push({ range: i.originalRange, options: (i.originalRange.isEmpty() && showEmptyDecorations) ? diffDeleteDecorationEmpty : diffDeleteDecoration });
                            }
                            if (m.lineRangeMapping.modified.contains(i.modifiedRange.startLineNumber)) {
                                modifiedDecorations.push({ range: i.modifiedRange, options: (i.modifiedRange.isEmpty() && showEmptyDecorations) ? diffAddDecorationEmpty : diffAddDecoration });
                            }
                        }
                    }
                }
            }
            if (movedTextToCompare) {
                for (const m of movedTextToCompare.changes) {
                    const fullRangeOriginal = m.original.toInclusiveRange();
                    if (fullRangeOriginal) {
                        originalDecorations.push({ range: fullRangeOriginal, options: renderIndicators ? diffLineDeleteDecorationBackgroundWithIndicator : diffLineDeleteDecorationBackground });
                    }
                    const fullRangeModified = m.modified.toInclusiveRange();
                    if (fullRangeModified) {
                        modifiedDecorations.push({ range: fullRangeModified, options: renderIndicators ? diffLineAddDecorationBackgroundWithIndicator : diffLineAddDecorationBackground });
                    }
                    for (const i of m.innerChanges || []) {
                        originalDecorations.push({ range: i.originalRange, options: diffDeleteDecoration });
                        modifiedDecorations.push({ range: i.modifiedRange, options: diffAddDecoration });
                    }
                }
            }
            const activeMovedText = this._diffModel.read(reader).activeMovedText.read(reader);
            for (const m of diff.movedTexts) {
                originalDecorations.push({
                    range: m.lineRangeMapping.original.toInclusiveRange(), options: {
                        description: 'moved',
                        blockClassName: 'movedOriginal' + (m === activeMovedText ? ' currentMove' : ''),
                        blockPadding: [MovedBlocksLinesPart.movedCodeBlockPadding, 0, MovedBlocksLinesPart.movedCodeBlockPadding, MovedBlocksLinesPart.movedCodeBlockPadding],
                    }
                });
                modifiedDecorations.push({
                    range: m.lineRangeMapping.modified.toInclusiveRange(), options: {
                        description: 'moved',
                        blockClassName: 'movedModified' + (m === activeMovedText ? ' currentMove' : ''),
                        blockPadding: [4, 0, 4, 4],
                    }
                });
            }
            return { originalDecorations, modifiedDecorations };
        });
        this._register(new RevertButtonsFeature(_editors, _diffModel, _options, widget));
        this._register(applyObservableDecorations(this._editors.original, this._decorations.map(d => d?.originalDecorations || [])));
        this._register(applyObservableDecorations(this._editors.modified, this._decorations.map(d => d?.modifiedDecorations || [])));
    }
}
class RevertButtonsFeature extends Disposable {
    constructor(_editors, _diffModel, _options, _widget) {
        super();
        this._editors = _editors;
        this._diffModel = _diffModel;
        this._options = _options;
        this._widget = _widget;
        const emptyArr = [];
        const selectedDiffs = derived(this, (reader) => {
            /** @description selectedDiffs */
            const model = this._diffModel.read(reader);
            const diff = model?.diff.read(reader);
            if (!diff) {
                return emptyArr;
            }
            const selections = this._editors.modifiedSelections.read(reader);
            if (selections.every(s => s.isEmpty())) {
                return emptyArr;
            }
            const lineRanges = new LineRangeSet(selections.map(s => LineRange.fromRangeInclusive(s)));
            const mappings = diff.mappings.filter(m => m.lineRangeMapping.innerChanges && lineRanges.intersects(m.lineRangeMapping.modified));
            const result = mappings.map(mapping => ({
                mapping,
                rangeMappings: mapping.lineRangeMapping.innerChanges.filter(c => selections.some(s => Range.areIntersecting(c.modifiedRange, s)))
            }));
            if (result.length === 0 || result.every(r => r.rangeMappings.length === 0)) {
                return emptyArr;
            }
            return result;
        });
        this._register(autorunWithStore((reader, store) => {
            const model = this._diffModel.read(reader);
            const diff = model?.diff.read(reader);
            if (!model || !diff) {
                return;
            }
            const movedTextToCompare = this._diffModel.read(reader).movedTextToCompare.read(reader);
            if (movedTextToCompare) {
                return;
            }
            if (!this._options.shouldRenderRevertArrows.read(reader)) {
                return;
            }
            const glyphWidgetsModified = [];
            const selectedDiffs_ = selectedDiffs.read(reader);
            const diffsSet = new Set(selectedDiffs_.map(d => d.mapping));
            if (selectedDiffs_.length > 0) {
                const selections = this._editors.modifiedSelections.read(reader);
                const btn = new RevertButton(selections[selections.length - 1].positionLineNumber, this._widget, selectedDiffs_.flatMap(d => d.rangeMappings), true);
                this._editors.modified.addGlyphMarginWidget(btn);
                glyphWidgetsModified.push(btn);
            }
            for (const m of diff.mappings) {
                if (diffsSet.has(m)) {
                    continue;
                }
                if (!m.lineRangeMapping.modified.isEmpty && m.lineRangeMapping.innerChanges) {
                    const btn = new RevertButton(m.lineRangeMapping.modified.startLineNumber, this._widget, m.lineRangeMapping.innerChanges, false);
                    this._editors.modified.addGlyphMarginWidget(btn);
                    glyphWidgetsModified.push(btn);
                }
            }
            store.add(toDisposable(() => {
                for (const w of glyphWidgetsModified) {
                    this._editors.modified.removeGlyphMarginWidget(w);
                }
            }));
        }));
    }
}
class RevertButton {
    getId() { return this._id; }
    constructor(_lineNumber, _widget, _diffs, _selection) {
        this._lineNumber = _lineNumber;
        this._widget = _widget;
        this._diffs = _diffs;
        this._selection = _selection;
        this._id = `revertButton${RevertButton.counter++}`;
        this._domNode = h('div.revertButton', {
            title: this._selection
                ? localizeWithPath('vs/editor/browser/widget/diffEditor/diffEditorDecorations', 'revertSelectedChanges', 'Revert Selected Changes')
                : localizeWithPath('vs/editor/browser/widget/diffEditor/diffEditorDecorations', 'revertChange', 'Revert Change')
        }, [renderIcon(Codicon.arrowRight)]).root;
        this._domNode.onmousedown = e => {
            // don't prevent context menu from showing up
            if (e.button !== 2) {
                e.stopPropagation();
                e.preventDefault();
            }
        };
        this._domNode.onmouseup = e => {
            e.stopPropagation();
            e.preventDefault();
        };
        this._domNode.onclick = (e) => {
            this._widget.revertRangeMappings(this._diffs);
            e.stopPropagation();
            e.preventDefault();
        };
    }
    /**
     * Get the dom node of the glyph widget.
     */
    getDomNode() {
        return this._domNode;
    }
    /**
     * Get the placement of the glyph widget.
     */
    getPosition() {
        return {
            lane: GlyphMarginLane.Right,
            range: {
                startColumn: 1,
                startLineNumber: this._lineNumber,
                endColumn: 1,
                endLineNumber: this._lineNumber,
            },
            zIndex: 10001,
        };
    }
}
RevertButton.counter = 0;
