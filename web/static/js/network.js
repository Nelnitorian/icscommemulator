// network.js

var cy = cytoscape({
    container: document.getElementById('cy'),

    elements: {
        "nodes": networkData.nodes,
        "edges": networkData.edges
    },

    style: [
        {
            selector: 'node',
            style: {
                'background-color': '#666',
                'label': 'data(name)'
            }
        },
        {
            selector: 'node.master',
            style: {
                'background-color': '#3A86FF'
            }
        },
        {
            selector: 'node.slave',
            style: {
                'background-color': '#FF6D00'
            }
        },
        {
            selector: 'node.selected',
            style: {
                'background-color': '#FFA500'
            }
        },
        {
            selector: 'edge',
            style: {
                'width': 3,
                'line-color': '#ccc',
                'curve-style': 'bezier',
                'target-arrow-color': '#ccc',
                'target-arrow-shape': 'triangle',
                'arrow-scale': '1'
            }
        },
        {
            selector: 'edge.selected',
            style: {
                'line-color': '#FFA500',
                'target-arrow-color': '#FFA500'
            }
        }
    ],

    layout: {
        name: 'preset',
    }
});

const KEY_DELETE = 'Delete';
const KEY_SUPR = 'Supr';
const KEY_ESCAPE = 'Escape';
const KEY_Z = 'z';
const KEY_Y = 'y';
const ACTION_ADD = 'add';
const ACTION_DELETE = 'delete';

const MASTER = 'master';
const SLAVE = 'slave';
const TRIANGLE = 'triangle';

let FILE_PATH;

const LEVEL = {
    ERROR: 1,
    WARNING: 2,
}

const scenarioId = getCurrentId();

let nodeCount = 3;
let edgeCount = 2;
let selectedElement = null;
let history = [];
let redoHistory = [];
let lastTaphold = 0;

let canvasDragHandler;

let intervalId;

const nodeConfigPopup = document.getElementById('configPopup');
const edgeConfigPopup = document.getElementById('edgeConfigPopup');
const nodeConfigForm = document.getElementById('configForm');
const edgeConfigForm = document.getElementById('edgeConfigForm');
const errorMessage = document.getElementById('error-message');

const nameElement = document.getElementById('name');
const commentElement = document.getElementById('comment');
const roleElement = document.getElementById('role');
const slaveConfigElement = document.getElementById('slave-config')

// Slave fields
const ipElement = document.getElementById('ip');
const macElement = document.getElementById('mac');
const portElement = document.getElementById('port');
const slaveIdElement = document.getElementById('slave_id');

const registersPanelElement = document.getElementById('registers_panel');

const discreteInputsTypeElement = document.getElementById('discrete_inputs_type');
const discreteInputsElement = document.getElementById('discrete_inputs');
const coilsTypeElement = document.getElementById('coils_type');
const coilsElement = document.getElementById('coils');
const inputRegistersTypeElement = document.getElementById('input_registers_type');
const inputRegistersElement = document.getElementById('input_registers');
const holdingRegistersTypeElement = document.getElementById('holding_registers_type');
const holdingRegistersElement = document.getElementById('holding_registers');

const identityPanelElement = document.getElementById('identity_panel');

const vendorNameElement = document.getElementById('vendor_name');
const productCodeElement = document.getElementById('product_code');
const majorMinorRevisionElement = document.getElementById('major_minor_revision');
const vendorUrlElement = document.getElementById('vendor_url');
const productNameElement = document.getElementById('product_name');
const modelNameElement = document.getElementById('model_name');
const userApplicationNameElement = document.getElementById('user_application_name');

// Edge fields
const edgeConfigDirectionElement = document.getElementById('edgeConfigDirection');

// Run elements
const runSettingsElement = document.getElementById('run_settings');
const runSettingsContentElement = document.getElementById('run_settings_content');
const simulationTimeElement = document.getElementById('simulation_time');
const cancelButtonElement = document.getElementById('cancel_button');
const runButtonElement = document.getElementById('run_button');

const runOverlayElement = document.getElementById('run_overlay');
const runOverlayContentElement = document.getElementById('run_overlay_content');

const timeProgressElement = document.getElementById('time_progress');
const percentageProgressElement = document.getElementById('percentage_progress');
const pcapSizeElement = document.getElementById('pcap_size');

// Add node on taphold on canvas
cy.on('taphold', function (evt) {
    lastTaphold = new Date().getTime();

    let element = evt.target;
    if (element === cy) {
        handleNewNode(evt);
    } else if (element.isNode()) {
        handleNodeTaphold(element);
    } else if (element.isEdge()) {
        handleEdgeTaphold(element)
    }
});

function handleNewNode(evt) {
    nodeCount++;
    let name = 'node' + nodeCount;
    let emptyIdentity = {
        major_minor_revision: "",
        model_name: "",
        product_code: "",
        product_name: "",
        user_application_name: "",
        vendor_name: "",
        vendor_url: "",
    };
    let newNode = cy.add([
        {
            group: 'nodes',
            classes: SLAVE,
            data: {
                id: name, role: SLAVE, name: name,
                holding_registers: {
                    type: 'sequential',
                    values: ''
                },
                coils: {
                    type: 'sequential',
                    values: ''
                },
                discrete_inputs: {
                    type: 'sequential',
                    values: ''
                },
                input_registers: {
                    type: 'sequential',
                    values: ''
                },
                comment: '',
                ip: assignIp('', ''),
                port: '502',
                slave_id: '1',
                identity: emptyIdentity
            }, position: { x: evt.position.x, y: evt.position.y }
        }
    ]);
    history.push({ action: 'add', element: newNode.json() });
    redoHistory = [];
}

