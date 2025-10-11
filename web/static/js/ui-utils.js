// ============================================================================
// MODULE: UI Utilities
// ============================================================================

class UIUtils {
    static showError(message, level = CONSTANTS.LOG_LEVEL.ERROR) {
        DOM.errorMessage.textContent = message;
        DOM.errorMessage.classList.add('show');
        
        setTimeout(() => {
            DOM.errorMessage.classList.remove('show');
        }, 5000);
    }
    
    static setElementCenter(element) {
        element.style.display = 'block';
        const width = element.offsetWidth;
        const height = element.offsetHeight;
        
        element.style.left = `${(window.innerWidth - width) / 2}px`;
        element.style.top = `${(window.innerHeight - height) / 2}px`;
    }
    
    static positionPopupNearNode(popup, node) {
        const renderedPosition = node.renderedPosition();
        const display = popup.style.display;
        
        popup.style.display = 'block';
        const width = popup.offsetWidth;
        const height = popup.offsetHeight;
        
        // Position horizontally
        if (window.innerWidth / 2 > renderedPosition.x) {
            popup.style.left = `${renderedPosition.x}px`;
        } else {
            popup.style.left = `${renderedPosition.x - width}px`;
        }
        
        // Position vertically
        if (window.innerHeight / 2 > renderedPosition.y) {
            popup.style.top = `${renderedPosition.y}px`;
        } else {
            popup.style.top = `${renderedPosition.y - height}px`;
        }
        
        popup.style.display = display;
    }
    
    static positionPanelNearParent(panel, parent) {
        const display = panel.style.display;
        panel.style.display = 'block';
        
        const pxLeft = parent.style.left;
        const spaceLeft = parseInt(pxLeft.substring(0, pxLeft.length - 2));
        const spaceRight = window.innerWidth - spaceLeft - parent.offsetWidth;
        
        if (spaceRight > spaceLeft) {
            panel.style.left = `${window.innerWidth - spaceRight}px`;
        } else {
            panel.style.left = `${spaceLeft - panel.offsetWidth}px`;
        }
        
        panel.style.top = parent.style.top;
        panel.style.display = display;
    }
    
    static registerToString(register, type) {
        if (register && type === CONSTANTS.REGISTER_TYPES.SEQUENTIAL) {
            return register.join(',');
        }
        return Object.keys(register).map(key => `${key}:${register[key]}`).join(',');
    }
}
