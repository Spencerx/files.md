let scrollInterval;
let isSelecting = false;
const resistance = 120; // the more it is, the slower scrolling is
const scrollMargin = 30;
let lastMousePos = null;

function startAutoScroll(direction) {
    if (scrollInterval) return; // Already scrolling

    scrollInterval = setInterval(() => {
        const scrollInfo = editor.getScrollInfo();
        const lineHeight = editor.defaultTextHeight();

        if (direction === 'up') {
            editor.scrollTo(null, Math.max(0, scrollInfo.top - lineHeight));
        } else if (direction === 'down') {
            const maxScroll = scrollInfo.height - scrollInfo.clientHeight;
            editor.scrollTo(null, Math.min(maxScroll, scrollInfo.top + lineHeight));
        }

        // Extend selection to follow auto-scroll
        if (lastMousePos && isSelecting) {
            const pos = editor.coordsChar(lastMousePos);
            if (pos) {
                const currentSelection = editor.getSelection();
                const anchor = editor.getCursor('anchor');
                editor.setSelection(anchor, pos);
            }
        }
    }, resistance);
}

function stopAutoScroll() {
    if (scrollInterval) {
        clearInterval(scrollInterval);
        scrollInterval = null;
    }
}

function checkAutoScroll(e) {
    if (!isSelecting) return;

    // Store the mouse position for selection extension during auto-scroll
    lastMousePos = {left: e.clientX, top: e.clientY};

    const editorRect = editor.getWrapperElement().getBoundingClientRect();
    const mouseY = e.clientY;

    // Check if mouse is near top or bottom of editor
    if (mouseY < editorRect.top + scrollMargin) {
        startAutoScroll('up');
    } else if (mouseY > editorRect.bottom - scrollMargin) {
        startAutoScroll('down');
    } else {
        stopAutoScroll();
    }
}

function initAutoscroll(editor) {
    editor.getWrapperElement().addEventListener("mousedown", function (e) {
        if (e.target.closest('.CodeMirror')) {
            isSelecting = true;
            // Check immediately on mousedown in case we start at the edge
            setTimeout(() => checkAutoScroll(e), 0);
        }
    }, true);
    document.addEventListener("mouseup", function () {
        isSelecting = false;
        lastMousePos = null;
        stopAutoScroll();
    });
    editor.getWrapperElement().addEventListener("mousemove", checkAutoScroll);
    // Stop scrolling when mouse leaves editor
    editor.getWrapperElement().addEventListener("mouseleave", function () {
        lastMousePos = null;
        stopAutoScroll();
    });

    // Additional: Check for auto-scroll during selection changes
    // This catches cases where the selection extends to edges programmatically
    editor.on('beforeSelectionChange', function (cm, obj) {
        if (isSelecting) {
            // Small delay to let the selection update, then check mouse position
            setTimeout(() => {
                const mouseEvent = window.lastMouseEvent;
                if (mouseEvent) {
                    checkAutoScroll(mouseEvent);
                }
            }, 0);
        }
    });
    // Track the last mouse event for reference
    document.addEventListener('mousemove', function (e) {
        window.lastMouseEvent = e;
    });
    // Track the last mouse event for reference
    document.addEventListener('mousemove', function (e) {
        window.lastMouseEvent = e;
    });
}
