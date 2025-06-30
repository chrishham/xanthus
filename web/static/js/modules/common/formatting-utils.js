// Formatting Utilities - Common formatting functions for display
export class FormattingUtils {
    // Format date in a consistent way across the application
    static formatDate(dateString) {
        return new Date(dateString).toLocaleDateString('en-US', {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    }

    // Calculate time since creation
    static getTimeSinceCreation(createdString) {
        const created = new Date(createdString);
        const now = new Date();
        const diffMs = now - created;
        
        const days = Math.floor(diffMs / (1000 * 60 * 60 * 24));
        const hours = Math.floor((diffMs % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
        const minutes = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60));
        
        if (days > 0) {
            return `${days}d ${hours}h`;
        } else if (hours > 0) {
            return `${hours}h ${minutes}m`;
        } else {
            return `${minutes}m`;
        }
    }

    // Format memory table output for VPS status displays
    static formatMemoryTable(memoryOutput) {
        if (!memoryOutput || memoryOutput === 'N/A') {
            return '<div class="text-xs text-gray-500">No memory data available</div>';
        }

        const lines = memoryOutput.trim().split('\n');
        if (lines.length < 2) {
            return `<pre class="text-xs font-mono text-gray-700">${memoryOutput}</pre>`;
        }

        // Parse the structured memory output
        let tableHTML = '<table class="w-full text-xs">';
        
        // Add header row
        tableHTML += '<thead><tr class="border-b border-gray-300">';
        tableHTML += '<th class="text-left py-1 px-2 font-semibold text-gray-700">Type</th>';
        tableHTML += '<th class="text-left py-1 px-2 font-semibold text-gray-700">Total</th>';
        tableHTML += '<th class="text-left py-1 px-2 font-semibold text-gray-700">Used</th>';
        tableHTML += '<th class="text-left py-1 px-2 font-semibold text-gray-700">Free</th>';
        tableHTML += '<th class="text-left py-1 px-2 font-semibold text-gray-700">Available</th>';
        tableHTML += '<th class="text-left py-1 px-2 font-semibold text-gray-700">Buff/Cache</th>';
        tableHTML += '</tr></thead>';

        // Add data rows
        tableHTML += '<tbody>';
        
        let currentType = '';
        lines.forEach(line => {
            if (line.includes('Memory Usage:')) {
                currentType = 'Memory';
            } else if (line.includes('Swap Usage:')) {
                currentType = 'Swap';
            } else if (line.includes('Total:') && line.includes('Used:') && currentType) {
                // Extract values using regex
                const totalMatch = line.match(/Total:\s*([0-9.]+G)/);
                const usedMatch = line.match(/Used:\s*([0-9.]+G)/);
                const freeMatch = line.match(/Free:\s*([0-9.]+G)/);
                const availableMatch = line.match(/Available:\s*([0-9.]+G)/);
                const buffCacheMatch = line.match(/Buff\/Cache:\s*([0-9.]+G)/);
                
                if (totalMatch && usedMatch && freeMatch) {
                    tableHTML += '<tr class="border-b border-gray-200">';
                    tableHTML += `<td class="py-1 px-2 font-medium text-gray-800">${currentType}</td>`;
                    tableHTML += `<td class="py-1 px-2 text-gray-600">${totalMatch[1]}</td>`;
                    tableHTML += `<td class="py-1 px-2 text-gray-600">${usedMatch[1]}</td>`;
                    tableHTML += `<td class="py-1 px-2 text-gray-600">${freeMatch[1]}</td>`;
                    tableHTML += `<td class="py-1 px-2 text-gray-600">${availableMatch ? availableMatch[1] : '-'}</td>`;
                    tableHTML += `<td class="py-1 px-2 text-gray-600">${buffCacheMatch ? buffCacheMatch[1] : '-'}</td>`;
                    tableHTML += '</tr>';
                }
            }
        });
        
        tableHTML += '</tbody></table>';

        return tableHTML;
    }

    // Format disk table output for VPS status displays
    static formatDiskTable(diskOutput) {
        if (!diskOutput || diskOutput === 'N/A') {
            return '<div class="text-xs text-gray-500">No disk data available</div>';
        }

        const lines = diskOutput.trim().split('\n');
        if (lines.length < 2) {
            return `<pre class="text-xs font-mono text-gray-700">${diskOutput}</pre>`;
        }

        // Parse header and data lines
        const headerLine = lines[0];
        const dataLines = lines.slice(1);

        // Handle "Mounted on" as a single column
        let headers;
        if (headerLine.includes('Mounted on')) {
            // Split carefully to keep "Mounted on" together
            const parts = headerLine.trim().split(/\s+/);
            const mountedIndex = parts.findIndex(part => part === 'Mounted');
            if (mountedIndex !== -1 && parts[mountedIndex + 1] === 'on') {
                headers = [...parts.slice(0, mountedIndex), 'Mounted on', ...parts.slice(mountedIndex + 2)];
            } else {
                headers = parts;
            }
        } else {
            headers = headerLine.trim().split(/\s+/);
        }
        
        let tableHTML = '<table class="w-full text-xs">';
        
        // Add header row
        tableHTML += '<thead><tr class="border-b border-gray-300">';
        headers.forEach(header => {
            tableHTML += `<th class="text-left py-1 px-2 font-semibold text-gray-700">${header}</th>`;
        });
        tableHTML += '</tr></thead>';

        // Add data rows
        tableHTML += '<tbody>';
        dataLines.forEach(line => {
            const cells = line.trim().split(/\s+/);
            if (cells.length > 0) {
                tableHTML += '<tr class="border-b border-gray-200">';
                
                // Handle cells to match header count
                let processedCells = [];
                if (headers.includes('Mounted on') && cells.length >= 6) {
                    // For disk output, typically: Filesystem Size Used Avail Use% MountPoint
                    processedCells = [
                        cells[0], // Filesystem
                        cells[1], // Size
                        cells[2], // Used
                        cells[3], // Avail
                        cells[4], // Use%
                        cells.slice(5).join(' ') // Mounted on (join remaining parts)
                    ];
                } else {
                    processedCells = cells;
                }

                processedCells.forEach((cell, index) => {
                    const isFirstCol = index === 0;
                    const cellClass = isFirstCol ? 'font-medium text-gray-800' : 'text-gray-600';
                    tableHTML += `<td class="py-1 px-2 ${cellClass}">${cell}</td>`;
                });
                
                tableHTML += '</tr>';
            }
        });
        tableHTML += '</tbody></table>';

        return tableHTML;
    }

    // Format currency amounts
    static formatCurrency(amount, currency = '€') {
        const num = parseFloat(amount);
        if (isNaN(num)) return `${currency}0.00`;
        return `${currency}${num.toFixed(2)}`;
    }

    // Format resource specifications
    static formatResourceSpec(cores, memory, disk, storageType = 'SSD') {
        return `${cores} vCPU, ${memory}GB RAM, ${disk}GB ${storageType}`;
    }
}

// Make functions available globally for use in templates
if (typeof window !== 'undefined') {
    window.formatMemoryTable = FormattingUtils.formatMemoryTable;
    window.formatDiskTable = FormattingUtils.formatDiskTable;
}

export default FormattingUtils;