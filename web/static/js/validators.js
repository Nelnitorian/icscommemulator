// ============================================================================
// MODULE: Data Validators
// ============================================================================

class Validators {
    static validateIP(ipAddress) {
        return ipaddr.isValid(ipAddress);
    }
    
    static validateMAC(macAddress) {
        // Acepta 4, 5, o 6 pares de hex con separadores : - .
        const macRegex = /^(?:[0-9A-Fa-f]{2}([:\-.]?)){3,5}[0-9A-Fa-f]{2}$/;
        return macRegex.test(macAddress);
    }

    static parseMACToColonFormat(mac, fallback) {
        if (!this.validateMAC(mac)) {
            return fallback;
        }
        
        // Si ya está en formato correcto con colons
        const colonFormatRegex = /^([0-9A-Fa-f]{2}:){3,5}[0-9A-Fa-f]{2}$/;
        if (colonFormatRegex.test(mac)) {
            return mac.toUpperCase();
        }
        
        // Limpiar todos los separadores
        const cleanMac = mac.replace(/[^0-9A-Fa-f]/g, '');
        
        // Validar longitud: 8 (4 pares), 10 (5 pares), o 12 (6 pares)
        if (![8, 10, 12].includes(cleanMac.length)) {
            console.error(`Invalid MAC length: ${cleanMac.length}`);
            return fallback;
        }
        
        // Insertar colons cada 2 caracteres
        return cleanMac.match(/.{2}/g).join(':').toUpperCase();
    }
    
    static validateMessageFields(message) {
        // Validate timestamp
        if (!Number.isInteger(message.timestamp) || message.timestamp < 0) {
            return false;
        }
        
        // Validate interval for recurrent messages
        if (message.recurrent && (!Number.isInteger(message.interval) || message.interval <= 0)) {
            return false;
        }
        
        // Validate start address
        const startAddress = parseInt(message.start_address, 16);
        if (message.function_code !== 43 && 
            (isNaN(startAddress) || startAddress < 0x0000 || startAddress > 0xFFFF)) {
            return false;
        }
        
        // Validate function code
        if (!CONSTANTS.FUNCTION_CODES.includes(message.function_code)) {
            return false;
        }
        
        // Validate count for read functions
        if ([1, 2, 3, 4].includes(message.function_code) && !Number.isInteger(message.count)) {
            return false;
        }
        
        // Validate values for write functions
        if ([5, 6, 15, 16].includes(message.function_code) && 
            (!Array.isArray(message.values) || !message.values.every(v => typeof v === 'number'))) {
            return false;
        }
        
        return true;
    }
    
    static checkCharacters(str, includeColon) {
        const specialChars = includeColon ? /[^0-9:\s,]/ : /[^0-9\s,]/;
        return !specialChars.test(str);
    }
    
    static parseRegisterValues(type, values) {
        if (!values) return -1;
        
        const includeColon = type === CONSTANTS.REGISTER_TYPES.SPARSE;
        
        if (!this.checkCharacters(values, includeColon)) {
            UIUtils.showError('Error: Incorrect register value format');
            return null;
        }
        
        if (type === CONSTANTS.REGISTER_TYPES.SEQUENTIAL) {
            return values.split(',').map(Number);
        } else {
            // Sparse format
            const result = values.split(',')
                .map(pair => pair.split(':').map(item => item.trim()))
                .reduce((acc, [key, value]) => {
                    acc[key] = parseInt(value, 10);
                    return acc;
                }, {});
            
            for (const key in result) {
                if (parseInt(key, 10) <= 0) {
                    UIUtils.showError('Error: Register key must be greater than 0');
                    return null;
                }
            }
            
            return result;
        }
    }
}
