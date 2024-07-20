/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
var __decorate = (this && this.__decorate) || function (decorators, target, key, desc) {
    var c = arguments.length, r = c < 3 ? target : desc === null ? desc = Object.getOwnPropertyDescriptor(target, key) : desc, d;
    if (typeof Reflect === "object" && typeof Reflect.decorate === "function") r = Reflect.decorate(decorators, target, key, desc);
    else for (var i = decorators.length - 1; i >= 0; i--) if (d = decorators[i]) r = (c < 3 ? d(r) : c > 3 ? d(target, key, r) : d(target, key)) || r;
    return c > 3 && r && Object.defineProperty(target, key, r), r;
};
var __param = (this && this.__param) || function (paramIndex, decorator) {
    return function (target, key) { decorator(target, key, paramIndex); }
};
var PostEditWidget_1;
import * as dom from '../../../../base/browser/dom.js';
import { Button } from '../../../../base/browser/ui/button/button.js';
import { toAction } from '../../../../base/common/actions.js';
import { Event } from '../../../../base/common/event.js';
import { Disposable, MutableDisposable, toDisposable } from '../../../../base/common/lifecycle.js';
import './postEditWidget.css';
import { IBulkEditService, ResourceTextEdit } from '../../../browser/services/bulkEditService.js';
import { IContextKeyService } from '../../../../platform/contextkey/common/contextkey.js';
import { IContextMenuService } from '../../../../platform/contextview/browser/contextView.js';
import { IInstantiationService } from '../../../../platform/instantiation/common/instantiation.js';
import { IKeybindingService } from '../../../../platform/keybinding/common/keybinding.js';
let PostEditWidget = PostEditWidget_1 = class PostEditWidget extends Disposable {
    constructor(typeId, editor, visibleContext, showCommand, range, edits, onSelectNewEdit, _contextMenuService, contextKeyService, _keybindingService) {
        super();
        this.typeId = typeId;
        this.editor = editor;
        this.showCommand = showCommand;
        this.range = range;
        this.edits = edits;
        this.onSelectNewEdit = onSelectNewEdit;
        this._contextMenuService = _contextMenuService;
        this._keybindingService = _keybindingService;
        this.allowEditorOverflow = true;
        this.suppressMouseDown = true;
        this.create();
        this.visibleContext = visibleContext.bindTo(contextKeyService);
        this.visibleContext.set(true);
        this._register(toDisposable(() => this.visibleContext.reset()));
        this.editor.addContentWidget(this);
        this.editor.layoutContentWidget(this);
        this._register(toDisposable((() => this.editor.removeContentWidget(this))));
        this._register(this.editor.onDidChangeCursorPosition(e => {
            if (!range.containsPosition(e.position)) {
                this.dispose();
            }
        }));
        this._register(Event.runAndSubscribe(_keybindingService.onDidUpdateKeybindings, () => {
            this._updateButtonTitle();
        }));
    }
    _updateButtonTitle() {
        const binding = this._keybindingService.lookupKeybinding(this.showCommand.id)?.getLabel();
        this.button.element.title = this.showCommand.label + (binding ? ` (${binding})` : '');
    }
    create() {
        this.domNode = dom.$('.post-edit-widget');
        this.button = this._register(new Button(this.domNode, {
            supportIcons: true,
        }));
        this.button.label = '$(insert)';
        this._register(dom.addDisposableListener(this.domNode, dom.EventType.CLICK, () => this.showSelector()));
    }
    getId() {
        return PostEditWidget_1.baseId + '.' + this.typeId;
    }
    getDomNode() {
        return this.domNode;
    }
    getPosition() {
        return {
            position: this.range.getEndPosition(),
            preference: [2 /* ContentWidgetPositionPreference.BELOW */]
        };
    }
    showSelector() {
        this._contextMenuService.showContextMenu({
            getAnchor: () => {
                const pos = dom.getDomNodePagePosition(this.button.element);
                return { x: pos.left + pos.width, y: pos.top + pos.height };
            },
            getActions: () => {
                return this.edits.allEdits.map((edit, i) => toAction({
                    id: '',
                    label: edit.label,
                    checked: i === this.edits.activeEditIndex,
                    run: () => {
                        if (i !== this.edits.activeEditIndex) {
                            return this.onSelectNewEdit(i);
                        }
                    },
                }));
            }
        });
    }
};
PostEditWidget.baseId = 'editor.widget.postEditWidget';
PostEditWidget = PostEditWidget_1 = __decorate([
    __param(7, IContextMenuService),
    __param(8, IContextKeyService),
    __param(9, IKeybindingService)
], PostEditWidget);
let PostEditWidgetManager = class PostEditWidgetManager extends Disposable {
    constructor(_id, _editor, _visibleContext, _showCommand, _instantiationService, _bulkEditService) {
        super();
        this._id = _id;
        this._editor = _editor;
        this._visibleContext = _visibleContext;
        this._showCommand = _showCommand;
        this._instantiationService = _instantiationService;
        this._bulkEditService = _bulkEditService;
        this._currentWidget = this._register(new MutableDisposable());
        this._register(Event.any(_editor.onDidChangeModel, _editor.onDidChangeModelContent)(() => this.clear()));
    }
    async applyEditAndShowIfNeeded(ranges, edits, canShowWidget, token) {
        const model = this._editor.getModel();
        if (!model || !ranges.length) {
            return;
        }
        const edit = edits.allEdits[edits.activeEditIndex];
        if (!edit) {
            return;
        }
        let insertTextEdit = [];
        if (typeof edit.insertText === 'string' ? edit.insertText === '' : edit.insertText.snippet === '') {
            insertTextEdit = [];
        }
        else {
            insertTextEdit = ranges.map(range => new ResourceTextEdit(model.uri, typeof edit.insertText === 'string'
                ? { range, text: edit.insertText, insertAsSnippet: false }
                : { range, text: edit.insertText.snippet, insertAsSnippet: true }));
        }
        const allEdits = [
            ...insertTextEdit,
            ...(edit.additionalEdit?.edits ?? [])
        ];
        const combinedWorkspaceEdit = {
            edits: allEdits
        };
        // Use a decoration to track edits around the trigger range
        const primaryRange = ranges[0];
        const editTrackingDecoration = model.deltaDecorations([], [{
                range: primaryRange,
                options: { description: 'paste-line-suffix', stickiness: 0 /* TrackedRangeStickiness.AlwaysGrowsWhenTypingAtEdges */ }
            }]);
        let editResult;
        let editRange;
        try {
            editResult = await this._bulkEditService.apply(combinedWorkspaceEdit, { editor: this._editor, token });
            editRange = model.getDecorationRange(editTrackingDecoration[0]);
        }
        finally {
            model.deltaDecorations(editTrackingDecoration, []);
        }
        if (canShowWidget && editResult.isApplied && edits.allEdits.length > 1) {
            this.show(editRange ?? primaryRange, edits, async (newEditIndex) => {
                const model = this._editor.getModel();
                if (!model) {
                    return;
                }
                await model.undo();
                this.applyEditAndShowIfNeeded(ranges, { activeEditIndex: newEditIndex, allEdits: edits.allEdits }, canShowWidget, token);
            });
        }
    }
    show(range, edits, onDidSelectEdit) {
        this.clear();
        if (this._editor.hasModel()) {
            this._currentWidget.value = this._instantiationService.createInstance(PostEditWidget, this._id, this._editor, this._visibleContext, this._showCommand, range, edits, onDidSelectEdit);
        }
    }
    clear() {
        this._currentWidget.clear();
    }
    tryShowSelector() {
        this._currentWidget.value?.showSelector();
    }
};
PostEditWidgetManager = __decorate([
    __param(4, IInstantiationService),
    __param(5, IBulkEditService)
], PostEditWidgetManager);
export { PostEditWidgetManager };
