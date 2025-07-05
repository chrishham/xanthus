// Applications Management Module - Alpine.js component
// Version: 2025-07-05-token-support
export function applicationsManagement() {
    return {
        applications: window.initialApplications || [],
        predefinedApps: window.initialPredefinedApps || [],
        loading: false,
        loadingTitle: 'Processing...',
        loadingMessage: 'Please wait while the operation completes.',
        autoRefreshEnabled: true, // Enabled by default for real-time status updates
        refreshInterval: 30000, // 30 seconds - good balance between freshness and performance
        intervalId: null,
        countdownInterval: null,
        countdown: 0,
        isRefreshing: false, // Prevent concurrent requests
        
        // Port forwarding modal state
        portForwardingModal: {
            show: false,
            app: null,
            domain: '',
            ports: [],
            newPort: { port: '', subdomain: '' }
        },

        init() {
            // Initialize applications list
            this.refreshApplications();
            
            // Start automatic refresh
            this.startAutoRefresh();
            
            // Stop polling when page is hidden/unfocused
            document.addEventListener('visibilitychange', () => {
                if (document.hidden) {
                    this.stopAutoRefresh();
                } else {
                    this.startAutoRefresh();
                    // Refresh immediately when page becomes visible
                    this.refreshApplicationsQuietly();
                }
            });
        },

        async refreshApplications() {
            if (this.isRefreshing) return; // Prevent concurrent requests
            this.isRefreshing = true;
            this.setLoadingState('Loading Applications', 'Retrieving application list...');
            try {
                const response = await fetch('/applications/list', {
                    method: 'GET',
                    headers: {
                        'Content-Type': 'application/json',
                    }
                });
                
                if (response.ok) {
                    const data = await response.json();
                    
                    // Filter out any invalid applications and deduplicate by ID
                    const validApps = (data.applications || []).filter(app => this.isValidApplication(app));
                    
                    // Deduplicate applications by ID to prevent duplicates
                    const uniqueApps = [];
                    const seenIds = new Set();
                    
                    for (const app of validApps) {
                        if (!seenIds.has(app.id)) {
                            seenIds.add(app.id);
                            uniqueApps.push(app);
                        }
                    }
                    
                    this.applications = uniqueApps;
                } else {
                    console.error('Failed to refresh applications');
                    Swal.fire('Error', 'Failed to refresh application list', 'error');
                }
            } catch (error) {
                console.error('Error refreshing applications:', error);
                Swal.fire('Error', 'Failed to refresh application list', 'error');
            } finally {
                this.loading = false;
                this.isRefreshing = false;
            }
        },

        startAutoRefresh() {
            if (this.autoRefreshEnabled && !this.intervalId) {
                // Start countdown
                this.startCountdown();
                
                this.intervalId = setInterval(() => {
                    // Only refresh if page is visible and not already refreshing
                    if (!document.hidden && !this.isRefreshing) {
                        this.refreshApplicationsQuietly();
                        // Restart countdown after refresh
                        this.startCountdown();
                    }
                }, this.refreshInterval);
            }
        },

        stopAutoRefresh() {
            if (this.intervalId) {
                clearInterval(this.intervalId);
                this.intervalId = null;
            }
            if (this.countdownInterval) {
                clearInterval(this.countdownInterval);
                this.countdownInterval = null;
                this.countdown = 0;
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

        startCountdown() {
            // Stop existing countdown
            if (this.countdownInterval) {
                clearInterval(this.countdownInterval);
            }
            
            // Start new countdown
            this.countdown = Math.floor(this.refreshInterval / 1000);
            this.countdownInterval = setInterval(() => {
                this.countdown--;
                if (this.countdown <= 0) {
                    clearInterval(this.countdownInterval);
                    this.countdownInterval = null;
                }
            }, 1000);
        },

        async refreshApplicationsQuietly() {
            // Refresh without showing loading spinner
            if (this.isRefreshing) return; // Prevent concurrent requests
            this.isRefreshing = true;
            try {
                const response = await fetch('/applications/list', {
                    method: 'GET',
                    headers: {
                        'Content-Type': 'application/json',
                    }
                });
                
                if (response.ok) {
                    const data = await response.json();
                    
                    // Filter out any invalid applications and deduplicate by ID
                    const validApps = (data.applications || []).filter(app => this.isValidApplication(app));
                    
                    // Deduplicate applications by ID to prevent duplicates
                    const uniqueApps = [];
                    const seenIds = new Set();
                    
                    for (const app of validApps) {
                        if (!seenIds.has(app.id)) {
                            seenIds.add(app.id);
                            uniqueApps.push(app);
                        }
                    }
                    
                    this.applications = uniqueApps;
                } else if (response.status === 401 || response.status === 403) {
                    // Authentication failed - stop auto-refresh and redirect to login
                    this.stopAutoRefresh();
                    console.log('Authentication expired, redirecting to login');
                    window.location.href = '/login';
                    return;
                }
            } catch (error) {
                // Handle network errors gracefully
                if (error.name === 'TypeError' && error.message.includes('NetworkError')) {
                    console.log('Network error during background refresh - stopping auto-refresh');
                    this.stopAutoRefresh();
                } else {
                    console.log('Background refresh failed:', error);
                }
                // Silently fail for background updates
            } finally {
                this.isRefreshing = false;
            }
        },

        async deployApplication(predefinedApp) {
            // Check prerequisites first
            this.setLoadingState('Checking Prerequisites', 'Verifying VPS instances and domains...');
            
            try {
                const response = await fetch('/applications/prerequisites');
                const data = await response.json();
                
                if (!response.ok) {
                    throw new Error(data.error || 'Failed to check prerequisites');
                }
                
                const { domains, servers } = data;
                
                // Check if we have domains and servers
                if (!domains || domains.length === 0) {
                    this.loading = false;
                    Swal.fire({
                        title: 'Setup Required',
                        html: `
                            <div class="text-left">
                                <p class="mb-4">No managed domains found. You need to configure SSL for at least one domain first.</p>
                                <div class="p-3 bg-blue-50 rounded-md text-sm">
                                    <strong>Next steps:</strong><br>
                                    1. Go to <a href="/dns" class="text-blue-600 underline">DNS Config</a><br>
                                    2. Configure SSL for your domain<br>
                                    3. Return to deploy applications
                                </div>
                            </div>
                        `,
                        icon: 'warning',
                        confirmButtonText: 'Go to DNS Config',
                        showCancelButton: true,
                        cancelButtonText: 'Cancel'
                    }).then((result) => {
                        if (result.isConfirmed) {
                            window.location.href = '/dns';
                        }
                    });
                    return;
                }
                
                if (!servers || servers.length === 0) {
                    this.loading = false;
                    Swal.fire({
                        title: 'Setup Required',
                        html: `
                            <div class="text-left">
                                <p class="mb-4">No VPS instances found. You need to create at least one VPS with K3s first.</p>
                                <div class="p-3 bg-blue-50 rounded-md text-sm">
                                    <strong>Next steps:</strong><br>
                                    1. Go to <a href="/vps" class="text-blue-600 underline">VPS Management</a><br>
                                    2. Create a new VPS with K3s<br>
                                    3. Return to deploy applications
                                </div>
                            </div>
                        `,
                        icon: 'warning',
                        confirmButtonText: 'Go to VPS Management',
                        showCancelButton: true,
                        cancelButtonText: 'Cancel'
                    }).then((result) => {
                        if (result.isConfirmed) {
                            window.location.href = '/vps';
                        }
                    });
                    return;
                }
                
                // Show deployment form
                await this.showDeploymentForm(predefinedApp, domains, servers);
                
            } catch (error) {
                console.error('Error checking prerequisites:', error);
                Swal.fire('Error', 'Failed to load prerequisites. Please check your configuration.', 'error');
            } finally {
                this.loading = false;
            }
        },

        async showDeploymentForm(predefinedApp, domains, servers) {
            const serverOptions = servers.map(s => 
                `<option value="${s.id}">${s.name} (${s.public_net.ipv4.ip})</option>`
            ).join('');

            const domainOptions = domains.map(d => 
                `<option value="${d.name}">${d.name}</option>`
            ).join('');

            const { value: formValues } = await Swal.fire({
                title: `Deploy ${predefinedApp.name}`,
                html: `
                    <div class="text-left">
                        <div class="flex items-center mb-4 p-3 bg-purple-50 rounded-md">
                            <div class="text-2xl mr-3">${predefinedApp.icon}</div>
                            <div>
                                <h4 class="font-medium text-gray-900">${predefinedApp.name}</h4>
                                <p class="text-sm text-gray-600">${predefinedApp.description}</p>
                            </div>
                        </div>
                        
                        <div class="space-y-4">
                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">Application Name</label>
                                <input id="app-name" class="swal2-input m-0 w-full" placeholder="my-${predefinedApp.id}">
                                <p class="text-xs text-gray-500 mt-1">A friendly name for your application instance</p>
                            </div>
                            
                            <!-- VPS Server Selection - Made more prominent -->
                            <div class="bg-blue-50 border border-blue-200 rounded-lg p-4">
                                <label class="block text-sm font-medium text-blue-900 mb-2">
                                    <span class="flex items-center">
                                        <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z"></path>
                                        </svg>
                                        VPS Server *
                                    </span>
                                </label>
                                <select id="app-vps" class="swal2-select m-0 w-full border-blue-300 focus:border-blue-500 focus:ring-blue-500" style="border: 2px solid #93c5fd;">
                                    <option value="">👆 Click to choose a VPS server</option>
                                    ${serverOptions}
                                </select>
                                <p class="text-xs text-blue-700 mt-1">Select the server where your application will be deployed</p>
                            </div>
                            
                            <!-- Domain Selection - Made more prominent and moved before subdomain -->
                            <div class="bg-green-50 border border-green-200 rounded-lg p-4">
                                <label class="block text-sm font-medium text-green-900 mb-2">
                                    <span class="flex items-center">
                                        <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9v-9m0-9v9"></path>
                                        </svg>
                                        Domain *
                                    </span>
                                </label>
                                <select id="app-domain" class="swal2-select m-0 w-full border-green-300 focus:border-green-500 focus:ring-green-500" style="border: 2px solid #86efac;">
                                    <option value="">👆 Click to select a domain</option>
                                    ${domainOptions}
                                </select>
                                <p class="text-xs text-green-700 mt-1">Choose the domain for your application</p>
                            </div>
                            
                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">Subdomain</label>
                                <input id="app-subdomain" class="swal2-input m-0 w-full" placeholder="${predefinedApp.id}">
                                <p class="text-xs text-gray-500 mt-1">Your app will be available at subdomain.domain.com</p>
                            </div>
                            
                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">Description (optional)</label>
                                <input id="app-description" class="swal2-input m-0 w-full" placeholder="My ${predefinedApp.name} instance">
                            </div>
                            
                            <div class="p-3 bg-green-50 border border-green-200 rounded-md text-sm">
                                <strong>What will be deployed:</strong><br>
                                • ${predefinedApp.name} v${predefinedApp.version}<br>
                                • Automatic HTTPS with SSL certificates<br>
                                • Persistent storage and configuration<br>
                                • Ready to use after deployment
                            </div>
                        </div>
                    </div>
                `,
                showCancelButton: true,
                confirmButtonText: 'Deploy Application',
                cancelButtonText: 'Cancel',
                confirmButtonColor: '#7c3aed',
                width: 750,
                preConfirm: () => {
                    const name = document.getElementById('app-name').value.trim();
                    const vps = document.getElementById('app-vps').value;
                    const subdomain = document.getElementById('app-subdomain').value.trim();
                    const domain = document.getElementById('app-domain').value;
                    const description = document.getElementById('app-description').value.trim();
                    
                    // Clear previous validation styling
                    document.getElementById('app-vps').style.borderColor = '';
                    document.getElementById('app-domain').style.borderColor = '';
                    
                    if (!name) {
                        Swal.showValidationMessage('Application name is required');
                        return false;
                    }
                    if (!vps) {
                        document.getElementById('app-vps').style.borderColor = '#dc2626';
                        document.getElementById('app-vps').style.borderWidth = '2px';
                        Swal.showValidationMessage('VPS server is required');
                        return false;
                    }
                    if (!subdomain) {
                        Swal.showValidationMessage('Subdomain is required');
                        return false;
                    }
                    if (!domain) {
                        document.getElementById('app-domain').style.borderColor = '#dc2626';
                        document.getElementById('app-domain').style.borderWidth = '2px';
                        Swal.showValidationMessage('Domain is required');
                        return false;
                    }
                    
                    return { 
                        name, 
                        vps, 
                        subdomain, 
                        domain, 
                        description,
                        app_type: predefinedApp.id
                    };
                }
            });

            if (formValues) {
                await this.createApplication(formValues);
            }
        },

        async createApplication(formData) {
            this.setLoadingState('Deploying Application', `Deploying "${formData.name}"...`);
            try {
                const response = await fetch('/applications/create', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(formData)
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    // Don't add to local state immediately to avoid duplicates
                    // The refresh after the dialog will handle updating the list
                    
                    // Check if this is a code-server or ArgoCD application with password
                    if ((formData.app_type === 'code-server' || formData.app_type === 'argocd') && data.initial_password) {
                        const appTypeName = formData.app_type === 'argocd' ? 'ArgoCD' : 'Code-Server';
                        const usernameInfo = formData.app_type === 'argocd' ? ' (username: admin)' : '';
                        const appIcon = formData.app_type === 'argocd' ? '🚀' : '💻';
                        const buttonText = formData.app_type === 'argocd' ? 'Open ArgoCD' : 'Open Code-Server';
                        
                        Swal.fire({
                            title: `${appTypeName} Deployed Successfully!`,
                            html: `
                                <div class="text-left">
                                    <p class="mb-4">Your ${appTypeName.toLowerCase()} instance "<strong>${formData.name}</strong>" has been deployed successfully!</p>
                                    
                                    <div class="mb-4 p-4 bg-blue-50 border border-blue-200 rounded-lg">
                                        <h4 class="font-medium text-blue-900 mb-2">${appIcon} Initial Login Password${usernameInfo}</h4>
                                        <div class="flex items-center space-x-2">
                                            <input type="text" value="${data.initial_password}" 
                                                   class="flex-1 px-3 py-2 border border-blue-300 rounded font-mono text-sm bg-white" 
                                                   readonly id="password-field">
                                            <button onclick="navigator.clipboard.writeText('${data.initial_password}'); this.textContent='Copied!'; setTimeout(() => this.textContent='Copy', 1000)" 
                                                    class="px-3 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 text-sm">
                                                Copy
                                            </button>
                                        </div>
                                        <p class="text-sm text-blue-700 mt-2">⚠️ Save this password securely - you'll need it to access your ${appTypeName.toLowerCase()} instance</p>
                                    </div>
                                    
                                    <div class="mb-4 p-4 bg-green-50 border border-green-200 rounded-lg">
                                        <h4 class="font-medium text-green-900 mb-2">🚀 Next Steps</h4>
                                        <ol class="text-sm text-green-800 space-y-1">
                                            <li>1. Visit your ${appTypeName.toLowerCase()} instance</li>
                                            <li>2. Enter the password above to log in${usernameInfo ? ' (username: admin)' : ''}</li>
                                            <li>3. You can change the password later using the "Change Password" button</li>
                                        </ol>
                                    </div>
                                    
                                    <div class="text-center">
                                        <a href="${data.application.url}" target="_blank" 
                                           class="inline-flex items-center px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700">
                                            <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"></path>
                                            </svg>
                                            ${buttonText}
                                        </a>
                                    </div>
                                </div>
                            `,
                            icon: 'success',
                            confirmButtonColor: '#7c3aed',
                            confirmButtonText: 'Got it!',
                            width: 700,
                            allowOutsideClick: false
                        }).then(() => {
                            // Refresh the list after the user dismisses the password dialog
                            // to ensure the application status is up to date
                            this.refreshApplications();
                        });
                    } else {
                        Swal.fire({
                            title: 'Success!',
                            text: `Application "${formData.name}" is being deployed. This may take a few minutes.`,
                            icon: 'success',
                            confirmButtonColor: '#7c3aed'
                        }).then(() => {
                            // Refresh after the user dismisses the success dialog
                            this.refreshApplications();
                        });
                    }
                } else {
                    Swal.fire('Error', data.error || 'Failed to deploy application', 'error');
                }
            } catch (error) {
                console.error('Error creating application:', error);
                Swal.fire('Error', 'Failed to deploy application', 'error');
            } finally {
                this.loading = false;
            }
        },

        // Helper function to validate application data
        isValidApplication(app) {
            const isValid = app && 
                   app.id && 
                   app.name && 
                   app.app_type && 
                   app.status && 
                   app.url &&
                   app.vps_name &&
                   app.created_at &&
                   app.name.trim() !== '' &&
                   app.url !== '' &&
                   app.vps_name.trim() !== '';
            
            if (!isValid) {
                console.log('Invalid application detected:', {
                    id: app?.id,
                    name: app?.name,
                    app_type: app?.app_type,
                    status: app?.status,
                    url: app?.url,
                    vps_name: app?.vps_name,
                    created_at: app?.created_at,
                    missing_fields: []
                        .concat(!app ? ['app is null/undefined'] : [])
                        .concat(!app?.id ? ['id'] : [])
                        .concat(!app?.name ? ['name'] : [])
                        .concat(!app?.app_type ? ['app_type'] : [])
                        .concat(!app?.status ? ['status'] : [])
                        .concat(!app?.url ? ['url'] : [])
                        .concat(!app?.vps_name ? ['vps_name'] : [])
                        .concat(!app?.created_at ? ['created_at'] : [])
                        .concat(app?.name && app.name.trim() === '' ? ['name is empty'] : [])
                        .concat(app?.url && app.url === '' ? ['url is empty'] : [])
                        .concat(app?.vps_name && app.vps_name.trim() === '' ? ['vps_name is empty'] : [])
                });
            }
            
            return isValid;
        },

        visitApplication(app) {
            window.open(app.url, '_blank');
        },

        async showUpgradeModal(app) {
            // Fetch available versions if it's a code-server app
            let versionsHtml = '';
            if (app.app_type === 'code-server') {
                try {
                    const versionsResponse = await fetch(`/applications/versions/code-server`);
                    if (versionsResponse.ok) {
                        const versionsData = await versionsResponse.json();
                        if (versionsData.success && versionsData.versions.length > 0) {
                            const options = versionsData.versions.map(v => 
                                `<option value="${v.version}" ${v.is_latest ? 'selected' : ''}>${v.version}${v.is_latest ? ' (Latest)' : ''}${!v.is_stable ? ' (Pre-release)' : ''}</option>`
                            ).join('');
                            versionsHtml = `
                                <select id="version-select" class="swal2-input m-0 w-full">
                                    <option value="latest">latest (Automatic)</option>
                                    ${options}
                                </select>
                                <div class="text-xs text-gray-500 mt-1">Or enter custom version below:</div>
                                <input id="custom-version" class="swal2-input m-0 w-full mt-1" placeholder="Custom version (optional)">
                            `;
                        }
                    }
                } catch (error) {
                    console.warn('Failed to fetch versions:', error);
                }
            }

            // Fallback to manual input if versions not available
            if (!versionsHtml) {
                versionsHtml = `<input id="new-version" class="swal2-input m-0 w-full" placeholder="Enter new version" value="latest">`;
            }

            const { value: newVersion } = await Swal.fire({
                title: 'Change Application Version',
                html: `
                    <div class="text-left">
                        <p class="mb-4">Change <strong>${app.name}</strong> to a different version:</p>
                        <div class="mb-4">
                            <label class="block text-sm font-medium text-gray-700 mb-1">Current Version:</label>
                            <div class="p-2 bg-gray-100 rounded text-sm">${app.app_version}</div>
                        </div>
                        <div>
                            <label class="block text-sm font-medium text-gray-700 mb-1">New Version:</label>
                            ${versionsHtml}
                        </div>
                    </div>
                `,
                showCancelButton: true,
                confirmButtonText: 'Change Version',
                confirmButtonColor: '#2563eb',
                preConfirm: () => {
                    // Check if we have version dropdown or manual input
                    const versionSelect = document.getElementById('version-select');
                    const customVersionInput = document.getElementById('custom-version');
                    const manualVersionInput = document.getElementById('new-version');
                    
                    let version;
                    if (versionSelect) {
                        const customVersion = customVersionInput?.value?.trim();
                        version = customVersion || versionSelect.value;
                    } else {
                        version = manualVersionInput?.value?.trim();
                    }
                    
                    if (!version) {
                        Swal.showValidationMessage('Version is required');
                        return false;
                    }
                    return version;
                }
            });

            if (newVersion) {
                await this.upgradeApplication(app.id, newVersion);
            }
        },

        async upgradeApplication(appId, version) {
            this.setLoadingState('Changing Version', 'Changing application version...');
            try {
                const response = await fetch(`/applications/${appId}/upgrade`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ version })
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    Swal.fire('Success!', 'Application version changed successfully', 'success');
                    await this.refreshApplications();
                } else {
                    Swal.fire('Error', data.error || 'Failed to change application version', 'error');
                }
            } catch (error) {
                console.error('Error upgrading application:', error);
                Swal.fire('Error', 'Failed to change application version', 'error');
            } finally {
                this.loading = false;
            }
        },

        async showPasswordChangeModal(app) {
            const appTypeName = app.app_type === 'argocd' ? 'ArgoCD' : 'Code-Server';
            const restartWarning = app.app_type === 'argocd' 
                ? 'This will restart your ArgoCD instance' 
                : 'This will restart your code-server instance';
            
            const { value: newPassword } = await Swal.fire({
                title: `Change ${appTypeName} Password`,
                html: `
                    <div class="text-left">
                        <p class="mb-4">Change the password for <strong>${app.name}</strong>:</p>
                        
                        <div class="mb-4 p-3 bg-yellow-50 border border-yellow-200 rounded-md">
                            <div class="flex items-center">
                                <svg class="w-5 h-5 text-yellow-600 mr-2" fill="currentColor" viewBox="0 0 20 20">
                                    <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"></path>
                                </svg>
                                <span class="text-sm text-yellow-800">${restartWarning}</span>
                            </div>
                        </div>
                        
                        <div>
                            <label class="block text-sm font-medium text-gray-700 mb-1">New Password:</label>
                            <input id="new-password" type="password" class="swal2-input m-0 w-full"
                                   placeholder="Enter new password (min 8 characters)" minlength="8">
                            <p class="text-xs text-gray-500 mt-1">Password must be at least 8 characters long</p>
                        </div>
                    </div>
                `,
                showCancelButton: true,
                confirmButtonText: 'Change Password',
                confirmButtonColor: '#059669',
                width: 600,
                preConfirm: () => {
                    const password = document.getElementById('new-password').value;
                    if (!password) {
                        Swal.showValidationMessage('Password is required');
                        return false;
                    }
                    if (password.length < 8) {
                        Swal.showValidationMessage('Password must be at least 8 characters long');
                        return false;
                    }
                    return password;
                }
            });

            if (newPassword) {
                await this.changePassword(app.id, app.name, newPassword);
            }
        },

        async showCurrentPasswordModal(app) {
            const appTypeName = app.app_type === 'argocd' ? 'ArgoCD' : 'Code-Server';
            const accessDescription = app.app_type === 'argocd' 
                ? 'You can use this password to access your ArgoCD admin interface'
                : 'You can use this password to access your code-server instance';
            const openButtonText = app.app_type === 'argocd' ? 'Open ArgoCD' : 'Open Code-Server';
            
            this.setLoadingState('Retrieving Password', `Getting current password for "${app.name}"...`);
            try {
                const response = await fetch(`/applications/${app.id}/password`, {
                    method: 'GET',
                    headers: {
                        'Content-Type': 'application/json',
                    }
                });
                
                const data = await response.json();
                
                if (response.ok && data.password) {
                    Swal.fire({
                        title: 'Current Password',
                        html: `
                            <div class="text-left">
                                <p class="mb-4">Current password for <strong>${app.name}</strong>:</p>
                                
                                <div class="mb-4 p-4 bg-blue-50 border border-blue-200 rounded-lg">
                                    <h4 class="font-medium text-blue-900 mb-2">🔐 Password</h4>
                                    <div class="flex items-center space-x-2">
                                        <input type="text" value="${data.password}" 
                                               class="flex-1 px-3 py-2 border border-blue-300 rounded font-mono text-sm bg-white" 
                                               readonly id="current-password-field">
                                        <button onclick="navigator.clipboard.writeText('${data.password}'); this.textContent='Copied!'; setTimeout(() => this.textContent='Copy', 1000)" 
                                                class="px-3 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 text-sm">
                                            Copy
                                        </button>
                                    </div>
                                    <p class="text-sm text-blue-700 mt-2">💡 ${accessDescription}</p>
                                </div>
                                
                                <div class="mb-4 p-4 bg-green-50 border border-green-200 rounded-lg">
                                    <h4 class="font-medium text-green-900 mb-2">🔗 Quick Access</h4>
                                    <div class="text-center">
                                        <a href="${app.url}" target="_blank" 
                                           class="inline-flex items-center px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700">
                                            <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"></path>
                                            </svg>
                                            ${openButtonText}
                                        </a>
                                    </div>
                                </div>
                            </div>
                        `,
                        icon: 'info',
                        confirmButtonColor: '#7c3aed',
                        confirmButtonText: 'Got it!',
                        width: 700,
                        allowOutsideClick: true
                    });
                } else {
                    Swal.fire('Error', data.error || 'Failed to retrieve current password', 'error');
                }
            } catch (error) {
                console.error('Error retrieving current password:', error);
                Swal.fire('Error', 'Failed to retrieve current password', 'error');
            } finally {
                this.loading = false;
            }
        },

        async showTokenModal(app) {
            console.log('showTokenModal called for app:', app);
            this.setLoadingState('Retrieving Token', `Getting authentication token for "${app.name}"...`);
            try {
                const response = await fetch(`/applications/${app.id}/token`, {
                    method: 'GET',
                    headers: {
                        'Content-Type': 'application/json',
                    }
                });
                
                const data = await response.json();
                
                if (response.ok && data.token) {
                    Swal.fire({
                        title: 'Authentication Token',
                        html: `
                            <div class="text-left">
                                <p class="mb-4">Authentication token for <strong>${app.name}</strong>:</p>
                                
                                <div class="mb-4 p-4 bg-purple-50 border border-purple-200 rounded-lg">
                                    <h4 class="font-medium text-purple-900 mb-2">🔐 Kubernetes Authentication Token</h4>
                                    <div class="flex items-center space-x-2">
                                        <textarea rows="4" class="flex-1 px-3 py-2 border border-purple-300 rounded font-mono text-xs bg-white resize-none" 
                                                  readonly onclick="this.select()">${data.token}</textarea>
                                        <button onclick="navigator.clipboard.writeText('${data.token}').then(() => {
                                            const btn = this;
                                            const originalText = btn.innerHTML;
                                            btn.innerHTML = '✓';
                                            btn.style.color = 'green';
                                            setTimeout(() => {
                                                btn.innerHTML = originalText;
                                                btn.style.color = '';
                                            }, 2000);
                                        })" 
                                                class="px-3 py-2 bg-purple-600 text-white rounded hover:bg-purple-700 text-sm">
                                            📋
                                        </button>
                                    </div>
                                    <p class="text-xs text-purple-700 mt-2">Click to select all, or use the copy button</p>
                                </div>
                                
                                <div class="mb-4 p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
                                    <h4 class="font-medium text-yellow-800 mb-2">ℹ️ How to use this token:</h4>
                                    <ol class="text-sm text-yellow-700 space-y-1">
                                        <li>1. Copy the authentication token above</li>
                                        <li>2. Visit your Headlamp dashboard</li>
                                        <li>3. Paste the token in the authentication field</li>
                                        <li>4. Click "Authenticate" to access your Kubernetes cluster</li>
                                    </ol>
                                </div>
                                
                                <div class="flex justify-center">
                                    <div class="bg-gray-50 border border-gray-200 rounded-lg p-3">
                                        <a href="${app.url}" target="_blank" 
                                           class="inline-flex items-center px-4 py-2 bg-purple-600 text-white rounded-md hover:bg-purple-700 text-sm font-medium">
                                            <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"></path>
                                            </svg>
                                            Open Headlamp
                                        </a>
                                    </div>
                                </div>
                            </div>
                        `,
                        icon: 'info',
                        confirmButtonColor: '#7c3aed',
                        confirmButtonText: 'Got it!',
                        width: 700,
                        allowOutsideClick: true
                    });
                } else {
                    Swal.fire('Error', data.error || 'Failed to retrieve authentication token', 'error');
                }
            } catch (error) {
                console.error('Error retrieving token:', error);
                Swal.fire('Error', 'Failed to retrieve authentication token', 'error');
            } finally {
                this.loading = false;
            }
        },

        async changePassword(appId, appName, newPassword) {
            this.setLoadingState('Changing Password', `Updating password for "${appName}"...`);
            try {
                const response = await fetch(`/applications/${appId}/password`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ new_password: newPassword })
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    Swal.fire({
                        title: 'Password Changed!',
                        html: `
                            <div class="text-left">
                                <p class="mb-4">Password for <strong>${appName}</strong> has been updated successfully!</p>
                                <div class="p-3 bg-green-50 border border-green-200 rounded-md">
                                    <div class="flex items-center">
                                        <svg class="w-5 h-5 text-green-600 mr-2" fill="currentColor" viewBox="0 0 20 20">
                                            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"></path>
                                        </svg>
                                        <span class="text-sm text-green-800">Your code-server instance is restarting with the new password</span>
                                    </div>
                                </div>
                            </div>
                        `,
                        icon: 'success',
                        confirmButtonColor: '#059669'
                    });
                } else {
                    Swal.fire('Error', data.error || 'Failed to change password', 'error');
                }
            } catch (error) {
                console.error('Error changing password:', error);
                Swal.fire('Error', 'Failed to change password', 'error');
            } finally {
                this.loading = false;
            }
        },

        async confirmDeleteApplication(appId, appName) {
            const result = await Swal.fire({
                title: 'Delete Application?',
                text: `Are you sure you want to delete "${appName}"? This action cannot be undone.`,
                icon: 'warning',
                showCancelButton: true,
                confirmButtonColor: '#dc2626',
                cancelButtonColor: '#6b7280',
                confirmButtonText: 'Yes, delete it!'
            });

            if (result.isConfirmed) {
                await this.deleteApplication(appId, appName);
            }
        },

        async deleteApplication(appId, appName) {
            this.setLoadingState('Deleting Application', `Deleting "${appName}"...`);
            try {
                const response = await fetch(`/applications/${appId}`, {
                    method: 'DELETE'
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    Swal.fire('Deleted!', `Application "${appName}" has been deleted.`, 'success');
                    await this.refreshApplications();
                } else {
                    Swal.fire('Error', data.error || 'Failed to delete application', 'error');
                }
            } catch (error) {
                console.error('Error deleting application:', error);
                Swal.fire('Error', 'Failed to delete application', 'error');
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

        setLoadingState(title, message) {
            this.loadingTitle = title;
            this.loadingMessage = message;
            this.loading = true;
        },

        getPredefinedAppIcon(appType) {
            const app = this.predefinedApps.find(a => a.id === appType);
            return app ? app.icon : '📦';
        },

        getPredefinedAppName(appType) {
            const app = this.predefinedApps.find(a => a.id === appType);
            return app ? app.name : appType;
        },

        // Port Forwarding Functions
        async showPortForwardingModal(app) {
            this.portForwardingModal.app = app;
            this.portForwardingModal.domain = this.extractDomain(app.url);
            this.portForwardingModal.ports = [];
            this.portForwardingModal.newPort = { port: '', subdomain: '' };
            this.portForwardingModal.show = true;
            
            // Show global loading overlay while fetching port forwards
            this.setLoadingState('Loading Port Forwards', 'Retrieving existing port forwards...');
            
            // Load existing port forwards
            await this.loadPortForwards(app.id);
            
            // Hide global loading overlay
            this.loading = false;
        },

        async loadPortForwards(appId) {
            try {
                const response = await fetch(`/applications/${appId}/port-forwards`);
                if (response.ok) {
                    const data = await response.json();
                    this.portForwardingModal.ports = data.port_forwards || [];
                }
            } catch (error) {
                console.error('Error loading port forwards:', error);
            }
        },

        async addPortForward() {
            const { port, subdomain } = this.portForwardingModal.newPort;
            const appId = this.portForwardingModal.app.id;
            
            if (!port || !subdomain) {
                Swal.fire('Error', 'Please fill in both port and subdomain', 'error');
                return;
            }

            // Show global loading overlay
            this.setLoadingState('Adding Port Forward', 'Creating service and ingress...');
            
            try {
                const response = await fetch(`/applications/${appId}/port-forwards`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        port: parseInt(port),
                        subdomain: subdomain.trim()
                    })
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    Swal.fire('Success!', `Port forward created: https://${subdomain}.${this.portForwardingModal.domain}`, 'success');
                    this.portForwardingModal.newPort = { port: '', subdomain: '' };
                    
                    // Refresh port forwards list with loading overlay
                    this.setLoadingState('Refreshing Port Forwards', 'Updating port forwards list...');
                    await this.loadPortForwards(appId);
                } else {
                    Swal.fire('Error', data.error || 'Failed to create port forward', 'error');
                }
            } catch (error) {
                console.error('Error adding port forward:', error);
                Swal.fire('Error', 'Failed to create port forward', 'error');
            } finally {
                this.loading = false;
            }
        },

        async removePortForward(portForwardId) {
            const result = await Swal.fire({
                title: 'Remove Port Forward?',
                text: 'This will delete the service and ingress for this port forward.',
                icon: 'warning',
                showCancelButton: true,
                confirmButtonColor: '#dc2626',
                cancelButtonColor: '#6b7280',
                confirmButtonText: 'Yes, remove it!'
            });

            if (result.isConfirmed) {
                const appId = this.portForwardingModal.app.id;
                
                // Show global loading overlay
                this.setLoadingState('Removing Port Forward', 'Deleting service and ingress...');
                
                try {
                    const response = await fetch(`/applications/${appId}/port-forwards/${portForwardId}`, {
                        method: 'DELETE'
                    });
                    
                    const data = await response.json();
                    
                    if (response.ok) {
                        Swal.fire('Removed!', 'Port forward has been removed.', 'success');
                        
                        // Refresh port forwards list with loading overlay
                        this.setLoadingState('Refreshing Port Forwards', 'Updating port forwards list...');
                        await this.loadPortForwards(appId);
                    } else {
                        Swal.fire('Error', data.error || 'Failed to remove port forward', 'error');
                    }
                } catch (error) {
                    console.error('Error removing port forward:', error);
                    Swal.fire('Error', 'Failed to remove port forward', 'error');
                } finally {
                    this.loading = false;
                }
            }
        },

        extractDomain(url) {
            try {
                const urlObj = new URL(url);
                const parts = urlObj.hostname.split('.');
                // Remove the first part (subdomain) and return the domain
                return parts.slice(1).join('.');
            } catch (error) {
                console.error('Error extracting domain from URL:', url, error);
                return '';
            }
        }
    }
}