function handleEdgeTaphold(edge) {
    selectElement(edge);
    setupEdgeForm(edge);

}

function setupEdgeForm(edge) {
    populateEdgeForm(edge);
    setEdgeFormLocation();

    edgeConfigPopup.style.display = 'block';

    addCanvasListener(setEdgeFormLocation);
}

function populateEdgeForm(edge) {
    let data = edge.data();

    edgeConfigDirectionElement.innerText = `${edge.source().data().name} → ${edge.target().data().name}`;

    let table = edgeConfigForm.getElementsByTagName('tbody')[0];
    table.innerHTML = '';

    if (data.messages && data.messages.length > 0) {
        data.messages.forEach(message => {
            addRow(message.timestamp, message.recurrent, message.interval, message.function_code, message.start_address, message.count, message.values);
        });
    } else {
        addRow();
    }

}

function setEdgeFormLocation() {
    setElementCenter(edgeConfigPopup);
}

function handleNodeTaphold(node) {
    selectElement(node);

    setupNodeForm(node);
}

function setupNodeForm(node) {
    populateNodeForm(node.data());
    setNodeFormLocation(node);

    slaveConfigElement.style.display = roleElement.value === MASTER ? 'none' : 'block';
    nodeConfigPopup.style.display = 'block';

    addCanvasListener(setNodeConfigLocation, node);
}

function setNodeConfigLocation(node) {
    setNodeFormLocation(node);
    if (registersPanelElement.style.display === 'block') {
        setRegistersPanelLocation();
    } else if (identityPanelElement.style.display === 'block') {
        setIdentityPanelLocation();
    }
}

function setNodeFormLocation(node) {
    let renderedPosition = node.renderedPosition();

    display = nodeConfigPopup.style.display;
    nodeConfigPopup.style.display = 'block';
    let offsetWidth = nodeConfigPopup.offsetWidth;
    let offsetHeight = nodeConfigPopup.offsetHeight;

    // x position beyond half screen
    if (window.innerWidth / 2 > renderedPosition.x) {
        nodeConfigPopup.style.left = renderedPosition.x + 'px';
    } else {
        nodeConfigPopup.style.left = (renderedPosition.x - offsetWidth) + 'px';
    }

    // y position beyond half screen
    if (window.innerHeight / 2 > renderedPosition.y) {
        nodeConfigPopup.style.top = renderedPosition.y + 'px';
    } else {
        nodeConfigPopup.style.top = (renderedPosition.y - offsetHeight) + 'px';
    }

    nodeConfigPopup.style.display = display;
}

function populateNodeForm(data) {

    nameElement.value = data.name;
    commentElement.value = data.comment || '';
    roleElement.value = data.role || SLAVE;

    ipElement.value = data.ip || '';
    macElement.value = data.mac || '';
    portElement.value = data.port || '502';
    slaveIdElement.value = data.slave_id || '1';
}

function setIdentityPanelLocation() {
    setPanelLocation(identityPanelElement, nodeConfigPopup);
}

function setRegistersPanelLocation() {
    setPanelLocation(registersPanelElement, nodeConfigPopup);
}

function setPanelLocation(panel, parent) {
    let display = panel.style.display;

    panel.style.display = 'block';
    let pxLeft = parent.style.left

    let spaceLeft = parseInt(pxLeft.substring(0, pxLeft.length - 2));
    let spaceRight = window.innerWidth - spaceLeft - parent.offsetWidth;

    if (spaceRight > spaceLeft) {
        panel.style.left = window.innerWidth - spaceRight + 'px';
    } else {
        panel.style.left = spaceLeft - panel.offsetWidth + 'px';
    }

    panel.style.top = parseInt(parent.style.top.substring(0, parent.style.top.length - 2)) + 'px';

    panel.style.display = display;
}

function isEmpty(obj) {
    for (const prop in obj) {
        if (Object.hasOwn(obj, prop)) {
            return false;
        }
    }

    return true;
}

function isSequential(obj) {
    if (obj instanceof Array) {
        return true;
    }
    if (obj === "") {
        return true;
    }
    return false;
}

function registerToString(register, type) {
    if (register && type === 'sequential') {
        return register.join(',');
    } else {
        return Object.keys(register).map(key => `${key}:${register[key]}`).join(',');
    }
}

function populateIdentityConfiguration(data) {
    let identity_data = data.identity || {};
    vendorNameElement.value = identity_data.vendor_name || '';
    productCodeElement.value = identity_data.product_code || '';
    majorMinorRevisionElement.value = identity_data.major_minor_revision || '';
    vendorUrlElement.value = identity_data.vendor_url || '';
    productNameElement.value = identity_data.product_name || '';
    modelNameElement.value = identity_data.model_name || '';
    userApplicationNameElement.value = identity_data.user_application_name || '';
}

function populateRegistersConfiguration(data) {
    discreteInputsTypeElement.value = data.discrete_inputs.type || 'sequential';
    discreteInputsElement.value = registerToString(data.discrete_inputs.values, discreteInputsTypeElement.value);

    coilsTypeElement.value = data.coils.type || 'sequential';
    coilsElement.value = registerToString(data.coils.values, coilsTypeElement.value);

    inputRegistersTypeElement.value = data.input_registers.type || 'sequential';
    inputRegistersElement.value = registerToString(data.input_registers.values, inputRegistersTypeElement.value);

    holdingRegistersTypeElement.value = data.holding_registers.type || 'sequential';
    holdingRegistersElement.value = registerToString(data.holding_registers.values, holdingRegistersTypeElement.value);
}

