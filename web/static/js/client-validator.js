// ============================================================================
// MODULE: Client-Side Validator
// ============================================================================

class ClientValidator {
    /**
     * Valida el escenario completo antes de guardarlo o ejecutarlo
     */
    static validateScenario() {
        const errors = [];
        const warnings = [];
        
        const nodes = cy.nodes();
        const edges = cy.edges();
        
        // 1. Al menos un nodo
        if (nodes.length === 0) {
            errors.push('The network must have at least one node');
        }
        
        // 2. IDs únicos
        const nodeIds = new Set();
        const duplicateIds = [];
        nodes.forEach(node => {
            const id = node.data('id');
            if (nodeIds.has(id)) {
                duplicateIds.push(id);
            }
            nodeIds.add(id);
        });
        
        if (duplicateIds.length > 0) {
            errors.push(`Duplicate node IDs: ${duplicateIds.join(', ')}`);
        }
        
        // 3. IPs válidas y dentro del rango
        nodes.forEach(node => {
            const ip = node.data('ip');
            const name = node.data('name');
            
            if (!ip) {
                errors.push(`Node "${name}" has no IP address`);
            } else if (!Validators.validateIP(ip)) {
                errors.push(`Node "${name}" has invalid IP: ${ip}`);
            } else if (!this.isIPInNetwork(ip, networkData.ip_network)) {
                errors.push(`Node "${name}" IP ${ip} is outside network ${networkData.ip_network}`);
            }
        });
        
        // 4. Al menos un master (warning)
        const masters = nodes.filter(n => n.data('role') === CONSTANTS.ROLES.MASTER);
        if (masters.length === 0) {
            warnings.push('Network has no master nodes');
        }
        
        // 5. Al menos un slave (warning)
        const slaves = nodes.filter(n => n.data('role') === CONSTANTS.ROLES.SLAVE);
        if (slaves.length === 0) {
            warnings.push('Network has no slave nodes');
        }
        
        // 6. Configuración de slaves
        slaves.forEach(slave => {
            const name = slave.data('name');
            const port = slave.data('port');
            const slaveId = slave.data('slave_id');
            
            if (!port || port === '') {
                errors.push(`Slave "${name}" has no port configured`);
            }
            
            if (!slaveId || slaveId === '') {
                errors.push(`Slave "${name}" has no slave_id configured`);
            }
        });
        
        // 7. Edges válidos
        edges.forEach(edge => {
            const source = edge.source();
            const target = edge.target();
            const edgeId = edge.data('id');
            
            if (source.data('role') === target.data('role')) {
                errors.push(`Edge "${edgeId}" connects nodes of the same role`);
            }
            
            const messages = edge.data('messages') || [];
            if (messages.length === 0) {
                warnings.push(`Edge "${edgeId}" has no messages configured`);
            }
        });
        
        // 8. Nodos aislados (warning)
        nodes.forEach(node => {
            if (node.degree() === 0) {
                warnings.push(`Node "${node.data('name')}" has no connections`);
            }
        });
        
        return { errors, warnings };
    }
    
    static isIPInNetwork(ip, network) {
        try {
            const parsedIP = ipaddr.parse(ip);
            const parsedNetwork = ipaddr.parseCIDR(network);
            return parsedIP.match(parsedNetwork);
        } catch (e) {
            return false;
        }
    }
    
    static showValidationResults(errors, warnings) {
        if (errors.length > 0) {
            const errorMsg = 'Validation errors:\n\n' + 
                           errors.map((e, i) => `${i + 1}. ${e}`).join('\n');
            alert(errorMsg);
            return false;
        }
        
        if (warnings.length > 0) {
            const warningMsg = 'Validation warnings:\n\n' + 
                             warnings.map((w, i) => `${i + 1}. ${w}`).join('\n') +
                             '\n\nDo you want to continue?';
            return confirm(warningMsg);
        }
        
        return true;
    }
}
