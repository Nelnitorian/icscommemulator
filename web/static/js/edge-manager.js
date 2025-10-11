// ============================================================================
// MODULE: Edge Manager
// ============================================================================

class EdgeManager {
    static createEdge(srcNode, dstNode) {
        // Prevent same-role connections
        if (srcNode.data('role') === dstNode.data('role')) {
            UIUtils.showError('Error: Cannot connect nodes of the same type');
            return;
        }
        
        // Check if edge already exists
        if (!srcNode.edgesWith(dstNode).empty()) {
            return;
        }
        
        const edgeId = State.incrementEdgeCount();
        let sourceNode, targetNode;
        
        // Master -> Slave direction
        if (srcNode.data('role') === CONSTANTS.ROLES.MASTER) {
            sourceNode = srcNode;
            targetNode = dstNode;
        } else {
            sourceNode = dstNode;
            targetNode = srcNode;
        }
        
        const newEdge = cy.add([{
            group: 'edges',
            data: {
                id: `edge${edgeId}`,
                source: sourceNode.id(),
                target: targetNode.id(),
                messages: []
            }
        }]);
        
        State.addToHistory(CONSTANTS.ACTIONS.ADD, newEdge.json());
    }
    
    static populateForm(edge) {
        const data = edge.data();
        DOM.edgeConfigDirection.innerText = 
            `${edge.source().data().name} → ${edge.target().data().name}`;
        
        const messagesContainer = document.getElementById('messages-container');
        messagesUI.init(messagesContainer);
        messagesUI.render(data.messages || []);
    }
    
    static addMessageRow(timestamp = '', recurrent = false, interval = '', 
                         functionCode = 1, startAddress = '', count = '', values = '') {
        const tbody = DOM.edgeConfigForm.getElementsByTagName('tbody')[0];
        const row = tbody.insertRow();
        
        row.innerHTML = `
            <td><input type="number" value="${timestamp}" min="0" placeholder="0"></td>
            <td><input type="checkbox" ${recurrent ? 'checked' : ''}></td>
            <td><input type="number" value="${interval}" min="1" placeholder="1000"></td>
            <td>
                <select>
                    ${CONSTANTS.FUNCTION_CODES.map(code => 
                        `<option value="${code}" ${code === functionCode ? 'selected' : ''}>${code}</option>`
                    ).join('')}
                </select>
            </td>
            <td><input type="text" value="${startAddress}" placeholder="0x0000"></td>
            <td><input type="number" value="${count}" min="1" placeholder="10"></td>
            <td><input type="text" value="${Array.isArray(values) ? values.join(',') : ''}" placeholder="0,1,2"></td>
            <td><button onclick="EdgeManager.removeRow(this)">🗑️</button></td>
        `;
    }
    
    static removeRow(button) {
        const row = button.parentNode.parentNode;
        row.parentNode.removeChild(row);
    }
    
    static saveConfig() {
        const edge = State.selectedElement;
        const data = edge.data();
        data.messages = messagesUI.parseMessages();
    }
    
    static showForm(edge) {
        this.populateForm(edge);
        DOM.showEdgeConfig();
        popupTracker.attachToElement(DOM.edgeConfigPopup, edge, 'center');
    }
}