// function populateRegistersConfiguration(data) {
//     discreteInputsTypeElement.value = isSequential(data.discrete_inputs) ? 'sequential' : 'sparse';
//     discreteInputsElement.value = !isEmpty(data.discrete_inputs) ? registerToString(data.discrete_inputs) : '';

//     coilsTypeElement.value = isSequential(data.coils) ? 'sequential' : 'sparse';
//     coilsElement.value = !isEmpty(data.coils) ? registerToString(data.coils) : '';

//     inputRegistersTypeElement.value = isSequential(data.input_registers) ? 'sequential' : 'sparse';
//     inputRegistersElement.value = !isEmpty(data.input_registers) ? registerToString(data.input_registers) : '';

//     holdingRegistersTypeElement.value = isSequential(data.holding_registers) ? 'sequential' : 'sparse';
//     holdingRegistersElement.value = !isEmpty(data.holding_registers) ? registerToString(data.holding_registers) : '';
// }

function showRegistersConfiguration(event) {
    event.preventDefault();

    // Close Identity panel
    if (identityPanelElement.style.display === 'block') {
        saveNodeIdentity(selectedElement.data());
        identityPanelElement.style.display = 'none';
    }

    // Handle Registers panel state
    if (registersPanelElement.style.display !== 'block') {
        registersPanelElement.style.display = 'block';
        populateRegistersConfiguration(selectedElement.data());
        setRegistersPanelLocation();
    } else {
        saveNodeRegisters(selectedElement.data());
        registersPanelElement.style.display = 'none';
    }
}


function showIdentityConfiguration(event) {
    event.preventDefault();
    // Close Registers panel
    if (registersPanelElement.style.display === 'block') {
        saveNodeRegisters(selectedElement.data());
        registersPanelElement.style.display = 'none';
    }

    // Handle Identity panel state
    if (identityPanelElement.style.display !== 'block') {
        identityPanelElement.style.display = 'block';
        populateIdentityConfiguration(selectedElement.data());
        setIdentityPanelLocation();
    } else {
        saveNodeIdentity(selectedElement.data());
        identityPanelElement.style.display = 'none';
    }
}

cy.on('tap', function (evt) {
    // if this tap has been generated from a taphold, do nothing
    if (new Date().getTime() - lastTaphold < 500) {
        lastTaphold = 0;
        return;
    }

    // handle configuration save
    handleConfigurationSave();

    // discern the target of the tap
    let element = evt.target
    if (element === cy) {
        if (selectedElement) {
            unselectElement();
        }
        // target is canvas
    } else if (element.isNode()) {
        handleNodeTap(element);
    } else if (element.isEdge()) {
        handleEdgeTap(element)
    }
});

function removeCanvasListener() {
    if (canvasDragHandler) {
        cy.off('pan zoom', canvasDragHandler);
        canvasDragHandler = null;
    }
}

function addCanvasListener(func, ele = null) {
    canvasDragHandler = () => func(ele);
    cy.on('pan zoom', canvasDragHandler);
}

function handleConfigurationSave() {
    if (nodeConfigPopup.style.display === 'block' && selectedElement.isNode()) {
        saveNodeConfig();
    } else if (edgeConfigPopup.style.display === 'block' && selectedElement.isEdge()) {
        saveEdgeConfig();
    }
    removeCanvasListener();
    registersPanelElement.style.display = 'none';
    identityPanelElement.style.display = 'none';
    nodeConfigPopup.style.display = 'none';
    edgeConfigPopup.style.display = 'none';
}

function unselectElement() {
    if (selectedElement) {
        selectedElement.removeClass('selected');
        if (selectedElement.isNode()) {
            selectedElement.removeClass("master");
            selectedElement.removeClass("slave");
            selectedElement.addClass(selectedElement.data('role'));
        }
        selectedElement = null;
    }
}

function parseElementValues(type, value, value_save, type_save) {
    let parsedValue = parseRegisterValues(type, value);
    let parsedType = type;
    if (parsedValue === -1) {
        parsedValue = "";
    } else if (parsedValue === null) {
        parsedValue = value_save;
        parsedType = type_save
    }
    return [parsedType, parsedValue];
}

function getNextIpInSubnet(ipAddress, subnet) {
    const ip = ipaddr.parse(ipAddress);
    const subnetParsed = ipaddr.parseCIDR(subnet);

    if (ip.match(subnetParsed)) {
        const nextIp = ip.toByteArray();

        for (let i = nextIp.length - 1; i >= 0; i--) {
            if (nextIp[i] < 255) {
                nextIp[i]++;
                break;
            }
            nextIp[i] = 0;
        }

        return ipaddr.fromByteArray(nextIp).toString();
    } else {
        throw new Error("La IP no pertenece a la subred especificada.");
    }
}

function getUniqueIp(existingIps, subnet) {
    let newIp = subnet.split('/')[0];
    do {
        newIp = getNextIpInSubnet(newIp, subnet);
    } while (existingIps.includes(newIp));
    return newIp;
}

function assignIp(ipCandidate, save) {
    const ipValue = ipCandidate.trim();
    if (ipValue === '') {
        const existingIps = cy.nodes().map(node => node.data('ip'));
        existingIps.push(getUniqueIp([], networkData.ip_network));
        return getUniqueIp(existingIps, networkData.ip_network);
    } else {
        if (ipaddr.isValid(ipValue)) {
            return ipValue;
        } else {
            console.error('Invalid IP address');
            configError("Error: Invalid IP address");
            // maintain previous ip value
            return save
        }
    }
}

