// ============================================================================
// MODULE: Popup Position Tracker (FIXED - Centrado Garantizado)
// ============================================================================

class PopupTracker {
    constructor() {
        this.activePopup = null;
        this.activeElement = null;
        this.secondaryPanels = [];
        this.updateHandler = null;
    }
    
    attachToElement(popup, element, positionStrategy = 'smart') {
        this.detach();
        
        this.activePopup = popup;
        this.activeElement = element;
        this.secondaryPanels = [];
        
        // CRÍTICO: Usar requestAnimationFrame para posicionar después del render
        requestAnimationFrame(() => {
            this.updatePosition(positionStrategy);
        });
        
        if (element.isNode()) {
            this.updateHandler = () => {
                this.updatePosition(positionStrategy);
                this.updateSecondaryPanels();
            };
            
            element.on('position', this.updateHandler);
            element.on('drag', this.updateHandler);
            cy.on('pan', this.updateHandler);
            cy.on('zoom', this.updateHandler);
            cy.on('viewport', this.updateHandler);
        }
    }
    
    addSecondaryPanel(panel) {
        if (!this.secondaryPanels.includes(panel)) {
            this.secondaryPanels.push(panel);
            this.positionPanel(panel, this.activePopup);
        }
    }
    
    removeSecondaryPanel(panel) {
        const index = this.secondaryPanels.indexOf(panel);
        if (index > -1) {
            this.secondaryPanels.splice(index, 1);
        }
    }
    
    updateSecondaryPanels() {
        if (!this.activePopup) return;
        
        this.secondaryPanels.forEach(panel => {
            if (panel.style.display === 'block') {
                this.positionPanel(panel, this.activePopup);
            }
        });
    }
    
    detach() {
        if (this.updateHandler && this.activeElement) {
            if (this.activeElement.isNode()) {
                this.activeElement.removeListener('position', this.updateHandler);
                this.activeElement.removeListener('drag', this.updateHandler);
            }
            cy.off('pan', this.updateHandler);
            cy.off('zoom', this.updateHandler);
            cy.off('viewport', this.updateHandler);
        }
        
        this.activePopup = null;
        this.activeElement = null;
        this.secondaryPanels = [];
        this.updateHandler = null;
    }
    
    updatePosition(strategy = 'smart') {
        if (!this.activePopup || !this.activeElement) return;
        
        if (this.activeElement.isNode()) {
            const renderedPos = this.activeElement.renderedPosition();
            const bounds = this.activeElement.renderedBoundingBox();
            
            if (strategy === 'smart') {
                this.positionSmart(renderedPos, bounds);
            } else if (strategy === 'center') {
                this.positionCenter();
            }
        } else if (this.activeElement.isEdge()) {
            // Para edges SIEMPRE centrar
            this.positionCenter();
        }
    }
    
    positionSmart(renderedPos, bounds) {
        const popup = this.activePopup;
        const originalDisplay = popup.style.display;
        popup.style.display = 'block';
        
        const popupWidth = popup.offsetWidth;
        const popupHeight = popup.offsetHeight;
        const padding = 15;
        
        const spaceRight = window.innerWidth - bounds.x2;
        const spaceLeft = bounds.x1;
        const spaceBottom = window.innerHeight - bounds.y2;
        const spaceTop = bounds.y1;
        
        let left, top;
        
        if (spaceRight >= popupWidth + padding) {
            left = bounds.x2 + padding;
        } else if (spaceLeft >= popupWidth + padding) {
            left = bounds.x1 - popupWidth - padding;
        } else if (renderedPos.x > window.innerWidth / 2) {
            left = Math.max(padding, bounds.x1 - popupWidth - padding);
        } else {
            left = Math.min(window.innerWidth - popupWidth - padding, bounds.x2 + padding);
        }
        
        if (spaceBottom >= popupHeight + padding) {
            top = bounds.y1;
        } else if (spaceTop >= popupHeight + padding) {
            top = bounds.y2 - popupHeight;
        } else {
            top = Math.max(padding, (window.innerHeight - popupHeight) / 2);
        }
        
        left = Math.max(padding, Math.min(left, window.innerWidth - popupWidth - padding));
        top = Math.max(padding, Math.min(top, window.innerHeight - popupHeight - padding));
        
        popup.style.left = `${left}px`;
        popup.style.top = `${top}px`;
        popup.style.display = originalDisplay;
    }
    
    positionCenter() {
        const popup = this.activePopup;
        if (!popup) return;
        
        // Asegurar que el popup esté visible para medir
        const wasHidden = popup.style.display === 'none';
        if (wasHidden) {
            popup.style.display = 'block';
        }
        
        // Forzar reflow para obtener dimensiones exactas
        popup.offsetHeight; // Trigger reflow
        
        const width = popup.offsetWidth;
        const height = popup.offsetHeight;
        
        const padding = 20;
        const left = Math.max(padding, (window.innerWidth - width) / 2);
        const top = Math.max(padding, (window.innerHeight - height) / 2);
        
        popup.style.left = `${left}px`;
        popup.style.top = `${top}px`;
        
        // No ocultar de nuevo, ya debería estar visible
    }
    
    positionPanel(panel, parentPopup) {
        const originalDisplay = panel.style.display;
        panel.style.display = 'block';
        
        const parentRect = parentPopup.getBoundingClientRect();
        const panelWidth = panel.offsetWidth;
        const panelHeight = panel.offsetHeight;
        const padding = 10;
        
        const spaceRight = window.innerWidth - parentRect.right;
        const spaceLeft = parentRect.left;
        
        let left, top;
        
        if (spaceRight >= panelWidth + padding) {
            left = parentRect.right + padding;
        } else if (spaceLeft >= panelWidth + padding) {
            left = parentRect.left - panelWidth - padding;
        } else {
            left = spaceRight > spaceLeft 
                ? Math.min(parentRect.right + padding, window.innerWidth - panelWidth - padding)
                : Math.max(padding, parentRect.left - panelWidth - padding);
        }
        
        left = Math.max(padding, Math.min(left, window.innerWidth - panelWidth - padding));
        top = parentRect.top;
        
        if (top + panelHeight > window.innerHeight - padding) {
            top = Math.max(padding, window.innerHeight - panelHeight - padding);
        }
        
        panel.style.left = `${left}px`;
        panel.style.top = `${top}px`;
        panel.style.display = originalDisplay;
    }
}

const popupTracker = new PopupTracker();
