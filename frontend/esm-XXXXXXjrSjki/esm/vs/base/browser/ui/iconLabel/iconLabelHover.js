/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import * as dom from '../../dom.js';
import { TimeoutTimer } from '../../../common/async.js';
import { CancellationTokenSource } from '../../../common/cancellation.js';
import { isMarkdownString } from '../../../common/htmlContent.js';
import { stripIcons } from '../../../common/iconLabels.js';
import { DisposableStore } from '../../../common/lifecycle.js';
import { isFunction, isString } from '../../../common/types.js';
import { localizeWithPath } from '../../../../nls.js';
export function setupNativeHover(htmlElement, tooltip) {
    if (isString(tooltip)) {
        // Icons don't render in the native hover so we strip them out
        htmlElement.title = stripIcons(tooltip);
    }
    else if (tooltip?.markdownNotSupportedFallback) {
        htmlElement.title = tooltip.markdownNotSupportedFallback;
    }
    else {
        htmlElement.removeAttribute('title');
    }
}
class UpdatableHoverWidget {
    constructor(hoverDelegate, target, fadeInAnimation) {
        this.hoverDelegate = hoverDelegate;
        this.target = target;
        this.fadeInAnimation = fadeInAnimation;
    }
    async update(content, focus, options) {
        if (this._cancellationTokenSource) {
            // there's an computation ongoing, cancel it
            this._cancellationTokenSource.dispose(true);
            this._cancellationTokenSource = undefined;
        }
        if (this.isDisposed) {
            return;
        }
        let resolvedContent;
        if (content === undefined || isString(content) || content instanceof HTMLElement) {
            resolvedContent = content;
        }
        else if (!isFunction(content.markdown)) {
            resolvedContent = content.markdown ?? content.markdownNotSupportedFallback;
        }
        else {
            // compute the content, potentially long-running
            // show 'Loading' if no hover is up yet
            if (!this._hoverWidget) {
                this.show(localizeWithPath('vs/base/browser/ui/iconLabel/iconLabelHover', 'iconLabel.loading', "Loading..."), focus);
            }
            // compute the content
            this._cancellationTokenSource = new CancellationTokenSource();
            const token = this._cancellationTokenSource.token;
            resolvedContent = await content.markdown(token);
            if (resolvedContent === undefined) {
                resolvedContent = content.markdownNotSupportedFallback;
            }
            if (this.isDisposed || token.isCancellationRequested) {
                // either the widget has been closed in the meantime
                // or there has been a new call to `update`
                return;
            }
        }
        this.show(resolvedContent, focus, options);
    }
    show(content, focus, options) {
        const oldHoverWidget = this._hoverWidget;
        if (this.hasContent(content)) {
            const hoverOptions = {
                content,
                target: this.target,
                appearance: {
                    showPointer: this.hoverDelegate.placement === 'element',
                    skipFadeInAnimation: !this.fadeInAnimation || !!oldHoverWidget, // do not fade in if the hover is already showing
                },
                position: {
                    hoverPosition: 2 /* HoverPosition.BELOW */,
                },
                ...options
            };
            this._hoverWidget = this.hoverDelegate.showHover(hoverOptions, focus);
        }
        oldHoverWidget?.dispose();
    }
    hasContent(content) {
        if (!content) {
            return false;
        }
        if (isMarkdownString(content)) {
            return !!content.value;
        }
        return true;
    }
    get isDisposed() {
        return this._hoverWidget?.isDisposed;
    }
    dispose() {
        this._hoverWidget?.dispose();
        this._cancellationTokenSource?.dispose(true);
        this._cancellationTokenSource = undefined;
    }
}
export function setupCustomHover(hoverDelegate, htmlElement, content, options) {
    let hoverPreparation;
    let hoverWidget;
    const hideHover = (disposeWidget, disposePreparation) => {
        const hadHover = hoverWidget !== undefined;
        if (disposeWidget) {
            hoverWidget?.dispose();
            hoverWidget = undefined;
        }
        if (disposePreparation) {
            hoverPreparation?.dispose();
            hoverPreparation = undefined;
        }
        if (hadHover) {
            hoverDelegate.onDidHideHover?.();
        }
    };
    const triggerShowHover = (delay, focus, target) => {
        return new TimeoutTimer(async () => {
            if (!hoverWidget || hoverWidget.isDisposed) {
                hoverWidget = new UpdatableHoverWidget(hoverDelegate, target || htmlElement, delay > 0);
                await hoverWidget.update(content, focus, options);
            }
        }, delay);
    };
    const onMouseOver = () => {
        if (hoverPreparation) {
            return;
        }
        const toDispose = new DisposableStore();
        const onMouseLeave = (e) => hideHover(false, e.fromElement === htmlElement);
        toDispose.add(dom.addDisposableListener(htmlElement, dom.EventType.MOUSE_LEAVE, onMouseLeave, true));
        const onMouseDown = () => hideHover(true, true);
        toDispose.add(dom.addDisposableListener(htmlElement, dom.EventType.MOUSE_DOWN, onMouseDown, true));
        const target = {
            targetElements: [htmlElement],
            dispose: () => { }
        };
        if (hoverDelegate.placement === undefined || hoverDelegate.placement === 'mouse') {
            // track the mouse position
            const onMouseMove = (e) => {
                target.x = e.x + 10;
                if ((e.target instanceof HTMLElement) && e.target.classList.contains('action-label')) {
                    hideHover(true, true);
                }
            };
            toDispose.add(dom.addDisposableListener(htmlElement, dom.EventType.MOUSE_MOVE, onMouseMove, true));
        }
        toDispose.add(triggerShowHover(hoverDelegate.delay, false, target));
        hoverPreparation = toDispose;
    };
    const mouseOverDomEmitter = dom.addDisposableListener(htmlElement, dom.EventType.MOUSE_OVER, onMouseOver, true);
    const onFocus = () => {
        if (hoverPreparation) {
            return;
        }
        const target = {
            targetElements: [htmlElement],
            dispose: () => { }
        };
        const toDispose = new DisposableStore();
        const onBlur = () => hideHover(true, true);
        toDispose.add(dom.addDisposableListener(htmlElement, dom.EventType.BLUR, onBlur, true));
        toDispose.add(triggerShowHover(hoverDelegate.delay, false, target));
        hoverPreparation = toDispose;
    };
    const focusDomEmitter = dom.addDisposableListener(htmlElement, dom.EventType.FOCUS, onFocus, true);
    const hover = {
        show: focus => {
            hideHover(false, true); // terminate a ongoing mouse over preparation
            triggerShowHover(0, focus); // show hover immediately
        },
        hide: () => {
            hideHover(true, true);
        },
        update: async (newContent, hoverOptions) => {
            content = newContent;
            await hoverWidget?.update(content, undefined, hoverOptions);
        },
        dispose: () => {
            mouseOverDomEmitter.dispose();
            focusDomEmitter.dispose();
            hideHover(true, true);
        }
    };
    return hover;
}