function verifyMac(macCandidate) {
    let macRegex = /^(?:[0-9A-Fa-f]{2}([:-]?|\.?)){5}[0-9A-Fa-f]{2}$/;
    return macRegex.test(macCandidate)
}

function parseToColonFormat(mac, save) {

    if (!verifyMac(mac)) {
        return save
    }

    // Regular expression to match the colon-separated format exactly
    const colonFormatRegex = /^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$/;

    // If the MAC is already in the correct colon-separated format, return it as is
    if (colonFormatRegex.test(mac)) {
        return mac.toUpperCase();
    }

    // Remove any existing separators (colons, dashes, dots)
    const cleanMac = mac.replace(/[^0-9A-Fa-f]/g, '');

    // Ensure the cleaned MAC has exactly 12 hexadecimal characters
    if (cleanMac.length !== 12) {
        throw new Error('Invalid MAC address format');
    }

    // Insert colons every two characters to match the required format
    const formattedMac = cleanMac.match(/.{2}/g).join(':').toUpperCase();

    return formattedMac;
}

function getNextIpInSubnet(ipAddress, subnet) {
    const ip = ipaddr.parse(ipAddress);
    const subnetParsed = ipaddr.parseCIDR(subnet);

    if (ip.match(subnetParsed)) {
        const nextIp = ip.toByteArray();

        for (let i = nextIp.length - 1; i >= 0; i--) {
            if (nextIp[i] < 255) {
                nextIp[i]++;
                break;
            }
            nextIp[i] = 0;
        }

        return ipaddr.fromByteArray(nextIp).toString();
    } else {
        throw new Error("La IP no pertenece a la subred especificada.");
    }
}

function saveNodeConfig() {
    let data = selectedElement.data();
    data.name = nameElement.value || data.name;
    data.comment = commentElement.value;
    data.role = roleElement.value;
    data.ip = assignIp(ipElement.value, data.ip);
    data.mac = parseToColonFormat(macElement.value, data.mac);
    if (data.role === SLAVE) {
        data.port = portElement.value;
        data.slave_id = slaveIdElement.value;
        // Parse register values

        if (registersPanelElement.style.display === 'block') {
            saveNodeRegisters(data);
        } else if (identityPanelElement.style.display === 'block') {
            saveNodeIdentity(data);
        }
    } else if (data.role === MASTER) {
        // If the node is a master node, add an arrow to its edges to slave nodes
        let connectedEdges = selectedElement.connectedEdges();
        connectedEdges.forEach(function (edge) {
            if (edge.target().data('role') === SLAVE) {
                edge.style({ 'target-arrow-shape': TRIANGLE });
            }
        });
    }
}

function saveNodeRegisters(data) {
    [data.discrete_inputs.type, data.discrete_inputs.values] = parseElementValues(discreteInputsTypeElement.value, discreteInputsElement.value, data.discrete_inputs.values, data.discrete_inputs.type);
    [data.coils.type, data.coils.values] = parseElementValues(coilsTypeElement.value, coilsElement.value, data.coils.values, data.coils.type);
    [data.input_registers.type, data.input_registers.values] = parseElementValues(inputRegistersTypeElement.value, inputRegistersElement.value, data.input_registers.values, data.input_registers.type);
    [data.holding_registers.type, data.holding_registers.values] = parseElementValues(holdingRegistersTypeElement.value, holdingRegistersElement.value, data.holding_registers.values, data.holding_registers.type);
}

// function saveNodeRegisters(data) {
//     data.discrete_inputs = parseElementValues(discreteInputsTypeElement.value, discreteInputsElement.value, data.discrete_inputs);
//     data.coils = parseElementValues(coilsTypeElement.value, coilsElement.value, data.coils);
//     data.input_registers = parseElementValues(inputRegistersTypeElement.value, inputRegistersElement.value, data.input_registers);
//     data.holding_registers = parseElementValues(holdingRegistersTypeElement.value, holdingRegistersElement.value, data.holding_registers);
// }

function saveNodeIdentity(data) {
    data.identity = {
        vendor_name: vendorNameElement.value,
        product_code: productCodeElement.value,
        major_minor_revision: majorMinorRevisionElement.value,
        vendor_url: vendorUrlElement.value,
        product_name: productNameElement.value,
        model_name: modelNameElement.value,
        user_application_name: userApplicationNameElement.value
    }
}

function hasValidFields(dict) {
    // Check if timestamp is a whole positive number
    if (!Number.isInteger(dict.timestamp) || dict.timestamp < 0) {
        return false;
    }

    // Check if interval is valid
    if (dict.recurrent) {
        if (!Number.isInteger(dict.interval) || dict.interval <= 0) {
            return false;
        }
    }

    // Check if startAddress is a number between 0x0000 and 0xFFFF
    const startAddress = parseInt(dict.start_address, 16);
    if (isNaN(startAddress) || startAddress < 0x0000 || startAddress > 0xFFFF) {
        return false;
    }

    // Check if functionCode is in the valid set of numbers
    const validFunctionCodes = [1, 2, 3, 4, 5, 6, 15, 16, 43];
    if (!validFunctionCodes.includes(dict.function_code)) {
        return false;
    }

    // Check additional fields based on functionCode
    if ([1, 2, 3, 4].includes(dict.function_code)) {
        // Check if count is an integer value
        if (!Number.isInteger(dict.count)) {
            return false;
        }
    } else if ([5, 6, 15, 16].includes(dict.function_code)) {
        // Check if values is a comma-separated list in string format
        if (!Array.isArray(dict.values) || !dict.values.every(value => typeof value === 'number')) {
            return false;
        }
    }

    return true;
}

