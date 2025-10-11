// ============================================================================
// MODULE: API Data Transformer
// ============================================================================

class APITransformer {
    static cytoscapeToAPI(cytoscapeJson) {
        const nodes = cytoscapeJson.elements.nodes || [];
        const edges = cytoscapeJson.elements.edges || [];
        
        // Transformar nodos: data, classes Y POSITION
        const transformedNodes = nodes.map(node => ({
            data: node.data,
            classes: node.classes,
            position: {
                x: node.position.x,
                y: node.position.y
            }
        }));
        
        // Transformar edges: solo data
        const transformedEdges = edges.map(edge => ({
            data: edge.data
        }));
        
        return {
            protocol: networkData.protocol,
            ip_network: networkData.ip_network,
            nodes: transformedNodes,
            edges: transformedEdges
        };
    }
}
