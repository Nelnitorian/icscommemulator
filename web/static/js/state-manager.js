// ============================================================================
// MODULE: Application State Manager
// ============================================================================

class StateManager {
    constructor() {
        this.scenarioId = this.getCurrentId();
        this.nodeCount = 3;
        this.edgeCount = 2;
        this.selectedElement = null;
        this.history = [];
        this.redoHistory = [];
        this.lastTaphold = 0;
        this.canvasDragHandler = null;
        this.intervalId = null;
        this.filePath = null;
    }
    
    getCurrentId() {
        const path = window.location.pathname;
        const parts = path.split('/');
        return parts[parts.length - 1] || parts[parts.length - 2];
    }
    
    incrementNodeCount() {
        return ++this.nodeCount;
    }
    
    incrementEdgeCount() {
        return ++this.edgeCount;
    }
    
    selectElement(element) {
        if (this.selectedElement) {
            this.unselectElement();
        }
        this.selectedElement = element;
        element.addClass('selected');
    }
    
    unselectElement() {
        if (this.selectedElement) {
            this.selectedElement.removeClass('selected');
            if (this.selectedElement.isNode()) {
                this.selectedElement.removeClass(CONSTANTS.ROLES.MASTER);
                this.selectedElement.removeClass(CONSTANTS.ROLES.SLAVE);
                this.selectedElement.addClass(this.selectedElement.data('role'));
            }
            this.selectedElement = null;
        }
    }
    
    addToHistory(action, element, connectedEdges = null) {
        this.history.push({
            action,
            element,
            connectedEdges
        });
        this.redoHistory = [];
    }
    
    undo() {
        if (this.history.length === 0) return;
        
        const lastAction = this.history.pop();
        this.redoHistory.push(lastAction);
        
        if (lastAction.action === CONSTANTS.ACTIONS.ADD) {
            cy.getElementById(lastAction.element.data.id).remove();
        } else {
            const restoredElement = cy.add(lastAction.element);
            if (lastAction.connectedEdges) {
                lastAction.connectedEdges.forEach(edge => cy.add(edge));
            }
            restoredElement.removeClass('selected');
        }
    }
    
    redo() {
        if (this.redoHistory.length === 0) return;
        
        const lastAction = this.redoHistory.pop();
        this.history.push(lastAction);
        
        if (lastAction.action === CONSTANTS.ACTIONS.DELETE) {
            cy.getElementById(lastAction.element.data.id).remove();
        } else {
            const restoredElement = cy.add(lastAction.element);
            if (lastAction.connectedEdges) {
                lastAction.connectedEdges.forEach(edge => cy.add(edge));
            }
            restoredElement.removeClass('selected');
        }
    }
    
    setLastTaphold() {
        this.lastTaphold = Date.now();
    }
    
    isTapholdRecent() {
        return Date.now() - this.lastTaphold < CONSTANTS.TAPHOLD_DELAY;
    }
    
    addCanvasListener(func, element = null) {
        this.canvasDragHandler = () => func(element);
        cy.on('pan zoom', this.canvasDragHandler);
    }
    
    removeCanvasListener() {
        if (this.canvasDragHandler) {
            cy.off('pan zoom', this.canvasDragHandler);
            this.canvasDragHandler = null;
        }
    }
}

const State = new StateManager();