function saveEdgeConfig() {
    let edge = selectedElement;
    let data = edge.data();

    // Is information correct?
    // For all cells in all rows. Store columns: 
    // [recurrent, time marker, fc, start address, count, values]
    // in a messages: [ {}, {}, {}...]

    let table = edgeConfigForm.getElementsByTagName('tbody')[0];
    let rows = table.rows;
    let messages = [];

    let invalid_rows = [];

    for (let i = 0; i < rows.length; i++) {
        let row = rows[i];
        let cells = row.getElementsByTagName('input');
        let select = row.getElementsByTagName('select')[0]; // Get the select element

        let message = {
            timestamp: parseInt(cells[0].value),
            recurrent: cells[1].checked,
            interval: parseInt(cells[2].value),
            function_code: parseInt(select.value), // Use the value from the select element
            start_address: parseInt(cells[3].value),
            count: parseInt(cells[4].value),
            values: JSON.parse("[" + cells[5].value + "]")
        }

        if (!hasValidFields(message)) {
            invalid_rows.push(i + 1);
            continue;
        }
        messages.push(message);
    }

    // un único mensaje erróneo
    if (invalid_rows.length === 1) {
        configError(`Error: Campos de mensaje inválidos en la fila ${invalid_rows[0]}`);
    } else if (invalid_rows.length > 1) {
        configError(`Error: Campos de mensaje inválidos en las filas ${invalid_rows.join(", ")}`);
    }
    data.messages = messages;
}

function
    handleEdgeTap(edge) {
    if (!selectedElement) {
        // selectedElement not null
        selectElement(edge);
        return
    }

    if (selectedElement !== edge) {
        // clicked a different edge
        unselectElement();
        selectElement(edge);
    } else {
        // clicked same edge
        unselectElement();
    }
}

function handleNodeTap(node) {
    if (!selectedElement) {
        // selectedElement not null
        selectElement(node);
        return
    }

    if (selectedElement.isNode()) {
        if (selectedElement === node) {
            // clicked same node -> unselect
            unselectElement();
        } else {
            // clicked another node
            handleNewEdge(node);
            // unselect the selected node
            unselectElement();
        }
    } else {
        // if it is edge ignore and we select the node
        selectElement(node);
    }
}

function handleNewEdge(node) {
    if (selectedElement.data('role') !== node.data('role')) {
        // Check if an edge already exists between the selected node and the current node
        if (selectedElement.edgesWith(node).empty()) {  // If no edge exists, create a new one
            createEdge(selectedElement, node);
        }
    } else {
        configError('Error: No se puede conectar nodos del mismo tipo');
    }
}

function selectElement(element) {
    if (selectedElement) {
        unselectElement();
    }
    selectedElement = element;
    // selectedElement.removeClass(selectedElement.data('role'));
    selectedElement.addClass('selected');
}

function createEdge(srcNode, dstNode) {
    edgeCount++;
    let sourceNode, targetNode;
    // Make sure the edge always goes from master to slave
    if (srcNode.data('role') === MASTER) {
        sourceNode = srcNode;
        targetNode = dstNode;
    } else {
        sourceNode = dstNode;
        targetNode = srcNode;
    }
    let newEdge = cy.add([
        { group: 'edges', data: { id: 'edge' + edgeCount, source: sourceNode.id(), messages: [], target: targetNode.id() } }
    ]);
    history.push({ action: 'add', element: newEdge.json() });
    redoHistory = [];
}

function handleAction(actionHistory, oppositeActionHistory, actionType) {
    if (actionHistory.length > 0) {
        let lastAction = actionHistory.pop();
        oppositeActionHistory.push(lastAction);
        if (lastAction.action === actionType) {
            cy.getElementById(lastAction.element.data.id).remove();
        } else {
            let restoredElement = cy.add(lastAction.element);
            if (lastAction.connectedEdges) {
                for (let edge of lastAction.connectedEdges) {
                    cy.add(edge);
                }
            }
            restoredElement.removeClass('selected');
        }
    }
}

document.addEventListener('keydown', function (evt) {
    if (evt.key === KEY_DELETE || evt.key === KEY_SUPR) {
        if (selectedElement &&
            !(nodeConfigPopup.style.display === 'block') &&
            !(edgeConfigPopup.style.display === 'block') &&
            !(runOverlayElement.style.display === 'block') &&
            !(runSettingsElement.style.display === 'block')) {
            handleConfigurationSave();

            let connectedEdges = selectedElement.connectedEdges().jsons();
            history.push({ action: 'delete', element: selectedElement.json(), connectedEdges: connectedEdges });
            selectedElement.remove();
            selectedElement = null;
            redoHistory = [];
        }
    } else if (evt.key === KEY_ESCAPE) {
        handleConfigurationSave();
        unselectElement();

    } else if (evt.ctrlKey) {
        if (evt.key === KEY_Z) {
            handleAction(history, redoHistory, ACTION_ADD);
        } else if (evt.key === KEY_Y) {
            handleAction(redoHistory, history, ACTION_DELETE);
        }
    }
});

