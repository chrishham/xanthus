// VPS Management Module - Alpine.js component

export function vpsManagement() {
    return {
        servers: window.initialServers || [],
        loading: false,
        loadingTitle: 'Processing...',
        loadingMessage: 'Please wait while the operation completes.',
        autoRefreshEnabled: true,
        refreshInterval: 10000, // 10 seconds - faster for better UX
        intervalId: null,
        adaptivePolling: true,

        init() {
            // Show loading modal during initial data fetch
            this.setLoadingState('Loading VPS Information', 'Fetching server status and details...');
            
            // Fetch initial VPS status and information
            this.fetchInitialVPSData();
            
            // Start automatic refresh
            this.startAutoRefresh();
            
            // Stop polling when page is hidden/unfocused
            document.addEventListener('visibilitychange', () => {
                if (document.hidden) {
                    this.stopAutoRefresh();
                } else {
                    this.startAutoRefresh();
                    // Refresh immediately when page becomes visible
                    this.refreshServersQuietly();
                }
            });
        },

        async fetchInitialVPSData() {
            try {
                // Fetch fresh VPS data from server
                const response = await fetch('/vps/list', {
                    method: 'GET',
                    headers: {
                        'Content-Type': 'application/json',
                    }
                });
                
                if (response.ok) {
                    const data = await response.json();
                    this.servers = data.servers || [];
                } else {
                    console.error('Failed to fetch initial VPS data');
                    // Keep using the initial servers data from the template
                }
            } catch (error) {
                console.error('Error fetching initial VPS data:', error);
                // Keep using the initial servers data from the template
            } finally {
                // Always hide loading modal after initial fetch
                this.loading = false;
            }
        },

        startAutoRefresh() {
            if (this.autoRefreshEnabled && !this.intervalId) {
                this.intervalId = setInterval(() => {
                    this.refreshServersQuietly();
                }, this.getAdaptiveInterval());
            }
        },

        getAdaptiveInterval() {
            if (!this.adaptivePolling) {
                return this.refreshInterval;
            }
            
            // Check if any servers are in transitional states
            const transitionalStates = ['initializing', 'starting', 'stopping'];
            const hasTransitionalServers = this.servers.some(server => 
                transitionalStates.includes(server.status)
            );
            
            // Use faster polling (5 seconds) when servers are transitioning
            return hasTransitionalServers ? 5000 : this.refreshInterval;
        },

        restartAutoRefreshWithNewInterval() {
            if (this.autoRefreshEnabled) {
                this.stopAutoRefresh();
                this.startAutoRefresh();
            }
        },

        stopAutoRefresh() {
            if (this.intervalId) {
                clearInterval(this.intervalId);
                this.intervalId = null;
            }
        },

        toggleAutoRefresh() {
            this.autoRefreshEnabled = !this.autoRefreshEnabled;
            if (this.autoRefreshEnabled) {
                this.startAutoRefresh();
            } else {
                this.stopAutoRefresh();
            }
        },

        async refreshServersQuietly() {
            // Refresh without showing loading spinner
            try {
                const response = await fetch('/vps/list', {
                    method: 'GET',
                    headers: {
                        'Content-Type': 'application/json',
                    }
                });
                
                if (response.ok) {
                    const data = await response.json();
                    const oldServers = this.servers;
                    this.servers = data.servers || [];
                    
                    // Check if we need to adjust polling interval
                    if (this.adaptivePolling) {
                        const oldHasTransitional = oldServers.some(s => 
                            ['initializing', 'starting', 'stopping'].includes(s.status)
                        );
                        const newHasTransitional = this.servers.some(s => 
                            ['initializing', 'starting', 'stopping'].includes(s.status)
                        );
                        
                        // Restart with new interval if transitional state changed
                        if (oldHasTransitional !== newHasTransitional) {
                            this.restartAutoRefreshWithNewInterval();
                        }
                    }
                }
            } catch (error) {
                console.log('Background refresh failed:', error);
                // Silently fail for background updates
            }
        },

        async refreshServers() {
            this.setLoadingState('Refreshing Servers', 'Loading server list...');
            try {
                const response = await fetch('/vps/list', {
                    method: 'GET',
                    headers: {
                        'Content-Type': 'application/json',
                    }
                });
                
                if (response.ok) {
                    const data = await response.json();
                    this.servers = data.servers || [];
                } else {
                    console.error('Failed to refresh servers');
                    Swal.fire('Error', 'Failed to refresh server list', 'error');
                }
            } catch (error) {
                console.error('Error refreshing servers:', error);
                Swal.fire('Error', 'Failed to refresh server list', 'error');
            } finally {
                this.loading = false;
            }
        },

        async showCreateServerModal() {
            // First, load server options
            try {
                const response = await fetch('/vps/server-options');
                const data = await response.json();
                
                if (!response.ok) {
                    throw new Error(data.error || 'Failed to load server options');
                }
                
                const { locations, serverTypes } = data;
                
                // Build location options
                const locationOptions = locations.map(loc => 
                    `<option value="${loc.name}">${loc.description} (${loc.name})</option>`
                ).join('');
                
                // Build server type options with monthly pricing
                const serverTypeOptions = serverTypes.map(type => {
                    const hourlyPrice = parseFloat(type.prices.find(p => p.location === 'nbg1' && p.price_hourly).price_hourly.net);
                    const monthlyPrice = (hourlyPrice * 24 * 30).toFixed(2); // 30 days approximation
                    return `<option value="${type.name}">${type.name} - ${type.cores} vCPU, ${type.memory}GB RAM, ${type.disk}GB SSD - €${monthlyPrice}/month</option>`;
                }).join('');
                
                const { value: formValues } = await Swal.fire({
                    title: 'Create New VPS',
                    html: `
                        <div class="text-left space-y-4">
                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">Server Name</label>
                                <input id="server-name" class="swal2-input w-full" placeholder="my-k3s-server" value="xanthus-k3s-${Date.now()}">
                            </div>
                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">Location</label>
                                <select id="server-location" class="swal2-select w-full">
                                    <option value="">Select a location</option>
                                    ${locationOptions}
                                </select>
                            </div>
                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">Server Type</label>
                                <select id="server-type" class="swal2-select w-full">
                                    <option value="">Select a server type</option>
                                    ${serverTypeOptions}
                                </select>
                            </div>
                            <div class="p-3 bg-blue-50 rounded-md text-sm">
                                <strong>OS:</strong> Ubuntu 24.04 LTS<br>
                                <strong>K3s:</strong> Auto-installed with SSL<br>
                                <strong>SSL:</strong> Cloudflare certificates included<br>
                                <strong>SSH:</strong> RSA key authentication
                            </div>
                            <div class="p-3 bg-amber-50 border border-amber-200 rounded-md text-sm">
                                <div class="flex items-start">
                                    <svg class="h-4 w-4 text-amber-400 mt-0.5 mr-2 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                                        <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
                                    </svg>
                                    <div>
                                        <strong class="text-amber-800">Additional Costs:</strong><br>
                                        <span class="text-amber-700">IPv4 address: €0.50/month (required for internet access)</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    `,
                    showCancelButton: true,
                    confirmButtonText: 'Create VPS',
                    confirmButtonColor: '#2563eb',
                    preConfirm: () => {
                        const name = document.getElementById('server-name').value;
                        const location = document.getElementById('server-location').value;
                        const serverType = document.getElementById('server-type').value;
                        
                        if (!name) {
                            Swal.showValidationMessage('Server name is required');
                            return false;
                        }
                        if (!location) {
                            Swal.showValidationMessage('Location is required');
                            return false;
                        }
                        if (!serverType) {
                            Swal.showValidationMessage('Server type is required');
                            return false;
                        }
                        
                        return { name, location, serverType };
                    }
                });

                if (formValues) {
                    await this.createServer(formValues.name, formValues.location, formValues.serverType);
                }
            } catch (error) {
                console.error('Error loading server options:', error);
                Swal.fire('Error', 'Failed to load server options. Please check your Hetzner API key configuration.', 'error');
            }
        },

        async createServer(name, location, serverType) {
            this.setLoadingState('Creating VPS', `Creating VPS "${name}"...`);
            try {
                const response = await fetch('/vps/create', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    body: `name=${encodeURIComponent(name)}&location=${encodeURIComponent(location)}&server_type=${encodeURIComponent(serverType)}`
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    Swal.fire({
                        title: 'Success!',
                        text: `VPS "${name}" is being created. This may take a few minutes.`,
                        icon: 'success',
                        confirmButtonColor: '#2563eb'
                    });
                    await this.refreshServers();
                } else {
                    Swal.fire('Error', data.error || 'Failed to create VPS', 'error');
                }
            } catch (error) {
                console.error('Error creating server:', error);
                Swal.fire('Error', 'Failed to create VPS', 'error');
            } finally {
                this.loading = false;
            }
        },

        async confirmDeleteServer(serverId, serverName) {
            const result = await Swal.fire({
                title: 'Delete VPS?',
                text: `Are you sure you want to delete "${serverName}"? This action cannot be undone.`,
                icon: 'warning',
                showCancelButton: true,
                confirmButtonColor: '#dc2626',
                cancelButtonColor: '#6b7280',
                confirmButtonText: 'Yes, delete it!'
            });

            if (result.isConfirmed) {
                await this.deleteServer(serverId, serverName);
            }
        },

        async deleteServer(serverId, serverName) {
            this.setLoadingState('Deleting VPS', `Deleting VPS "${serverName}"...`);
            try {
                const response = await fetch('/vps/delete', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    body: `server_id=${serverId}`
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    Swal.fire('Deleted!', `VPS "${serverName}" has been deleted.`, 'success');
                    await this.refreshServers();
                } else {
                    Swal.fire('Error', data.error || 'Failed to delete VPS', 'error');
                }
            } catch (error) {
                console.error('Error deleting server:', error);
                Swal.fire('Error', 'Failed to delete VPS', 'error');
            } finally {
                this.loading = false;
            }
        },

        async powerOffServer(serverId, serverName) {
            await this.performServerAction('poweroff', serverId, serverName, 'powering off');
        },

        async powerOnServer(serverId, serverName) {
            await this.performServerAction('poweron', serverId, serverName, 'powering on');
        },

        async rebootServer(serverId, serverName) {
            await this.performServerAction('reboot', serverId, serverName, 'rebooting');
        },

        async performServerAction(action, serverId, serverName, actionText) {
            const actionTitles = {
                'poweroff': 'Powering Off VPS',
                'poweron': 'Powering On VPS', 
                'reboot': 'Rebooting VPS'
            };
            this.setLoadingState(actionTitles[action] || 'VPS Action', `${actionText.charAt(0).toUpperCase() + actionText.slice(1)} "${serverName}"...`);
            try {
                const response = await fetch(`/vps/${action}`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    body: `server_id=${serverId}`
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    Swal.fire('Success!', `VPS "${serverName}" is ${actionText}.`, 'success');
                    // Refresh after a short delay to allow status to update
                    setTimeout(() => this.refreshServers(), 2000);
                } else {
                    Swal.fire('Error', data.error || `Failed to perform ${action}`, 'error');
                }
            } catch (error) {
                console.error(`Error performing ${action}:`, error);
                Swal.fire('Error', `Failed to perform ${action}`, 'error');
            } finally {
                this.loading = false;
            }
        },

        formatDate(dateString) {
            return new Date(dateString).toLocaleDateString('en-US', {
                year: 'numeric',
                month: 'short',
                day: 'numeric',
                hour: '2-digit',
                minute: '2-digit'
            });
        },

        getTimeSinceCreation(createdString) {
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
        },

        async downloadSSHKeyFile() {
            try {
                // Create a temporary download link
                const downloadUrl = '/vps/ssh-key?download=true';
                const link = document.createElement('a');
                link.href = downloadUrl;
                link.download = 'xanthus-key.pem';
                document.body.appendChild(link);
                link.click();
                document.body.removeChild(link);
                
                // Update the button to show success
                const btn = document.getElementById('download-ssh-btn');
                if (btn) {
                    const originalText = btn.innerHTML;
                    btn.innerHTML = '✅ Downloaded!';
                    btn.classList.add('bg-green-600', 'hover:bg-green-700');
                    btn.classList.remove('bg-blue-600', 'hover:bg-blue-700');
                    setTimeout(() => {
                        btn.innerHTML = originalText;
                        btn.classList.remove('bg-green-600', 'hover:bg-green-700');
                        btn.classList.add('bg-blue-600', 'hover:bg-blue-700');
                    }, 2000);
                }
            } catch (error) {
                console.error('Error downloading SSH key:', error);
                const btn = document.getElementById('download-ssh-btn');
                if (btn) {
                    const originalText = btn.innerHTML;
                    btn.innerHTML = '❌ Failed';
                    btn.classList.add('bg-red-600', 'hover:bg-red-700');
                    btn.classList.remove('bg-blue-600', 'hover:bg-blue-700');
                    setTimeout(() => {
                        btn.innerHTML = originalText;
                        btn.classList.remove('bg-red-600', 'hover:bg-red-700');
                        btn.classList.add('bg-blue-600', 'hover:bg-blue-700');
                    }, 2000);
                }
            }
        },

        async downloadSSHKey() {
            try {
                // Create a temporary download link
                const downloadUrl = '/vps/ssh-key?download=true';
                const link = document.createElement('a');
                link.href = downloadUrl;
                link.download = 'xanthus-key.pem';
                document.body.appendChild(link);
                link.click();
                document.body.removeChild(link);
                
                Swal.fire({
                    title: 'SSH Key Downloaded!',
                    html: `
                        <div class="text-left text-sm">
                            <p class="mb-2">Your SSH private key has been downloaded as <code>xanthus-key.pem</code></p>
                            <p class="mb-2"><strong>Next steps:</strong></p>
                            <ol class="list-decimal list-inside space-y-1 text-xs">
                                <li>Set correct permissions: <code>chmod 600 xanthus-key.pem</code></li>
                                <li>Connect to your VPS: <code>ssh -i xanthus-key.pem root@&lt;server-ip&gt;</code></li>
                            </ol>
                        </div>
                    `,
                    icon: 'success',
                    confirmButtonText: 'Got it!'
                });
            } catch (error) {
                console.error('Error downloading SSH key:', error);
                Swal.fire('Error', 'Failed to download SSH key', 'error');
            }
        },

        async showSSHInstructions(serverIP) {
            // Show loading modal immediately
            this.setLoadingState('Loading SSH Instructions', 'Fetching SSH key and connection details...');
            
            try {
                const response = await fetch('/vps/ssh-key');
                const data = await response.json();
                
                if (response.ok) {
                    // Hide loading modal before showing SSH instructions
                    this.loading = false;
                    Swal.fire({
                        title: 'SSH Setup & Instructions',
                        html: `
                            <div class="text-left text-sm">
                                <div class="mb-4 p-3 bg-blue-50 border border-blue-200 rounded-md">
                                    <h4 class="font-semibold mb-2 flex items-center">
                                        <span class="mr-2">🖥️</span>
                                        Quick Connect
                                    </h4>
                                    <div class="text-xs text-blue-800 font-mono mb-2 p-2 bg-white rounded border">
                                        ssh -i xanthus-key.pem root@${serverIP}
                                    </div>
                                    <button id="download-ssh-btn" class="w-full px-3 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 text-xs">
                                        📥 Download SSH Key File
                                    </button>
                                </div>
                                
                                <div class="mb-4">
                                    <h4 class="font-semibold mb-2">🔑 SSH Private Key:</h4>
                                    <textarea readonly class="w-full h-32 p-2 text-xs font-mono bg-gray-100 border rounded" onclick="this.select()">${data.private_key}</textarea>
                                    <p class="text-xs text-gray-600 mt-1">Click to select all text, then copy</p>
                                </div>
                                
                                <div class="mb-4">
                                    <h4 class="font-semibold mb-2">📋 Setup Steps:</h4>
                                    <ol class="list-decimal list-inside space-y-1 text-xs">
                                        <li><strong>Option 1 (Download):</strong> Click "Download SSH Key File" above, then move to <code>~/.ssh/xanthus-key.pem</code></li>
                                        <li><strong>Option 2 (Manual):</strong> Copy the key above and save to <code>~/.ssh/xanthus-key.pem</code></li>
                                        <li>Set correct permissions: <code>chmod 600 ~/.ssh/xanthus-key.pem</code></li>
                                        <li>Connect to server: <code>ssh -i ~/.ssh/xanthus-key.pem root@${serverIP}</code></li>
                                    </ol>
                                </div>
                                
                                <div class="p-3 bg-green-50 border border-green-200 rounded-md text-xs">
                                    <div class="flex items-start">
                                        <svg class="h-4 w-4 text-green-400 mt-0.5 mr-2 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                                            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
                                        </svg>
                                        <div>
                                            <strong class="text-green-800">Pro Tip:</strong><br>
                                            <span class="text-green-700">This key works for all your Xanthus-managed VPS instances. Save it once and use it everywhere!</span>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        `,
                        width: 700,
                        showCloseButton: true,
                        confirmButtonText: 'Close',
                        didOpen: () => {
                            // Add event listener for download button
                            document.getElementById('download-ssh-btn').addEventListener('click', () => {
                                this.downloadSSHKeyFile();
                            });
                        }
                    });
                } else {
                    this.loading = false;
                    Swal.fire('Error', data.error || 'Failed to get SSH key', 'error');
                }
            } catch (error) {
                console.error('Error getting SSH instructions:', error);
                this.loading = false;
                Swal.fire('Error', 'Failed to get SSH instructions', 'error');
            }
        },

        async checkVPSStatus(serverId) {
            this.setLoadingState('Health Check', 'Checking VPS status and services...');
            try {
                const response = await fetch(`/vps/${serverId}/status`);
                const data = await response.json();
                
                if (response.ok) {
                    const statusColor = data.reachable ? 'success' : 'error';
                    const k3sStatus = data.k3s_status === 'active' ? '✅ Running' : '❌ ' + data.k3s_status;
                    
                    // Setup status with progress indication
                    const setupStatusColors = {
                        'READY': '✅',
                        'VERIFYING': '🔍',
 
                        'INSTALLING_HELM': '📦',
                        'WAITING_K3S': '⏳',
                        'INSTALLING_K3S': '📦',
                        'INSTALLING': '🚀',
                        'UNKNOWN': '❓'
                    };
                    const setupIcon = setupStatusColors[data.setup_status] || '❓';
                    const setupStatus = `${setupIcon} ${data.setup_status || 'UNKNOWN'}`;
                    const setupMessage = data.setup_message || 'No status information available';
                    
                    // Show auto-refresh button for non-ready states
                    const showAutoRefresh = data.setup_status && data.setup_status !== 'READY';
                    
                    Swal.fire({
                        title: `VPS Status & Health`,
                        html: `
                            <div class="text-left text-sm">
                                <div class="mb-4 p-3 ${data.setup_status === 'READY' ? 'bg-green-50 border border-green-200' : 'bg-blue-50 border border-blue-200'} rounded-md">
                                    <h4 class="font-semibold mb-2">🚀 Setup Status:</h4>
                                    <div class="font-medium">${setupStatus}</div>
                                    <div class="text-xs text-gray-600 mt-1">${setupMessage}</div>
                                    ${showAutoRefresh ? '<div class="text-xs text-blue-600 mt-2">💡 This dialog will auto-refresh every 5 seconds while setup is in progress</div>' : ''}
                                </div>
                                
                                <div class="grid grid-cols-2 gap-4 mb-6">
                                    <div class="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                                        <span class="font-medium text-gray-700">SSH Status:</span>
                                        <span class="font-semibold ml-2">${data.reachable ? '✅ Connected' : '❌ Failed'}</span>
                                    </div>
                                    <div class="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                                        <span class="font-medium text-gray-700">K3s Service:</span>
                                        <span class="font-semibold ml-2">${k3sStatus}</span>
                                    </div>
                                </div>
                                
                                <div class="mb-4">
                                    <h4 class="font-semibold mb-3 flex items-center">
                                        <span class="mr-2">🖥️</span>
                                        System Load
                                    </h4>
                                    <div class="bg-gray-50 p-3 rounded-lg border">
                                        <pre class="text-xs font-mono text-gray-700 whitespace-pre overflow-x-auto leading-relaxed">${data.system_load?.uptime || 'N/A'}</pre>
                                    </div>
                                </div>
                                
                                <div class="mb-4">
                                    <h4 class="font-semibold mb-3 flex items-center">
                                        <span class="mr-2">💾</span>
                                        Memory Usage
                                    </h4>
                                    <div class="bg-gray-50 p-3 rounded-lg border">
                                        ${formatMemoryTable(data.system_load?.memory)}
                                    </div>
                                </div>
                                
                                <div class="mb-4">
                                    <h4 class="font-semibold mb-3 flex items-center">
                                        <span class="mr-2">💿</span>
                                        Disk Usage
                                    </h4>
                                    <div class="bg-gray-50 p-3 rounded-lg border">
                                        ${formatDiskTable(data.disk_usage?.root)}
                                    </div>
                                </div>
                                
                                <div class="mb-4">
                                    <h4 class="font-semibold mb-3 flex items-center">
                                        <span class="mr-2">🔧</span>
                                        Services Status
                                    </h4>
                                    <div class="bg-gray-50 p-3 rounded-lg border">
                                        <div class="space-y-2">
                                            ${Object.entries(data.services || {}).filter(([service, status]) => service !== 'systemd-resolved').map(([service, status]) => 
                                                `<div class="flex items-center justify-between text-sm">
                                                    <span class="font-medium text-gray-700">${service}:</span>
                                                    <span class="font-semibold ml-2">${status === 'active' ? '✅ Active' : '❌ ' + status}</span>
                                                </div>`
                                            ).join('')}
                                        </div>
                                    </div>
                                </div>
                                
                                <div class="mb-4">
                                    <button id="view-logs-btn" class="w-full px-4 py-2 bg-purple-600 text-white rounded hover:bg-purple-700 focus:outline-none focus:ring-2 focus:ring-purple-500 text-sm">
                                        📝 View K3s Logs
                                    </button>
                                </div>
                            </div>
                        `,
                        icon: statusColor,
                        width: 600,
                        confirmButtonText: 'Close',
                        didOpen: () => {
                            // Add event listener for View Logs button
                            document.getElementById('view-logs-btn').addEventListener('click', () => {
                                Swal.close();
                                this.showK3sLogs(serverId);
                            });
                            
                            // Auto-refresh for non-ready states
                            if (showAutoRefresh) {
                                const refreshInterval = setInterval(async () => {
                                    try {
                                        const refreshResponse = await fetch(`/vps/${serverId}/status`);
                                        const refreshData = await refreshResponse.json();
                                        
                                        if (refreshResponse.ok) {
                                            const newSetupIcon = setupStatusColors[refreshData.setup_status] || '❓';
                                            const newSetupStatus = `${newSetupIcon} ${refreshData.setup_status || 'UNKNOWN'}`;
                                            const newSetupMessage = refreshData.setup_message || 'No status information available';
                                            
                                            // Update the setup status section
                                            const statusDiv = document.querySelector('.swal2-html-container div div');
                                            if (statusDiv) {
                                                statusDiv.innerHTML = `
                                                    <h4 class="font-semibold mb-2">🚀 Setup Status:</h4>
                                                    <div class="font-medium">${newSetupStatus}</div>
                                                    <div class="text-xs text-gray-600 mt-1">${newSetupMessage}</div>
                                                    ${refreshData.setup_status !== 'READY' ? '<div class="text-xs text-blue-600 mt-2">💡 This dialog will auto-refresh every 5 seconds while setup is in progress</div>' : '<div class="text-xs text-green-600 mt-2">🎉 Setup completed! All components are ready.</div>'}
                                                `;
                                                
                                                // Change background color when ready
                                                if (refreshData.setup_status === 'READY') {
                                                    statusDiv.parentElement.className = 'mb-4 p-3 bg-green-50 border border-green-200 rounded-md';
                                                    clearInterval(refreshInterval);
                                                }
                                            }
                                        }
                                    } catch (error) {
                                        console.log('Auto-refresh failed:', error);
                                    }
                                }, 5000);
                                
                                // Clear interval when modal is closed
                                const observer = new MutationObserver((mutations) => {
                                    mutations.forEach((mutation) => {
                                        if (!document.querySelector('.swal2-container')) {
                                            clearInterval(refreshInterval);
                                            observer.disconnect();
                                        }
                                    });
                                });
                                observer.observe(document.body, { childList: true, subtree: true });
                            }
                        }
                    });
                } else {
                    Swal.fire('Error', data.error || 'Failed to check VPS status', 'error');
                }
            } catch (error) {
                console.error('Error checking VPS status:', error);
                Swal.fire('Error', 'Failed to check VPS status', 'error');
            } finally {
                this.loading = false;
            }
        },

        async showK3sLogs(serverId) {
            this.setLoadingState('Loading K3s Logs', 'Retrieving K3s service logs...');
            try {
                const response = await fetch(`/vps/${serverId}/k3s-logs?lines=100`);
                const data = await response.json();
                
                if (response.ok) {
                    Swal.fire({
                        title: 'K3s Service Logs',
                        html: `
                            <div class="text-left">
                                <div class="mb-2 text-sm text-gray-600">
                                    Server: ${serverId} | Lines: ${data.lines}
                                </div>
                                <textarea readonly class="w-full h-96 p-2 text-xs font-mono bg-gray-100 border rounded" onclick="this.select()">${data.logs}</textarea>
                                <p class="text-xs text-gray-600 mt-1">Click to select all logs</p>
                            </div>
                        `,
                        width: 800,
                        showCloseButton: true,
                        confirmButtonText: 'Close'
                    });
                } else {
                    Swal.fire('Error', data.error || 'Failed to get K3s logs', 'error');
                }
            } catch (error) {
                console.error('Error getting K3s logs:', error);
                Swal.fire('Error', 'Failed to get K3s logs', 'error');
            } finally {
                this.loading = false;
            }
        },

        async showVPSLogs(serverId) {
            this.setLoadingState('Loading Logs', 'Retrieving VPS system logs...');
            try {
                const response = await fetch(`/vps/${serverId}/logs?lines=100`);
                const data = await response.json();
                
                if (response.ok) {
                    Swal.fire({
                        title: 'VPS System Logs',
                        html: `
                            <div class="text-left">
                                <div class="mb-2 text-sm text-gray-600">
                                    Server: ${serverId} | Lines: ${data.lines}
                                </div>
                                <textarea readonly class="w-full h-96 p-2 text-xs font-mono bg-gray-100 border rounded" onclick="this.select()">${data.logs}</textarea>
                                <p class="text-xs text-gray-600 mt-1">Click to select all logs</p>
                            </div>
                        `,
                        width: 800,
                        showCloseButton: true,
                        confirmButtonText: 'Close'
                    });
                } else {
                    Swal.fire('Error', data.error || 'Failed to get VPS logs', 'error');
                }
            } catch (error) {
                console.error('Error getting VPS logs:', error);
                Swal.fire('Error', 'Failed to get VPS logs', 'error');
            } finally {
                this.loading = false;
            }
        },

        async showConfigureVPS(serverId) {
            // First, get the list of available domains
            try {
                const domainsResponse = await fetch('/dns/list');
                const domainsData = await domainsResponse.json();
                
                if (!domainsResponse.ok) {
                    Swal.fire('Error', 'Failed to load domains', 'error');
                    return;
                }

                const domains = domainsData.domains || [];
                const managedDomains = domains.filter(d => d.managed_by_xanthus);
                
                if (managedDomains.length === 0) {
                    Swal.fire('No Domains', 'No managed domains found. Please configure SSL for a domain first.', 'warning');
                    return;
                }

                const domainOptions = managedDomains.map(d => 
                    `<option value="${d.name}">${d.name}</option>`
                ).join('');

                const { value: formValues } = await Swal.fire({
                    title: 'Configure VPS',
                    html: `
                        <div class="text-left">
                            <div class="mb-4">
                                <label class="block text-sm font-medium text-gray-700 mb-2">
                                    Select Domain for SSL Configuration:
                                </label>
                                <select id="domain-select" class="w-full p-2 border border-gray-300 rounded-md">
                                    ${domainOptions}
                                </select>
                            </div>
                            <div class="text-sm text-gray-600">
                                This will update the K3s cluster with SSL certificates for the selected domain.
                            </div>
                        </div>
                    `,
                    focusConfirm: false,
                    showCancelButton: true,
                    confirmButtonText: 'Configure',
                    cancelButtonText: 'Cancel',
                    preConfirm: () => {
                        const domain = document.getElementById('domain-select').value;
                        return { domain };
                    }
                });

                if (formValues) {
                    await this.configureVPS(serverId, formValues.domain);
                }
            } catch (error) {
                console.error('Error showing configure dialog:', error);
                Swal.fire('Error', 'Failed to load configuration dialog', 'error');
            }
        },

        async configureVPS(serverId, domain) {
            this.setLoadingState('Configuring VPS', `Configuring SSL for domain "${domain}"...`);
            try {
                const response = await fetch(`/vps/${serverId}/configure`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    body: `domain=${encodeURIComponent(domain)}`
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    Swal.fire('Success!', data.message, 'success');
                } else {
                    Swal.fire('Error', data.error || 'Failed to configure VPS', 'error');
                }
            } catch (error) {
                console.error('Error configuring VPS:', error);
                Swal.fire('Error', 'Failed to configure VPS', 'error');
            } finally {
                this.loading = false;
            }
        },

        setLoadingState(title, message) {
            this.loadingTitle = title;
            this.loadingMessage = message;
            this.loading = true;
        },

        async openTerminal(serverId, serverName) {
            this.setLoadingState('Opening Terminal', `Creating WebSocket terminal for "${serverName}"...`);
            
            try {
                // Get server details for SSH connection
                const server = this.servers.find(s => s.id === serverId);
                if (!server) {
                    throw new Error('Server not found');
                }

                // Get SSH key
                const keyResponse = await fetch('/vps/ssh-key');
                const keyData = await keyResponse.json();
                if (!keyResponse.ok) {
                    throw new Error(keyData.error || 'Failed to get SSH key');
                }

                // Dynamically import terminal module with cache-busting
                const { webSocketTerminal } = await import(`./terminal.js?v=${Date.now()}`);
                
                // Create WebSocket terminal instance
                const terminal = webSocketTerminal();
                
                // Extract the correct IP address path
                const publicIp = server.public_net?.ipv4?.ip;
                
                // Validate server data
                if (!publicIp) {
                    throw new Error('Server does not have a public IPv4 address');
                }


                // Create terminal session
                const sessionData = await terminal.createTerminalSession({
                    serverId: serverId,
                    host: publicIp,
                    user: 'root',
                    privateKey: keyData.private_key
                });

                // Create unique container ID for this terminal
                const containerId = `terminal-${sessionData.session_id}`;
                
                // Open terminal in modal with xterm.js
                Swal.fire({
                    title: `WebSocket Terminal - ${serverName}`,
                    html: `
                        <div class="text-left">
                            <div class="mb-4 text-sm text-gray-600">
                                Server: ${serverName} | Session: ${sessionData.session_id}
                            </div>
                            <div id="${containerId}" style="height: 500px; background: #000; border-radius: 4px;"></div>
                            <div class="mt-2 text-xs text-gray-500">
                                WebSocket terminal - session will auto-close when dialog is closed.
                            </div>
                        </div>
                    `,
                    width: 900,
                    showCloseButton: true,
                    showConfirmButton: false,
                    didOpen: () => {
                        // Initialize terminal in the modal
                        if (terminal.initTerminal(containerId)) {
                            // Connect to WebSocket session
                            terminal.connectToSession(sessionData.session_id);
                        }
                    },
                    willClose: () => {
                        // Clean up terminal session and WebSocket
                        terminal.destroy();
                        terminal.stopTerminalSession(sessionData.session_id)
                            .catch(err => console.log('Failed to cleanup terminal session:', err));
                    }
                });

            } catch (error) {
                console.error('Error opening WebSocket terminal:', error);
                Swal.fire('Error', error.message || 'Failed to open terminal', 'error');
            } finally {
                this.loading = false;
            }
        },

        async openTerminalNewTab(serverId, serverName) {
            this.setLoadingState('Opening Terminal', `Creating WebSocket terminal for "${serverName}"...`);
            
            try {
                // Get server details for SSH connection
                const server = this.servers.find(s => s.id === serverId);
                if (!server) {
                    throw new Error('Server not found');
                }

                // Get SSH key
                const keyResponse = await fetch('/vps/ssh-key');
                const keyData = await keyResponse.json();
                if (!keyResponse.ok) {
                    throw new Error(keyData.error || 'Failed to get SSH key');
                }

                // Dynamically import terminal module with cache-busting
                const { webSocketTerminal } = await import(`./terminal.js?v=${Date.now()}`);
                
                // Create WebSocket terminal instance
                const terminal = webSocketTerminal();
                
                // Extract the correct IP address path
                const publicIp = server.public_net?.ipv4?.ip;
                
                // Validate server data
                if (!publicIp) {
                    throw new Error('Server does not have a public IPv4 address');
                }


                // Create terminal session
                const sessionData = await terminal.createTerminalSession({
                    serverId: serverId,
                    host: publicIp,
                    user: 'root',
                    privateKey: keyData.private_key
                });


                // Wait a moment for session to be fully registered
                await new Promise(resolve => setTimeout(resolve, 100));
                
                // Create a standalone terminal page URL
                const terminalUrl = `/terminal-page/${sessionData.session_id}?server=${encodeURIComponent(serverName)}`;
                const newTab = window.open(terminalUrl, '_blank');
                
                if (newTab) {
                    Swal.fire({
                        title: 'WebSocket Terminal Opened',
                        html: `
                            <div class="text-left text-sm">
                                <p class="mb-2">WebSocket terminal opened in new tab for <strong>${serverName}</strong></p>
                                <div class="bg-gray-100 p-2 rounded mb-2">
                                    <strong>Session ID:</strong> ${sessionData.session_id}<br>
                                    <strong>Connection:</strong> WebSocket (production-ready)
                                </div>
                                <p class="text-gray-600">Terminal will work through port 443/80. Close the tab when done.</p>
                            </div>
                        `,
                        icon: 'success',
                        confirmButtonText: 'OK'
                    });
                } else {
                    Swal.fire('Popup Blocked', 'Please allow popups for this site to open terminal in new tab', 'warning');
                }

            } catch (error) {
                console.error('Error opening WebSocket terminal:', error);
                Swal.fire('Error', error.message || 'Failed to open terminal', 'error');
            } finally {
                this.loading = false;
            }
        },

        // Show cost information popup
        showCostInfo() {
            Swal.fire({
                title: '💰 Monthly Cost',
                html: `
                    <div class="text-left text-sm">
                        <div class="mb-4">
                            <p class="text-gray-700 mb-2">The monthly cost includes:</p>
                            <ul class="list-disc list-inside text-gray-600 space-y-1">
                                <li>Server resources (CPU, RAM, storage)</li>
                                <li>IPv4 address (€0.50/month)</li>
                                <li>Bandwidth and network usage</li>
                            </ul>
                        </div>
                        
                        <div class="p-3 bg-blue-50 border border-blue-200 rounded-md">
                            <div class="flex items-start">
                                <svg class="h-4 w-4 text-blue-400 mt-0.5 mr-2 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                                    <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd" />
                                </svg>
                                <div>
                                    <strong class="text-blue-800">Note:</strong><br>
                                    <span class="text-blue-700">This is the standard monthly rate. If you delete the VPS before the month ends, you'll be charged the hourly rate instead.</span>
                                </div>
                            </div>
                        </div>
                    </div>
                `,
                icon: 'info',
                confirmButtonText: 'Got it!',
                confirmButtonColor: '#2563eb',
                width: 500
            });
        },

        // Show accumulated cost information popup
        showAccumulatedCostInfo(server) {
            // Get the hourly rate from the specific server passed as parameter
            const hourlyRate = server?.labels?.hourly_cost || 'N/A';
            
            Swal.fire({
                title: '📊 Accumulated Cost',
                html: `
                    <div class="text-left text-sm">
                        <div class="mb-4">
                            <p class="text-gray-700 mb-2">The accumulated cost represents:</p>
                            <ul class="list-disc list-inside text-gray-600 space-y-1">
                                <li>Total charges since VPS creation</li>
                                <li>Based on actual usage time (hours × hourly rate)</li>
                                <li>Updated in real-time based on current timestamp</li>
                                <li>Includes all server resources and IPv4 costs</li>
                            </ul>
                        </div>
                        
                        <div class="mb-4 p-3 bg-blue-50 border border-blue-200 rounded-md">
                            <div class="flex items-start">
                                <svg class="h-4 w-4 text-blue-400 mt-0.5 mr-2 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                                    <path fill-rule="evenodd" d="M13 6a3 3 0 11-6 0 3 3 0 016 0zM18 8a2 2 0 11-4 0 2 2 0 014 0zM14 15a4 4 0 00-8 0v3h8v-3z" clip-rule="evenodd" />
                                </svg>
                                <div>
                                    <strong class="text-blue-800">Current Hourly Rate:</strong><br>
                                    <span class="text-blue-700">€${hourlyRate}/hour (includes server resources + IPv4 address cost)</span>
                                </div>
                            </div>
                        </div>
                        
                        <div class="p-3 bg-amber-50 border border-amber-200 rounded-md">
                            <div class="flex items-start">
                                <svg class="h-4 w-4 text-amber-400 mt-0.5 mr-2 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                                    <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
                                </svg>
                                <div>
                                    <strong class="text-amber-800">Important:</strong><br>
                                    <span class="text-amber-700">This is what you'll be charged if you delete the VPS now, rather than waiting for the full month.</span>
                                </div>
                            </div>
                        </div>
                    </div>
                `,
                icon: 'info',
                confirmButtonText: 'Got it!',
                confirmButtonColor: '#2563eb',
                width: 500
            });
        },


        // Show hourly rate information popup
        showHourlyRateInfo() {
            Swal.fire({
                title: '⏰ Hourly Rate',
                html: `
                    <div class="text-left text-sm">
                        <div class="mb-4">
                            <p class="text-gray-700 mb-2">The hourly rate includes:</p>
                            <ul class="list-disc list-inside text-gray-600 space-y-1">
                                <li>Server resources per hour</li>
                                <li>IPv4 address cost (€0.50/month ÷ 730 hours)</li>
                                <li>All network and bandwidth usage</li>
                            </ul>
                        </div>
                        
                        <div class="mb-4 p-3 bg-green-50 border border-green-200 rounded-md">
                            <div class="flex items-start">
                                <svg class="h-4 w-4 text-green-400 mt-0.5 mr-2 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
                                </svg>
                                <div>
                                    <strong class="text-green-800">Cost Savings:</strong><br>
                                    <span class="text-green-700">Using hourly billing when deleting early can save money compared to paying the full monthly rate.</span>
                                </div>
                            </div>
                        </div>
                        
                        <div class="p-3 bg-blue-50 border border-blue-200 rounded-md">
                            <div class="flex items-start">
                                <svg class="h-4 w-4 text-blue-400 mt-0.5 mr-2 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                                    <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd" />
                                </svg>
                                <div>
                                    <strong class="text-blue-800">Billing:</strong><br>
                                    <span class="text-blue-700">Hetzner charges hourly until you reach the monthly cap, then switches to monthly billing.</span>
                                </div>
                            </div>
                        </div>
                    </div>
                `,
                icon: 'info', 
                confirmButtonText: 'Got it!',
                confirmButtonColor: '#2563eb',
                width: 550
            });
        },

        // Timezone management functions
        async showTimezoneInfo(server) {
            try {
                const response = await fetch(`/vps/${server.id}/timezone`, {
                    method: 'GET',
                    headers: {
                        'Content-Type': 'application/json',
                    }
                });

                if (response.ok) {
                    const data = await response.json();
                    
                    Swal.fire({
                        title: '🕐 Timezone Information',
                        html: `
                            <div class="text-left text-sm">
                                <div class="mb-4 p-4 bg-blue-50 border border-blue-200 rounded-md">
                                    <div class="flex items-center mb-3">
                                        <svg class="h-5 w-5 text-blue-500 mr-2" fill="currentColor" viewBox="0 0 20 20">
                                            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-12a1 1 0 10-2 0v4a1 1 0 00.293.707l2.828 2.829a1 1 0 101.415-1.415L11 9.586V6z" clip-rule="evenodd" />
                                        </svg>
                                        <strong class="text-blue-800">Server Timezone Settings</strong>
                                    </div>
                                    
                                    <div class="space-y-3">
                                        <div>
                                            <strong class="text-gray-700">Current System Timezone:</strong>
                                            <div class="mt-1 p-2 bg-gray-100 rounded border font-mono text-sm">
                                                ${data.current_timezone}
                                            </div>
                                        </div>
                                        
                                        <div>
                                            <strong class="text-gray-700">Configured Timezone:</strong>
                                            <div class="mt-1 p-2 bg-gray-100 rounded border font-mono text-sm">
                                                ${data.config_timezone}
                                            </div>
                                        </div>
                                    </div>
                                </div>
                                
                                <div class="text-xs text-gray-500 mt-3">
                                    <p>💡 <strong>Tip:</strong> Use the timezone manager to change the server's timezone. This will affect all applications and deployments on this server.</p>
                                </div>
                            </div>
                        `,
                        width: '600px',
                        confirmButtonText: 'OK',
                        confirmButtonColor: '#3b82f6',
                        showCloseButton: true
                    });
                } else {
                    throw new Error('Failed to fetch timezone information');
                }
            } catch (error) {
                console.error('Error fetching timezone info:', error);
                Swal.fire({
                    title: 'Error',
                    text: 'Failed to fetch timezone information',
                    icon: 'error'
                });
            }
        },

        async showTimezoneManager(server) {
            // Show loading immediately
            this.setLoadingState('Loading Timezone Manager', 'Fetching timezone information...');
            
            try {
                // Make both API calls in parallel for better performance
                const [timezoneResponse, timezonesResponse] = await Promise.all([
                    fetch(`/vps/${server.id}/timezone`),
                    fetch('/vps/timezones')
                ]);
                
                const timezoneData = await timezoneResponse.json();
                const timezonesData = await timezonesResponse.json();
                
                const timezones = timezonesData.timezones || [];
                const currentTimezone = timezoneData.config_timezone || timezoneData.current_timezone || 'UTC';
                
                // Create timezone options
                const timezoneOptions = timezones.map(tz => 
                    `<option value="${tz}" ${tz === currentTimezone ? 'selected' : ''}>${tz}</option>`
                ).join('');
                
                // Stop loading and show the modal
                this.loading = false;
                
                Swal.fire({
                    title: '🕐 Timezone Manager',
                    html: `
                        <div class="text-left text-sm">
                            <div class="mb-4 p-4 bg-indigo-50 border border-indigo-200 rounded-md">
                                <div class="flex items-center mb-3">
                                    <svg class="h-5 w-5 text-indigo-500 mr-2" fill="currentColor" viewBox="0 0 20 20">
                                        <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-12a1 1 0 10-2 0v4a1 1 0 00.293.707l2.828 2.829a1 1 0 101.415-1.415L11 9.586V6z" clip-rule="evenodd" />
                                    </svg>
                                    <strong class="text-indigo-800">Change Server Timezone</strong>
                                </div>
                                
                                <div class="space-y-3">
                                    <div>
                                        <strong class="text-gray-700">Current Timezone:</strong>
                                        <div class="mt-1 p-2 bg-gray-100 rounded border font-mono text-sm">
                                            ${currentTimezone}
                                        </div>
                                    </div>
                                    
                                    <div>
                                        <label for="timezone-select" class="block text-sm font-medium text-gray-700 mb-2">
                                            Select New Timezone:
                                        </label>
                                        <select id="timezone-select" class="w-full p-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500">
                                            ${timezoneOptions}
                                        </select>
                                    </div>
                                </div>
                            </div>
                            
                            <div class="text-xs text-gray-500 mt-3">
                                <p>⚠️ <strong>Warning:</strong> Changing the timezone will update the VPS system immediately. Existing applications need to be restarted to pick up the new timezone. New deployments will automatically use the new timezone.</p>
                            </div>
                        </div>
                    `,
                    width: '600px',
                    showCancelButton: true,
                    confirmButtonText: 'Change Timezone',
                    cancelButtonText: 'Cancel',
                    confirmButtonColor: '#4f46e5',
                    cancelButtonColor: '#6b7280',
                    showCloseButton: true,
                    preConfirm: () => {
                        const selectedTimezone = document.getElementById('timezone-select').value;
                        if (!selectedTimezone) {
                            Swal.showValidationMessage('Please select a timezone');
                            return false;
                        }
                        return selectedTimezone;
                    }
                }).then(async (result) => {
                    if (result.isConfirmed) {
                        await this.changeTimezone(server, result.value);
                    }
                });
            } catch (error) {
                console.error('Error loading timezone manager:', error);
                this.loading = false;
                
                Swal.fire({
                    title: 'Error',
                    text: 'Failed to load timezone manager',
                    icon: 'error'
                });
            }
        },

        async changeTimezone(server, timezone) {
            try {
                this.setLoadingState('Changing Timezone', `Updating timezone to ${timezone} for ${server.name}...`);
                
                const formData = new FormData();
                formData.append('timezone', timezone);
                
                const response = await fetch(`/vps/${server.id}/timezone`, {
                    method: 'POST',
                    body: formData
                });

                if (response.ok) {
                    const data = await response.json();
                    
                    // Update the server's timezone in the local data
                    const serverIndex = this.servers.findIndex(s => s.id === server.id);
                    if (serverIndex !== -1) {
                        if (!this.servers[serverIndex].labels) {
                            this.servers[serverIndex].labels = {};
                        }
                        this.servers[serverIndex].labels.configured_timezone = timezone;
                    }
                    
                    Swal.fire({
                        title: 'Success!',
                        text: `Timezone changed to ${timezone} successfully`,
                        icon: 'success',
                        confirmButtonColor: '#10b981'
                    });
                } else {
                    const errorData = await response.json();
                    throw new Error(errorData.message || 'Failed to change timezone');
                }
            } catch (error) {
                console.error('Error changing timezone:', error);
                Swal.fire({
                    title: 'Error',
                    text: `Failed to change timezone: ${error.message}`,
                    icon: 'error'
                });
            } finally {
                this.loading = false;
            }
        },

        // Cleanup on component destroy
        destroy() {
            this.stopAutoRefresh();
        }
    }
}

// Helper function to format memory table output
function formatMemoryTable(memoryOutput) {
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

// Helper function to format disk table output  
function formatDiskTable(diskOutput) {
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