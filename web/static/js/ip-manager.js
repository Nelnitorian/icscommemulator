// ============================================================================
// MODULE: IP Address Manager
// ============================================================================

class IPManager {
    static getNextIPInSubnet(ipAddress, subnet) {
        const ip = ipaddr.parse(ipAddress);
        const subnetParsed = ipaddr.parseCIDR(subnet);
        
        if (!ip.match(subnetParsed)) {
            throw new Error("The IP does not belong to the specified subnet.");
        }
        
        const nextIp = ip.toByteArray();
        
        for (let i = nextIp.length - 1; i >= 0; i--) {
            if (nextIp[i] < 255) {
                nextIp[i]++;
                break;
            }
            nextIp[i] = 0;
        }
        
        return ipaddr.fromByteArray(nextIp).toString();
    }
    
    static getUniqueIP(existingIPs, subnet) {
        let newIP = subnet.split('/')[0];
        
        while (existingIPs.includes(newIP)) {
            newIP = this.getNextIPInSubnet(newIP, subnet);
        }
        
        return newIP;
    }
    
    static assignIP(ipCandidate, fallback) {
        const ipValue = ipCandidate.trim();
        
        if (ipValue === '') {
            const existingIPs = cy.nodes().map(node => node.data('ip'));
            existingIPs.push(this.getUniqueIP([], networkData.ip_network));
            return this.getUniqueIP(existingIPs, networkData.ip_network);
        }
        
        if (Validators.validateIP(ipValue)) {
            return ipValue;
        } else {
            console.error('Invalid IP address');
            UIUtils.showError('Error: Invalid IP address');
            return fallback;
        }
    }
}
    