document.addEventListener('DOMContentLoaded', function () {
    roleElement.addEventListener('change', function () {
        if (selectedElement.connectedEdges().length > 0) {
            configError('Error: No se puede cambiar el rol de un nodo conectado');
            this.value = selectedElement.data('role');
            return;
        }

        if (this.value === 'slave') {
            slaveConfigElement.style.display = 'block';
        } else {
            slaveConfigElement.style.display = 'none';
        }
    });

    discreteInputsTypeElement.addEventListener('change', function () {
        var placeholder = this.value === 'sparse' ? '1:1,2:0,3:1,4:0' : '0,1,0,0,1';
        discreteInputsElement.placeholder = placeholder;
    });

    coilsTypeElement.addEventListener('change', function () {
        var placeholder = this.value === 'sparse' ? '1:1,2:0,3:1,4:0' : '0,1,0,0,1';
        coilsElement.placeholder = placeholder;
    });

    inputRegistersTypeElement.addEventListener('change', function () {
        var placeholder = this.value === 'sparse' ? '1:1,2:2,3:3,4:4' : '0,1,2,3,4';
        inputRegistersElement.placeholder = placeholder;
    });

    holdingRegistersTypeElement.addEventListener('change', function () {
        var placeholder = this.value === 'sparse' ? '1:1,2:2,3:3,4:4' : '0,1,2,3,4';
        holdingRegistersElement.placeholder = placeholder;
    });


    if (areAllNodesAtOrigin()) {
        // Define the concentric layout
        var concentricLayout = {
            name: 'concentric',
            concentric: function (node) {
                return node.degree();
            },
            levelWidth: function (nodes) {
                return 1;
            },
            spacingFactor: 2,
            padding: 100
        };
        cy.layout(concentricLayout).run();
    }
});

function areAllNodesAtOrigin() {
    let nodes = cy.nodes();

    return nodes.every(node => {
        let position = node.position();
        return position.x === 0 && position.y === 0;
    });
}

function zoomIn() {
    cy.zoom(cy.zoom() * 1.2);
    cy.center();
}

function zoomOut() {
    cy.zoom(cy.zoom() * 0.8);
    cy.center();
}

function parseRegisterValues(type, values) {
    if (!values) {
        return -1;
    }
    let includeColon = type === 'sparse';
    if (checkCharacters(values, includeColon)) {
        if (type === 'sequential') {
            let array = values.split(',').map(Number);
            return array;
        } else {  // Sparse
            result = values.split(',')
                .map(pair => pair.split(':').map(item => item.trim()))
                .reduce((acc, [key, value]) => {
                    acc[key] = parseInt(value, 10);
                    return acc;
                }, {});
            for (let key in result) {
                if (parseInt(key, 10) = 0) {
                    console.log('Error: Register key cannot be less than or equal to 0');
                    configError('Error: Register key cannot be less than or equal to 0');
                    return null;
                }
            }
            return result
        }
    } else {
        configError('Error: Formato incorrecto de valores de registro');
        return null;
    }
}

function checkCharacters(str, includeColon) {
    let specialChars = includeColon ? /[^0-9:\s,]/ : /[^0-9\s,]/;
    return !specialChars.test(str);
}

// Edge popup configuration

function addRow(timestamp = '', recurrent = false, interval = '', functionCode = 1, startAddress = '', count = '', values = '') {
    const table = edgeConfigForm.getElementsByTagName('tbody')[0];
    const newRow = table.insertRow();

    const timestampCell = newRow.insertCell(0);
    const recurrentCell = newRow.insertCell(1);
    const intervalCell = newRow.insertCell(2);
    const functionCodeCell = newRow.insertCell(3);
    const startAddressCell = newRow.insertCell(4);
    const countCell = newRow.insertCell(5);
    const valuesCell = newRow.insertCell(6);
    const deleteCell = newRow.insertCell(7);

    timestampCell.innerHTML = `<input type="text" placeholder="0" value="${timestamp}">`;
    recurrentCell.innerHTML = `<input type="checkbox" ${recurrent ? 'checked' : ''} onclick="recurrentToggle(this)">`;
    intervalCell.innerHTML = `<td><input type="text" id="interval" ${recurrent ? '' : 'class="hidden"'} placeholder="5" value="${interval}"></td>`
    functionCodeCell.innerHTML = `
      <select onchange="updateFields(event)">
        <option value="1" ${functionCode === 1 ? 'selected' : ''}>Read Coils (1)</option>
        <option value="2" ${functionCode === 2 ? 'selected' : ''}>Read Discrete Inputs (2)</option>
        <option value="3" ${functionCode === 3 ? 'selected' : ''}>Read Holding Registers (3)</option>
        <option value="4" ${functionCode === 4 ? 'selected' : ''}>Read Input Registers (4)</option>
        <option value="5" ${functionCode === 5 ? 'selected' : ''}>Write Single Coil (5)</option>
        <option value="6" ${functionCode === 6 ? 'selected' : ''}>Write Single Register (6)</option>
        <option value="15" ${functionCode === 15 ? 'selected' : ''}>Write Multiple Coils (15)</option>
        <option value="16" ${functionCode === 16 ? 'selected' : ''}>Write Multiple Registers (16)</option>
        <option value="43" ${functionCode === 43 ? 'selected' : ''}>Read Device Information (43)</option>
      </select>`;

    startAddressCell.innerHTML = `<input type="text" ${[43].includes(functionCode) ? "class=\"hidden\"" : ""} placeholder="0" value="${startAddress}">`;
    countCell.innerHTML = `<input type="text" ${[5, 6, 15, 16, 43].includes(functionCode) ? "class=\"hidden\"" : ""} placeholder="10" value="${count}">`;
    valuesCell.innerHTML = `<input type="text" ${[1, 2, 3, 4, 43].includes(functionCode) ? "class=\"hidden\"" : ""} placeholder="1,2,3" value="${values}">`;
    deleteCell.innerHTML = '<button onclick="deleteRow(this)">Delete</button>';

    setEdgeFormLocation();
}

