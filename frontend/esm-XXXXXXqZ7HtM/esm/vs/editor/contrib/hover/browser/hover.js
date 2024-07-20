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
var ModesHoverController_1;
import { KeyChord } from '../../../../base/common/keyCodes.js';
import { Disposable, DisposableStore } from '../../../../base/common/lifecycle.js';
import { EditorAction, registerEditorAction, registerEditorContribution } from '../../../browser/editorExtensions.js';
import { Range } from '../../../common/core/range.js';
import { EditorContextKeys } from '../../../common/editorContextKeys.js';
import { ILanguageService } from '../../../common/languages/language.js';
import { GotoDefinitionAtPositionEditorContribution } from '../../gotoSymbol/browser/link/goToDefinitionAtPosition.js';
import { ContentHoverWidget, ContentHoverController } from './contentHover.js';
import { MarginHoverWidget } from './marginHover.js';
import { IInstantiationService } from '../../../../platform/instantiation/common/instantiation.js';
import { IOpenerService } from '../../../../platform/opener/common/opener.js';
import { editorHoverBorder } from '../../../../platform/theme/common/colorRegistry.js';
import { registerThemingParticipant } from '../../../../platform/theme/common/themeService.js';
import { HoverParticipantRegistry } from './hoverTypes.js';
import { MarkdownHoverParticipant } from './markdownHoverParticipant.js';
import { MarkerHoverParticipant } from './markerHoverParticipant.js';
import { InlineSuggestionHintsContentWidget } from '../../inlineCompletions/browser/inlineCompletionsHintsWidget.js';
import { IKeybindingService } from '../../../../platform/keybinding/common/keybinding.js';
import * as nls from '../../../../nls.js';
import './hover.css';
import { RunOnceScheduler } from '../../../../base/common/async.js';
// sticky hover widget which doesn't disappear on focus out and such
const _sticky = false;
let ModesHoverController = ModesHoverController_1 = class ModesHoverController extends Disposable {
    getWidgetContent() { return this._contentWidget?.getWidgetContent(); }
    static get(editor) {
        return editor.getContribution(ModesHoverController_1.ID);
    }
    constructor(_editor, _instantiationService, _openerService, _languageService, _keybindingService) {
        super();
        this._editor = _editor;
        this._instantiationService = _instantiationService;
        this._openerService = _openerService;
        this._languageService = _languageService;
        this._keybindingService = _keybindingService;
        this._toUnhook = new DisposableStore();
        this._hoverActivatedByColorDecoratorClick = false;
        this._isMouseDown = false;
        this._hoverClicked = false;
        this._contentWidget = null;
        this._glyphWidget = null;
        this._reactToEditorMouseMoveRunner = this._register(new RunOnceScheduler(() => this._reactToEditorMouseMove(this._mouseMoveEvent), 0));
        this._hookEvents();
        this._register(this._editor.onDidChangeConfiguration((e) => {
            if (e.hasChanged(60 /* EditorOption.hover */)) {
                this._unhookEvents();
                this._hookEvents();
            }
        }));
    }
    _hookEvents() {
        const hoverOpts = this._editor.getOption(60 /* EditorOption.hover */);
        this._isHoverEnabled = hoverOpts.enabled;
        this._isHoverSticky = hoverOpts.sticky;
        this._hidingDelay = hoverOpts.hidingDelay;
        if (this._isHoverEnabled) {
            this._toUnhook.add(this._editor.onMouseDown((e) => this._onEditorMouseDown(e)));
            this._toUnhook.add(this._editor.onMouseUp((e) => this._onEditorMouseUp(e)));
            this._toUnhook.add(this._editor.onMouseMove((e) => this._onEditorMouseMove(e)));
            this._toUnhook.add(this._editor.onKeyDown((e) => this._onKeyDown(e)));
        }
        else {
            this._toUnhook.add(this._editor.onMouseMove((e) => this._onEditorMouseMove(e)));
            this._toUnhook.add(this._editor.onKeyDown((e) => this._onKeyDown(e)));
        }
        this._toUnhook.add(this._editor.onMouseLeave((e) => this._onEditorMouseLeave(e)));
        this._toUnhook.add(this._editor.onDidChangeModel(() => {
            this._cancelScheduler();
            this._hideWidgets();
        }));
        this._toUnhook.add(this._editor.onDidChangeModelContent(() => this._cancelScheduler()));
        this._toUnhook.add(this._editor.onDidScrollChange((e) => this._onEditorScrollChanged(e)));
    }
    _cancelScheduler() {
        this._mouseMoveEvent = undefined;
        this._reactToEditorMouseMoveRunner.cancel();
    }
    _unhookEvents() {
        this._toUnhook.clear();
    }
    _onEditorScrollChanged(e) {
        if (e.scrollTopChanged || e.scrollLeftChanged) {
            this._hideWidgets();
        }
    }
    _onEditorMouseDown(mouseEvent) {
        this._isMouseDown = true;
        const target = mouseEvent.target;
        if (target.type === 9 /* MouseTargetType.CONTENT_WIDGET */ && target.detail === ContentHoverWidget.ID) {
            this._hoverClicked = true;
            // mouse down on top of content hover widget
            return;
        }
        if (target.type === 12 /* MouseTargetType.OVERLAY_WIDGET */ && target.detail === MarginHoverWidget.ID) {
            // mouse down on top of overlay hover widget
            return;
        }
        if (target.type !== 12 /* MouseTargetType.OVERLAY_WIDGET */) {
            this._hoverClicked = false;
        }
        if (!this._contentWidget?.widget.isResizing) {
            this._hideWidgets();
        }
    }
    _onEditorMouseUp(mouseEvent) {
        this._isMouseDown = false;
    }
    _onEditorMouseLeave(mouseEvent) {
        this._cancelScheduler();
        const targetEm = (mouseEvent.event.browserEvent.relatedTarget);
        if (this._contentWidget?.widget.isResizing || this._contentWidget?.containsNode(targetEm)) {
            // When the content widget is resizing
            // when the mouse is inside hover widget
            return;
        }
        if (!_sticky) {
            this._hideWidgets();
        }
    }
    _isMouseOverWidget(mouseEvent) {
        const target = mouseEvent.target;
        if (this._isHoverSticky
            && target.type === 9 /* MouseTargetType.CONTENT_WIDGET */
            && target.detail === ContentHoverWidget.ID) {
            // mouse moved on top of content hover widget
            return true;
        }
        if (this._isHoverSticky
            && this._contentWidget?.containsNode(mouseEvent.event.browserEvent.view?.document.activeElement)
            && !mouseEvent.event.browserEvent.view?.getSelection()?.isCollapsed) {
            // selected text within content hover widget
            return true;
        }
        if (!this._isHoverSticky
            && target.type === 9 /* MouseTargetType.CONTENT_WIDGET */
            && target.detail === ContentHoverWidget.ID
            && this._contentWidget?.isColorPickerVisible) {
            // though the hover is not sticky, the color picker needs to.
            return true;
        }
        if (this._isHoverSticky
            && target.type === 12 /* MouseTargetType.OVERLAY_WIDGET */
            && target.detail === MarginHoverWidget.ID) {
            // mouse moved on top of overlay hover widget
            return true;
        }
        return false;
    }
    _onEditorMouseMove(mouseEvent) {
        this._mouseMoveEvent = mouseEvent;
        if (this._contentWidget?.isFocused || this._contentWidget?.isResizing) {
            return;
        }
        if (this._isMouseDown && this._hoverClicked) {
            return;
        }
        if (this._isHoverSticky && this._contentWidget?.isVisibleFromKeyboard) {
            // Sticky mode is on and the hover has been shown via keyboard
            // so moving the mouse has no effect
            return;
        }
        const mouseIsOverWidget = this._isMouseOverWidget(mouseEvent);
        // If the mouse is over the widget and the hiding timeout is defined, then cancel it
        if (mouseIsOverWidget) {
            this._reactToEditorMouseMoveRunner.cancel();
            return;
        }
        // If the mouse is not over the widget, and if sticky is on,
        // then give it a grace period before reacting to the mouse event
        if (this._contentWidget?.isVisible && this._isHoverSticky && this._hidingDelay > 0) {
            if (!this._reactToEditorMouseMoveRunner.isScheduled()) {
                this._reactToEditorMouseMoveRunner.schedule(this._hidingDelay);
            }
            return;
        }
        this._reactToEditorMouseMove(mouseEvent);
    }
    _reactToEditorMouseMove(mouseEvent) {
        if (!mouseEvent) {
            return;
        }
        const target = mouseEvent.target;
        const mouseOnDecorator = target.element?.classList.contains('colorpicker-color-decoration');
        const decoratorActivatedOn = this._editor.getOption(146 /* EditorOption.colorDecoratorsActivatedOn */);
        if ((mouseOnDecorator && ((decoratorActivatedOn === 'click' && !this._hoverActivatedByColorDecoratorClick) ||
            (decoratorActivatedOn === 'hover' && !this._isHoverEnabled && !_sticky) ||
            (decoratorActivatedOn === 'clickAndHover' && !this._isHoverEnabled && !this._hoverActivatedByColorDecoratorClick)))
            || !mouseOnDecorator && !this._isHoverEnabled && !this._hoverActivatedByColorDecoratorClick) {
            this._hideWidgets();
            return;
        }
        const contentWidget = this._getOrCreateContentWidget();
        if (contentWidget.maybeShowAt(mouseEvent)) {
            this._glyphWidget?.hide();
            return;
        }
        if (target.type === 2 /* MouseTargetType.GUTTER_GLYPH_MARGIN */ && target.position) {
            this._contentWidget?.hide();
            if (!this._glyphWidget) {
                this._glyphWidget = new MarginHoverWidget(this._editor, this._languageService, this._openerService);
            }
            this._glyphWidget.startShowingAt(target.position.lineNumber);
            return;
        }
        if (_sticky) {
            return;
        }
        this._hideWidgets();
    }
    _onKeyDown(e) {
        if (!this._editor.hasModel()) {
            return;
        }
        const resolvedKeyboardEvent = this._keybindingService.softDispatch(e, this._editor.getDomNode());
        // If the beginning of a multi-chord keybinding is pressed, or the command aims to focus the hover, set the variable to true, otherwise false
        const mightTriggerFocus = (resolvedKeyboardEvent.kind === 1 /* ResultKind.MoreChordsNeeded */ || (resolvedKeyboardEvent.kind === 2 /* ResultKind.KbFound */ && resolvedKeyboardEvent.commandId === 'editor.action.showHover' && this._contentWidget?.isVisible));
        if (e.keyCode !== 5 /* KeyCode.Ctrl */ && e.keyCode !== 6 /* KeyCode.Alt */ && e.keyCode !== 57 /* KeyCode.Meta */ && e.keyCode !== 4 /* KeyCode.Shift */
            && !mightTriggerFocus) {
            // Do not hide hover when a modifier key is pressed
            this._hideWidgets();
        }
    }
    _hideWidgets() {
        if (_sticky) {
            return;
        }
        if ((this._isMouseDown && this._hoverClicked && this._contentWidget?.isColorPickerVisible) || InlineSuggestionHintsContentWidget.dropDownVisible) {
            return;
        }
        this._hoverActivatedByColorDecoratorClick = false;
        this._hoverClicked = false;
        this._glyphWidget?.hide();
        this._contentWidget?.hide();
    }
    _getOrCreateContentWidget() {
        if (!this._contentWidget) {
            this._contentWidget = this._instantiationService.createInstance(ContentHoverController, this._editor);
        }
        return this._contentWidget;
    }
    showContentHover(range, mode, source, focus, activatedByColorDecoratorClick = false) {
        this._hoverActivatedByColorDecoratorClick = activatedByColorDecoratorClick;
        this._getOrCreateContentWidget().startShowingAtRange(range, mode, source, focus);
    }
    focus() {
        this._contentWidget?.focus();
    }
    scrollUp() {
        this._contentWidget?.scrollUp();
    }
    scrollDown() {
        this._contentWidget?.scrollDown();
    }
    scrollLeft() {
        this._contentWidget?.scrollLeft();
    }
    scrollRight() {
        this._contentWidget?.scrollRight();
    }
    pageUp() {
        this._contentWidget?.pageUp();
    }
    pageDown() {
        this._contentWidget?.pageDown();
    }
    goToTop() {
        this._contentWidget?.goToTop();
    }
    goToBottom() {
        this._contentWidget?.goToBottom();
    }
    get isColorPickerVisible() {
        return this._contentWidget?.isColorPickerVisible;
    }
    get isHoverVisible() {
        return this._contentWidget?.isVisible;
    }
    dispose() {
        super.dispose();
        this._unhookEvents();
        this._toUnhook.dispose();
        this._glyphWidget?.dispose();
        this._contentWidget?.dispose();
    }
};
ModesHoverController.ID = 'editor.contrib.hover';
ModesHoverController = ModesHoverController_1 = __decorate([
    __param(1, IInstantiationService),
    __param(2, IOpenerService),
    __param(3, ILanguageService),
    __param(4, IKeybindingService)
], ModesHoverController);
export { ModesHoverController };
var HoverFocusBehavior;
(function (HoverFocusBehavior) {
    HoverFocusBehavior["NoAutoFocus"] = "noAutoFocus";
    HoverFocusBehavior["FocusIfVisible"] = "focusIfVisible";
    HoverFocusBehavior["AutoFocusImmediately"] = "autoFocusImmediately";
})(HoverFocusBehavior || (HoverFocusBehavior = {}));
class ShowOrFocusHoverAction extends EditorAction {
    constructor() {
        super({
            id: 'editor.action.showHover',
            label: nls.localizeWithPath('vs/editor/contrib/hover/browser/hover', {
                key: 'showOrFocusHover',
                comment: [
                    'Label for action that will trigger the showing/focusing of a hover in the editor.',
                    'If the hover is not visible, it will show the hover.',
                    'This allows for users to show the hover without using the mouse.'
                ]
            }, "Show or Focus Hover"),
            metadata: {
                description: `Show or Focus Hover`,
                args: [{
                        name: 'args',
                        schema: {
                            type: 'object',
                            properties: {
                                'focus': {
                                    description: 'Controls if and when the hover should take focus upon being triggered by this action.',
                                    enum: [HoverFocusBehavior.NoAutoFocus, HoverFocusBehavior.FocusIfVisible, HoverFocusBehavior.AutoFocusImmediately],
                                    enumDescriptions: [
                                        nls.localizeWithPath('vs/editor/contrib/hover/browser/hover', 'showOrFocusHover.focus.noAutoFocus', 'The hover will not automatically take focus.'),
                                        nls.localizeWithPath('vs/editor/contrib/hover/browser/hover', 'showOrFocusHover.focus.focusIfVisible', 'The hover will take focus only if it is already visible.'),
                                        nls.localizeWithPath('vs/editor/contrib/hover/browser/hover', 'showOrFocusHover.focus.autoFocusImmediately', 'The hover will automatically take focus when it appears.'),
                                    ],
                                    default: HoverFocusBehavior.FocusIfVisible,
                                }
                            },
                        }
                    }]
            },
            alias: 'Show or Focus Hover',
            precondition: undefined,
            kbOpts: {
                kbExpr: EditorContextKeys.editorTextFocus,
                primary: KeyChord(2048 /* KeyMod.CtrlCmd */ | 41 /* KeyCode.KeyK */, 2048 /* KeyMod.CtrlCmd */ | 39 /* KeyCode.KeyI */),
                weight: 100 /* KeybindingWeight.EditorContrib */
            }
        });
    }
    run(accessor, editor, args) {
        if (!editor.hasModel()) {
            return;
        }
        const controller = ModesHoverController.get(editor);
        if (!controller) {
            return;
        }
        const focusArgument = args?.focus;
        let focusOption = HoverFocusBehavior.FocusIfVisible;
        if (focusArgument in HoverFocusBehavior) {
            focusOption = focusArgument;
        }
        else if (typeof focusArgument === 'boolean' && focusArgument) {
            focusOption = HoverFocusBehavior.AutoFocusImmediately;
        }
        const showContentHover = (focus) => {
            const position = editor.getPosition();
            const range = new Range(position.lineNumber, position.column, position.lineNumber, position.column);
            controller.showContentHover(range, 1 /* HoverStartMode.Immediate */, 1 /* HoverStartSource.Keyboard */, focus);
        };
        const accessibilitySupportEnabled = editor.getOption(2 /* EditorOption.accessibilitySupport */) === 2 /* AccessibilitySupport.Enabled */;
        if (controller.isHoverVisible) {
            if (focusOption !== HoverFocusBehavior.NoAutoFocus) {
                controller.focus();
            }
            else {
                showContentHover(accessibilitySupportEnabled);
            }
        }
        else {
            showContentHover(accessibilitySupportEnabled || focusOption === HoverFocusBehavior.AutoFocusImmediately);
        }
    }
}
class ShowDefinitionPreviewHoverAction extends EditorAction {
    constructor() {
        super({
            id: 'editor.action.showDefinitionPreviewHover',
            label: nls.localizeWithPath('vs/editor/contrib/hover/browser/hover', {
                key: 'showDefinitionPreviewHover',
                comment: [
                    'Label for action that will trigger the showing of definition preview hover in the editor.',
                    'This allows for users to show the definition preview hover without using the mouse.'
                ]
            }, "Show Definition Preview Hover"),
            alias: 'Show Definition Preview Hover',
            precondition: undefined
        });
    }
    run(accessor, editor) {
        const controller = ModesHoverController.get(editor);
        if (!controller) {
            return;
        }
        const position = editor.getPosition();
        if (!position) {
            return;
        }
        const range = new Range(position.lineNumber, position.column, position.lineNumber, position.column);
        const goto = GotoDefinitionAtPositionEditorContribution.get(editor);
        if (!goto) {
            return;
        }
        const promise = goto.startFindDefinitionFromCursor(position);
        promise.then(() => {
            controller.showContentHover(range, 1 /* HoverStartMode.Immediate */, 1 /* HoverStartSource.Keyboard */, true);
        });
    }
}
class ScrollUpHoverAction extends EditorAction {
    constructor() {
        super({
            id: 'editor.action.scrollUpHover',
            label: nls.localizeWithPath('vs/editor/contrib/hover/browser/hover', {
                key: 'scrollUpHover',
                comment: [
                    'Action that allows to scroll up in the hover widget with the up arrow when the hover widget is focused.'
                ]
            }, "Scroll Up Hover"),
            alias: 'Scroll Up Hover',
            precondition: EditorContextKeys.hoverFocused,
            kbOpts: {
                kbExpr: EditorContextKeys.hoverFocused,
                primary: 16 /* KeyCode.UpArrow */,
                weight: 100 /* KeybindingWeight.EditorContrib */
            }
        });
    }
    run(accessor, editor) {
        const controller = ModesHoverController.get(editor);
        if (!controller) {
            return;
        }
        controller.scrollUp();
    }
}
class ScrollDownHoverAction extends EditorAction {
    constructor() {
        super({
            id: 'editor.action.scrollDownHover',
            label: nls.localizeWithPath('vs/editor/contrib/hover/browser/hover', {
                key: 'scrollDownHover',
                comment: [
                    'Action that allows to scroll down in the hover widget with the up arrow when the hover widget is focused.'
                ]
            }, "Scroll Down Hover"),
            alias: 'Scroll Down Hover',
            precondition: EditorContextKeys.hoverFocused,
            kbOpts: {
                kbExpr: EditorContextKeys.hoverFocused,
                primary: 18 /* KeyCode.DownArrow */,
                weight: 100 /* KeybindingWeight.EditorContrib */
            }
        });
    }
    run(accessor, editor) {
        const controller = ModesHoverController.get(editor);
        if (!controller) {
            return;
        }
        controller.scrollDown();
    }
}
class ScrollLeftHoverAction extends EditorAction {
    constructor() {
        super({
            id: 'editor.action.scrollLeftHover',
            label: nls.localizeWithPath('vs/editor/contrib/hover/browser/hover', {
                key: 'scrollLeftHover',
                comment: [
                    'Action that allows to scroll left in the hover widget with the left arrow when the hover widget is focused.'
                ]
            }, "Scroll Left Hover"),
            alias: 'Scroll Left Hover',
            precondition: EditorContextKeys.hoverFocused,
            kbOpts: {
                kbExpr: EditorContextKeys.hoverFocused,
                primary: 15 /* KeyCode.LeftArrow */,
                weight: 100 /* KeybindingWeight.EditorContrib */
            }
        });
    }
    run(accessor, editor) {
        const controller = ModesHoverController.get(editor);
        if (!controller) {
            return;
        }
        controller.scrollLeft();
    }
}
class ScrollRightHoverAction extends EditorAction {
    constructor() {
        super({
            id: 'editor.action.scrollRightHover',
            label: nls.localizeWithPath('vs/editor/contrib/hover/browser/hover', {
                key: 'scrollRightHover',
                comment: [
                    'Action that allows to scroll right in the hover widget with the right arrow when the hover widget is focused.'
                ]
            }, "Scroll Right Hover"),
            alias: 'Scroll Right Hover',
            precondition: EditorContextKeys.hoverFocused,
            kbOpts: {
                kbExpr: EditorContextKeys.hoverFocused,
                primary: 17 /* KeyCode.RightArrow */,
                weight: 100 /* KeybindingWeight.EditorContrib */
            }
        });
    }
    run(accessor, editor) {
        const controller = ModesHoverController.get(editor);
        if (!controller) {
            return;
        }
        controller.scrollRight();
    }
}
class PageUpHoverAction extends EditorAction {
    constructor() {
        super({
            id: 'editor.action.pageUpHover',
            label: nls.localizeWithPath('vs/editor/contrib/hover/browser/hover', {
                key: 'pageUpHover',
                comment: [
                    'Action that allows to page up in the hover widget with the page up command when the hover widget is focused.'
                ]
            }, "Page Up Hover"),
            alias: 'Page Up Hover',
            precondition: EditorContextKeys.hoverFocused,
            kbOpts: {
                kbExpr: EditorContextKeys.hoverFocused,
                primary: 11 /* KeyCode.PageUp */,
                secondary: [512 /* KeyMod.Alt */ | 16 /* KeyCode.UpArrow */],
                weight: 100 /* KeybindingWeight.EditorContrib */
            }
        });
    }
    run(accessor, editor) {
        const controller = ModesHoverController.get(editor);
        if (!controller) {
            return;
        }
        controller.pageUp();
    }
}
class PageDownHoverAction extends EditorAction {
    constructor() {
        super({
            id: 'editor.action.pageDownHover',
            label: nls.localizeWithPath('vs/editor/contrib/hover/browser/hover', {
                key: 'pageDownHover',
                comment: [
                    'Action that allows to page down in the hover widget with the page down command when the hover widget is focused.'
                ]
            }, "Page Down Hover"),
            alias: 'Page Down Hover',
            precondition: EditorContextKeys.hoverFocused,
            kbOpts: {
                kbExpr: EditorContextKeys.hoverFocused,
                primary: 12 /* KeyCode.PageDown */,
                secondary: [512 /* KeyMod.Alt */ | 18 /* KeyCode.DownArrow */],
                weight: 100 /* KeybindingWeight.EditorContrib */
            }
        });
    }
    run(accessor, editor) {
        const controller = ModesHoverController.get(editor);
        if (!controller) {
            return;
        }
        controller.pageDown();
    }
}
class GoToTopHoverAction extends EditorAction {
    constructor() {
        super({
            id: 'editor.action.goToTopHover',
            label: nls.localizeWithPath('vs/editor/contrib/hover/browser/hover', {
                key: 'goToTopHover',
                comment: [
                    'Action that allows to go to the top of the hover widget with the home command when the hover widget is focused.'
                ]
            }, "Go To Top Hover"),
            alias: 'Go To Bottom Hover',
            precondition: EditorContextKeys.hoverFocused,
            kbOpts: {
                kbExpr: EditorContextKeys.hoverFocused,
                primary: 14 /* KeyCode.Home */,
                secondary: [2048 /* KeyMod.CtrlCmd */ | 16 /* KeyCode.UpArrow */],
                weight: 100 /* KeybindingWeight.EditorContrib */
            }
        });
    }
    run(accessor, editor) {
        const controller = ModesHoverController.get(editor);
        if (!controller) {
            return;
        }
        controller.goToTop();
    }
}
class GoToBottomHoverAction extends EditorAction {
    constructor() {
        super({
            id: 'editor.action.goToBottomHover',
            label: nls.localizeWithPath('vs/editor/contrib/hover/browser/hover', {
                key: 'goToBottomHover',
                comment: [
                    'Action that allows to go to the bottom in the hover widget with the end command when the hover widget is focused.'
                ]
            }, "Go To Bottom Hover"),
            alias: 'Go To Bottom Hover',
            precondition: EditorContextKeys.hoverFocused,
            kbOpts: {
                kbExpr: EditorContextKeys.hoverFocused,
                primary: 13 /* KeyCode.End */,
                secondary: [2048 /* KeyMod.CtrlCmd */ | 18 /* KeyCode.DownArrow */],
                weight: 100 /* KeybindingWeight.EditorContrib */
            }
        });
    }
    run(accessor, editor) {
        const controller = ModesHoverController.get(editor);
        if (!controller) {
            return;
        }
        controller.goToBottom();
    }
}
registerEditorContribution(ModesHoverController.ID, ModesHoverController, 2 /* EditorContributionInstantiation.BeforeFirstInteraction */);
registerEditorAction(ShowOrFocusHoverAction);
registerEditorAction(ShowDefinitionPreviewHoverAction);
registerEditorAction(ScrollUpHoverAction);
registerEditorAction(ScrollDownHoverAction);
registerEditorAction(ScrollLeftHoverAction);
registerEditorAction(ScrollRightHoverAction);
registerEditorAction(PageUpHoverAction);
registerEditorAction(PageDownHoverAction);
registerEditorAction(GoToTopHoverAction);
registerEditorAction(GoToBottomHoverAction);
HoverParticipantRegistry.register(MarkdownHoverParticipant);
HoverParticipantRegistry.register(MarkerHoverParticipant);
// theming
registerThemingParticipant((theme, collector) => {
    const hoverBorder = theme.getColor(editorHoverBorder);
    if (hoverBorder) {
        collector.addRule(`.monaco-editor .monaco-hover .hover-row:not(:first-child):not(:empty) { border-top: 1px solid ${hoverBorder.transparent(0.5)}; }`);
        collector.addRule(`.monaco-editor .monaco-hover hr { border-top: 1px solid ${hoverBorder.transparent(0.5)}; }`);
        collector.addRule(`.monaco-editor .monaco-hover hr { border-bottom: 0px solid ${hoverBorder.transparent(0.5)}; }`);
    }
});
