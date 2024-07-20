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
var LightBulbWidget_1;
import * as dom from '../../../../base/browser/dom.js';
import { Gesture } from '../../../../base/browser/touch.js';
import { Codicon } from '../../../../base/common/codicons.js';
import { Emitter, Event } from '../../../../base/common/event.js';
import { Disposable } from '../../../../base/common/lifecycle.js';
import { ThemeIcon } from '../../../../base/common/themables.js';
import './lightBulbWidget.css';
import { ShowAiIconMode } from '../../../common/config/editorOptions.js';
import { computeIndentLevel } from '../../../common/model/utils.js';
import { autoFixCommandId, quickFixCommandId } from './codeAction.js';
import * as nls from '../../../../nls.js';
import { ICommandService } from '../../../../platform/commands/common/commands.js';
import { IKeybindingService } from '../../../../platform/keybinding/common/keybinding.js';
var LightBulbState;
(function (LightBulbState) {
    LightBulbState.Hidden = { type: 0 /* Type.Hidden */ };
    class Showing {
        constructor(actions, trigger, editorPosition, widgetPosition) {
            this.actions = actions;
            this.trigger = trigger;
            this.editorPosition = editorPosition;
            this.widgetPosition = widgetPosition;
            this.type = 1 /* Type.Showing */;
        }
    }
    LightBulbState.Showing = Showing;
})(LightBulbState || (LightBulbState = {}));
let LightBulbWidget = LightBulbWidget_1 = class LightBulbWidget extends Disposable {
    constructor(_editor, _keybindingService, commandService) {
        super();
        this._editor = _editor;
        this._keybindingService = _keybindingService;
        this._onClick = this._register(new Emitter());
        this.onClick = this._onClick.event;
        this._state = LightBulbState.Hidden;
        this._iconClasses = [];
        this._domNode = dom.$('div.lightBulbWidget');
        this._register(Gesture.ignoreTarget(this._domNode));
        this._editor.addContentWidget(this);
        this._register(this._editor.onDidChangeModelContent(_ => {
            // cancel when the line in question has been removed
            const editorModel = this._editor.getModel();
            if (this.state.type !== 1 /* LightBulbState.Type.Showing */ || !editorModel || this.state.editorPosition.lineNumber >= editorModel.getLineCount()) {
                this.hide();
            }
        }));
        this._register(dom.addStandardDisposableGenericMouseDownListener(this._domNode, e => {
            if (this.state.type !== 1 /* LightBulbState.Type.Showing */) {
                return;
            }
            const option = this._editor.getOption(64 /* EditorOption.lightbulb */).experimental.showAiIcon;
            if ((option === ShowAiIconMode.On || option === ShowAiIconMode.OnCode)
                && this.state.actions.allAIFixes
                && this.state.actions.validActions.length === 1) {
                const action = this.state.actions.validActions[0].action;
                if (action.command?.id) {
                    commandService.executeCommand(action.command.id, ...(action.command.arguments || []));
                    e.preventDefault();
                    return;
                }
            }
            // Make sure that focus / cursor location is not lost when clicking widget icon
            this._editor.focus();
            e.preventDefault();
            // a bit of extra work to make sure the menu
            // doesn't cover the line-text
            const { top, height } = dom.getDomNodePagePosition(this._domNode);
            const lineHeight = this._editor.getOption(66 /* EditorOption.lineHeight */);
            let pad = Math.floor(lineHeight / 3);
            if (this.state.widgetPosition.position !== null && this.state.widgetPosition.position.lineNumber < this.state.editorPosition.lineNumber) {
                pad += lineHeight;
            }
            this._onClick.fire({
                x: e.posx,
                y: top + height + pad,
                actions: this.state.actions,
                trigger: this.state.trigger,
            });
        }));
        this._register(dom.addDisposableListener(this._domNode, 'mouseenter', (e) => {
            if ((e.buttons & 1) !== 1) {
                return;
            }
            // mouse enters lightbulb while the primary/left button
            // is being pressed -> hide the lightbulb
            this.hide();
        }));
        this._register(this._editor.onDidChangeConfiguration(e => {
            // hide when told to do so
            if (e.hasChanged(64 /* EditorOption.lightbulb */)) {
                if (!this._editor.getOption(64 /* EditorOption.lightbulb */).enabled) {
                    this.hide();
                }
                this._updateLightBulbTitleAndIcon();
            }
        }));
        this._register(Event.runAndSubscribe(this._keybindingService.onDidUpdateKeybindings, () => {
            this._preferredKbLabel = this._keybindingService.lookupKeybinding(autoFixCommandId)?.getLabel() ?? undefined;
            this._quickFixKbLabel = this._keybindingService.lookupKeybinding(quickFixCommandId)?.getLabel() ?? undefined;
            this._updateLightBulbTitleAndIcon();
        }));
    }
    dispose() {
        super.dispose();
        this._editor.removeContentWidget(this);
    }
    getId() {
        return 'LightBulbWidget';
    }
    getDomNode() {
        return this._domNode;
    }
    getPosition() {
        return this._state.type === 1 /* LightBulbState.Type.Showing */ ? this._state.widgetPosition : null;
    }
    update(actions, trigger, atPosition) {
        if (actions.validActions.length <= 0) {
            return this.hide();
        }
        const options = this._editor.getOptions();
        if (!options.get(64 /* EditorOption.lightbulb */).enabled) {
            return this.hide();
        }
        const model = this._editor.getModel();
        if (!model) {
            return this.hide();
        }
        const { lineNumber, column } = model.validatePosition(atPosition);
        const tabSize = model.getOptions().tabSize;
        const fontInfo = options.get(50 /* EditorOption.fontInfo */);
        const lineContent = model.getLineContent(lineNumber);
        const indent = computeIndentLevel(lineContent, tabSize);
        const lineHasSpace = fontInfo.spaceWidth * indent > 22;
        const isFolded = (lineNumber) => {
            return lineNumber > 2 && this._editor.getTopForLineNumber(lineNumber) === this._editor.getTopForLineNumber(lineNumber - 1);
        };
        let effectiveLineNumber = lineNumber;
        if (!lineHasSpace) {
            if (lineNumber > 1 && !isFolded(lineNumber - 1)) {
                effectiveLineNumber -= 1;
            }
            else if (!isFolded(lineNumber + 1)) {
                effectiveLineNumber += 1;
            }
            else if (column * fontInfo.spaceWidth < 22) {
                // cannot show lightbulb above/below and showing
                // it inline would overlay the cursor...
                return this.hide();
            }
        }
        this.state = new LightBulbState.Showing(actions, trigger, atPosition, {
            position: { lineNumber: effectiveLineNumber, column: !!model.getLineContent(effectiveLineNumber).match(/^\S\s*$/) ? 2 : 1 },
            preference: LightBulbWidget_1._posPref
        });
        this._editor.layoutContentWidget(this);
    }
    hide() {
        if (this.state === LightBulbState.Hidden) {
            return;
        }
        this.state = LightBulbState.Hidden;
        this._editor.layoutContentWidget(this);
    }
    get state() { return this._state; }
    set state(value) {
        this._state = value;
        this._updateLightBulbTitleAndIcon();
    }
    _updateLightBulbTitleAndIcon() {
        this._domNode.classList.remove(...this._iconClasses);
        this._iconClasses = [];
        if (this.state.type !== 1 /* LightBulbState.Type.Showing */) {
            return;
        }
        const updateAutoFixLightbulbTitle = () => {
            if (this._preferredKbLabel) {
                this.title = nls.localizeWithPath('vs/editor/contrib/codeAction/browser/lightBulbWidget', 'preferredcodeActionWithKb', "Show Code Actions. Preferred Quick Fix Available ({0})", this._preferredKbLabel);
            }
        };
        const updateLightbulbTitle = () => {
            if (this._quickFixKbLabel) {
                this.title = nls.localizeWithPath('vs/editor/contrib/codeAction/browser/lightBulbWidget', 'codeActionWithKb', "Show Code Actions ({0})", this._quickFixKbLabel);
            }
            else {
                this.title = nls.localizeWithPath('vs/editor/contrib/codeAction/browser/lightBulbWidget', 'codeAction', "Show Code Actions");
            }
        };
        let icon;
        const option = this._editor.getOption(64 /* EditorOption.lightbulb */).experimental.showAiIcon;
        if (option === ShowAiIconMode.On || option === ShowAiIconMode.OnCode) {
            if (option === ShowAiIconMode.On && this.state.actions.allAIFixes) {
                icon = Codicon.sparkleFilled;
                if (this.state.actions.allAIFixes && this.state.actions.validActions.length === 1) {
                    if (this.state.actions.validActions[0].action.command?.id === `inlineChat.start`) {
                        const keybinding = this._keybindingService.lookupKeybinding('inlineChat.start')?.getLabel() ?? undefined;
                        this.title = keybinding ? nls.localizeWithPath('vs/editor/contrib/codeAction/browser/lightBulbWidget', 'codeActionStartInlineChatWithKb', 'Start Inline Chat ({0})', keybinding) : nls.localizeWithPath('vs/editor/contrib/codeAction/browser/lightBulbWidget', 'codeActionStartInlineChat', 'Start Inline Chat');
                    }
                    else {
                        this.title = nls.localizeWithPath('vs/editor/contrib/codeAction/browser/lightBulbWidget', 'codeActionTriggerAiAction', "Trigger AI Action");
                    }
                }
                else {
                    updateLightbulbTitle();
                }
            }
            else if (this.state.actions.hasAutoFix) {
                if (this.state.actions.hasAIFix) {
                    icon = Codicon.lightbulbSparkleAutofix;
                }
                else {
                    icon = Codicon.lightbulbAutofix;
                }
                updateAutoFixLightbulbTitle();
            }
            else if (this.state.actions.hasAIFix) {
                icon = Codicon.lightbulbSparkle;
                updateLightbulbTitle();
            }
            else {
                icon = Codicon.lightBulb;
                updateLightbulbTitle();
            }
        }
        else {
            if (this.state.actions.hasAutoFix) {
                icon = Codicon.lightbulbAutofix;
                updateAutoFixLightbulbTitle();
            }
            else {
                icon = Codicon.lightBulb;
                updateLightbulbTitle();
            }
        }
        this._iconClasses = ThemeIcon.asClassNameArray(icon);
        this._domNode.classList.add(...this._iconClasses);
    }
    set title(value) {
        this._domNode.title = value;
    }
};
LightBulbWidget.ID = 'editor.contrib.lightbulbWidget';
LightBulbWidget._posPref = [0 /* ContentWidgetPositionPreference.EXACT */];
LightBulbWidget = LightBulbWidget_1 = __decorate([
    __param(1, IKeybindingService),
    __param(2, ICommandService)
], LightBulbWidget);
export { LightBulbWidget };