function recurrentToggle(evt) {
    let intervalCell = evt.parentNode.parentNode.cells[2].getElementsByTagName('input')[0];
    if (evt.checked) {
        intervalCell.classList.remove('hidden');
        intervalCell.classList.add('visible');
    } else {
        intervalCell.classList.add('hidden');
        intervalCell.classList.remove('visible');
    }
}


function deleteRow(btn) {
    var row = btn.parentNode.parentNode;
    row.parentNode.removeChild(row);
}

function configError(message) {
    errorMessage.style.opacity = '1';
    errorMessage.textContent = message;
    errorMessage.style.display = 'block';
    setTimeout(function () {
        errorMessage.style.opacity = '0';
    }, 3000);
}

function updateFields(event) {
    const selectElement = event.target;
    const row = selectElement.parentNode.parentNode;
    const functionCode = selectElement.value;
    const startAddressField = row.cells[4].getElementsByTagName('input')[0];
    const countField = row.cells[5].getElementsByTagName('input')[0];
    const valuesField = row.cells[6].getElementsByTagName('input')[0];


    startAddressField.classList.remove('hidden');

    // Show or hide the "count" and "values" fields depending on the function code.
    if (['1', '2', '3', '4'].includes(functionCode)) {
        countField.classList.remove('hidden');
        countField.classList.add('visible');
        valuesField.classList.add('hidden');
        valuesField.classList.remove('visible');
    } else if (['5', '6', '15', '16'].includes(functionCode)) {
        countField.classList.add('hidden');
        countField.classList.remove('visible');
        valuesField.classList.remove('hidden');
        valuesField.classList.add('visible');
        valuesField.placeholder = functionCode === '5' ? '0 or 1' : '0x0001 or 14';
    } else if (functionCode === '43') {
        countField.classList.add('hidden');
        countField.classList.remove('visible');
        valuesField.classList.add('hidden');
        valuesField.classList.remove('visible');
        startAddressField.classList.add('hidden');
    } else {
        countField.classList.add('hidden');
        countField.classList.remove('visible');
        valuesField.classList.add('hidden');
        valuesField.classList.remove('visible');
    }
}

function askForConfirmation(logs) {
    let errors = logs.filter(log => log.startsWith("[ERROR]"));
    let warnings = logs.filter(log => log.startsWith("[WARNING]"));

    if (errors.length > 0) {
        alert("Errors:\n" + errors.join("\n"));
        return false;
    }

    if (warnings.length > 0) {
        return confirm(warnings.join("\n") + "\n\nDo you want to proceed?");
    }

    return true;
}

function run() {
    runSettingsElement.style.display = 'block';
    simulationTimeElement.value = '';
    setRunSettingsLocatiton();
}

function setElementCenter(element) {

    // Ensure the element is visible to get its dimensions
    let display = element.style.display;
    element.style.display = 'block';

    // Get the dimensions of the window and the element
    let windowWidth = window.innerWidth;
    let windowHeight = window.innerHeight;
    let popupWidth = element.offsetWidth;
    let popupHeight = element.offsetHeight;

    // Calculate the position to center the element
    let leftPosition = (windowWidth - popupWidth) / 2;
    let topPosition = (windowHeight - popupHeight) / 2;

    // Set the element's position
    element.style.position = 'absolute';
    element.style.left = leftPosition + 'px';
    element.style.top = topPosition + 'px';

    // Restore the element's display property
    element.style.display = display;

}

function setRunSettingsLocatiton() {
    setElementCenter(runSettingsContentElement);
}

function simulate() {
    let simulation_time = parseInt(simulationTimeElement.value);
    if (simulation_time === '') {
        alert('Error: Debe ingresar un tiempo de simulación');
        return;
    }
    if (!Number.isInteger(simulation_time)) {
        alert('Error: El tiempo de simulación debe ser un número entero');
        return;
    }
    if (simulation_time <= 0) {
        alert('Error: El tiempo de simulación debe ser mayor a 0');
        return;
    }

    fetch('/api/run/' + scenarioId, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ "simulation_time": simulation_time })
    })
        .then(response => response.json().then(data => ({ status: response.status, body: data })))
        .then(({ status, body }) => {
            if (status === 200) {
                runSettingsElement.style.display = 'none';
                FILE_PATH = body.file_path;
                start(simulation_time);
            } else {
                alert(`Error al iniciar la simulación: ${body.error}`);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            alert('Error al iniciar la simulación');
        });
}

async function start() {
    runOverlayElement.style.display = 'block';
    setRunOverlayContentLocatiton();
    reserRunOverlayContent();

    intervalId = setInterval(fetchRunData, 1000);
}

function reserRunOverlayContent() {
    timeProgressElement.innerText = '00:00:00 / 00:00:00';
    percentageProgressElement.innerText = '0%';
    pcapSizeElement.innerText = '0';
}

function setRunOverlayContentLocatiton() {
    setElementCenter(runOverlayContentElement);
}

function fetchRunData() {
    try {
        fetch('/api/run/', {
            method: 'GET',
        })
            .then(response => response.json())
            .then(data => {
                const elapsedSeconds = data.elapsed_seconds;
                const totalSeconds = data.total_seconds;
                const pcapSize = data.pcap_size;
                const running = data.running;

                if (running) {
                    if (elapsedSeconds >= totalSeconds) {
                        alert('Simulación finalizada. Guardado en ' + FILE_PATH);
                        cleanRunOverlay();

                    } else {
                        // Format time in hh:mm:ss
                        const formatTime = seconds => new Date(seconds * 1000).toISOString().substr(11, 8);

                        timeProgressElement.innerText = `${formatTime(elapsedSeconds)} / ${formatTime(totalSeconds)}`;
                        percentageProgressElement.innerText = `${((elapsedSeconds / totalSeconds) * 100).toFixed(2)}%`;
                        pcapSizeElement.innerText = `${pcapSize}`;

                    }
                }
                setRunOverlayContentLocatiton();
                console.log(data);
            });
    } catch (error) {
        console.error('Error fetching run data:', error);
    }
}

function stop() {
    cleanRunOverlay();
    fetch('/api/run/', {
        method: 'DELETE',
    }).then(response => { console.log(response.json()); });
}

function cleanRunOverlay() {
    clearInterval(intervalId);
    runOverlayElement.style.display = 'none';
}

function cancel() {
    runSettingsElement.style.display = 'none';
}

function save() {
    let nodes = cy.nodes().jsons();
    let edges = cy.edges().jsons();

    for (let node of nodes) {
        delete node.data.label;
    }

    let network = { nodes: nodes, edges: edges };
    let logs = verifyNetworkData(network);
    network.protocol = networkData.protocol
    network.ip_network = networkData.ip_network
    let permission = true;
    if (logs.length > 0) {
        permission = askForConfirmation(logs);
    }
    if (permission) {
        saveNetwork(network);
    }
}

function getCurrentId() {
    let currentPath = window.location.pathname;
    let pathSegments = currentPath.split('/');
    let id = pathSegments[pathSegments.length - 1];
    return id;
}

function saveNetwork(json) {
    let network = JSON.stringify(json);
    // Send the network to the /api/save_network endpoint
    fetch('/api/network/' + scenarioId, {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(network)
    }).then(response => {
        if (response.ok) {
            alert("Red guardada correctamente");
        } else {
            configError('Error al guardar la red: ' + response.statusText);
        }
    });
}

function verifyNetworkData(data) {
    let logs = [];



    // Check if all ids are different
    const nodeIds = data.nodes.map(node => node.data.id);
    if (nodeIds.length !== new Set(nodeIds).size) {
        // Find duplicated node IDs
        const duplicates = nodeIds.filter((item, index) => nodeIds.indexOf(item) !== index);

        // Print duplicated node IDs
        const log = `[ERROR] All node IDs are not unique. Duplicates: ${[...new Set(duplicates)]}`;
        console.log(log);
        logs.push(log);
    }

    // Check if the edges are made from nodes with different roles
    const nodeRoles = Object.fromEntries(data.nodes.map(node => [node.data.id, node.data.role]));
    data.edges.forEach(edge => {
        const sourceRole = nodeRoles[edge.data.source];
        const targetRole = nodeRoles[edge.data.target];
        if (sourceRole === targetRole) {
            let log = `[ERROR] Edge ${edge.data.id} connects nodes with the same role (${sourceRole})`;
            console.log(log);
            logs.push(log);
        }
    });

    // Check if there are either empty or repeated ips
    const seenIPs = new Set();
    const repeatedIPs = new Map();

    data.nodes.forEach(node => {
        if (!node.data.ip) {
            let log = `[ERROR] Node ${node.data.name} has empty IP address`;
            console.log(log);
            logs.push(log);
        } else if (seenIPs.has(node.data.ip)) {
            let log = `[ERROR] Node ${node.data.name} has a repeated IP address: ${node.data.ip}`;
            console.log(log);
            logs.push(log);
            if (repeatedIPs.has(node.data.ip)) {
                repeatedIPs.get(node.data.ip).push(node.data.name);
            } else {
                repeatedIPs.set(node.data.ip, [node.data.name]);
            }
        } else {
            seenIPs.add(node.data.ip);
        }
    });

    // Print nodes with repeated IPs
    repeatedIPs.forEach((nodes, ip) => {
        let log = `[ERROR] IP address ${ip} is used by nodes: ${nodes.join(', ')}`;
        console.log(log);
        logs.push(log);
    });


    // Check if there is no slave without any defined register
    data.nodes.forEach(node => {
        if (node.data.role === "slave") {
            if (!["holding_registers", "coils", "discrete_inputs", "input_registers"].some(key => node.data[key].values)) {
                let log = `[WARNING] Slave node ${node.data.name} has no defined registers`;
                console.log(log);
                logs.push(log);
            }
        }
    });

    // Check if there is no edge with an empty "messages" field
    data.edges.forEach(edge => {
        if (edge.data.messages.length === 0) {
            let log = `[WARNING] Communication ${edge.data.source} → ${edge.data.target} is empty.`;
            console.log(log);
            logs.push(log);
        }
    });


    // Extract source and target IDs from the links
    const linkedNodeIds = new Set();
    data.edges.forEach(edge => {
        linkedNodeIds.add(edge.data.source);
        linkedNodeIds.add(edge.data.target);
    });

    // Find nodes without links
    const nodesWithoutLinks = nodeIds.filter(nodeId => !linkedNodeIds.has(nodeId));

    // Print nodes without links if any are found
    nodesWithoutLinks.forEach(nodeId => {
        let node = cy.getElementById(nodeId);
        let log = `[WARNING] Node without link: ${node.data().name}`;
        console.log(log);
        logs.push(log);
    });

    return logs;
}