// VPS Creation Wizard Module - Alpine.js component
export function vpsCreationWizard() {
    return {
        currentStep: 1,
        loading: false,
        loadingMessage: 'Loading...',
        validatingKey: false,
        creating: false,
        
        // Step 1: Provider Selection
        selectedProvider: '',
        
        // Step 2: Hetzner Key (Hetzner only)
        hetznerKey: '',
        existingKey: '',
        
        // OCI Setup data
        sshPublicKey: '',
        ociInstanceName: '',
        ociPublicIP: '',
        ociUsername: 'ubuntu',
        ociShape: 'VM.Standard.A1.Flex',
        
        // Hetzner: Location and Server Type
        locations: [],
        selectedLocation: null,
        serverTypes: [],
        filteredServerTypes: [],
        selectedServerType: null,
        architectureFilter: '',
        sortBy: 'price_asc',
        
        // Review Step
        serverName: '',
        nameValidationState: '', // '', 'checking', 'valid', 'invalid'
        nameValidationMessage: '',
        nameValidationTimeout: null,
        
        async init() {
            this.serverName = `xanthus-k3s-${Date.now()}`;
            // Generate SSH key for OCI setup
            await this.loadSSHKey();
        },
        selectProvider(provider) {
            this.selectedProvider = provider;
        },
        
        async loadSSHKey() {
            try {
                const response = await fetch('/vps/oci-ssh-key');
                if (response.ok) {
                    const data = await response.json();
                    this.sshPublicKey = data.public_key || 'Loading...';
                } else {
                    this.sshPublicKey = 'Error loading SSH key';
                }
            } catch (error) {
                console.error('Error loading SSH key:', error);
                this.sshPublicKey = 'Error loading SSH key';
            }
        },
        
        async copySSHKey() {
            try {
                await navigator.clipboard.writeText(this.sshPublicKey);
                Swal.fire({
                    title: 'Copied!',
                    text: 'SSH public key copied to clipboard',
                    icon: 'success',
                    timer: 2000,
                    showConfirmButton: false
                });
            } catch (error) {
                console.error('Failed to copy SSH key:', error);
                Swal.fire('Error', 'Failed to copy SSH key to clipboard', 'error');
            }
        },
        
        async checkExistingKey() {
            this.loading = true;
            this.loadingMessage = 'Checking for existing Hetzner API key...';
            
            try {
                const response = await fetch('/vps/check-key');
                if (response.ok) {
                    const data = await response.json();
                    if (data.exists) {
                        this.existingKey = data.masked_key;
                    }
                }
            } catch (error) {
                console.error('Error checking existing key:', error);
            } finally {
                this.loading = false;
            }
        },
        
        showNewKeyInput() {
            this.existingKey = '';
        },
        
        async useExistingKey() {
            this.currentStep = 3;
            await this.loadLocations();
        },
        
        async validateHetznerKey() {
            if (!this.hetznerKey) return;
            
            this.validatingKey = true;
            try {
                const response = await fetch('/vps/validate-key', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    body: `key=${encodeURIComponent(this.hetznerKey)}`
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    this.currentStep = 3;
                    await this.loadLocations();
                } else {
                    Swal.fire('Invalid Key', data.error || 'The Hetzner API key is invalid', 'error');
                }
            } catch (error) {
                console.error('Error validating key:', error);
                Swal.fire('Error', 'Failed to validate API key', 'error');
            } finally {
                this.validatingKey = false;
            }
        },

        // OCI Token Generator and Validation
        ociCredentials: {
            tenancy: '',
            user: '',
            region: 'us-phoenix-1',
            fingerprint: '',
            privateKey: ''
        },
        ociToken: '',
        showTokenGenerator: false,
        validatingOCIToken: false,

        showOCITokenGenerator() {
            this.showTokenGenerator = true;
        },

        hideOCITokenGenerator() {
            this.showTokenGenerator = false;
        },

        generateOCIToken() {
            if (!this.ociCredentials.tenancy || !this.ociCredentials.user || 
                !this.ociCredentials.region || !this.ociCredentials.fingerprint || 
                !this.ociCredentials.privateKey) {
                Swal.fire('Missing Information', 'Please fill in all OCI credential fields', 'warning');
                return;
            }

            try {
                const tokenData = {
                    tenancy: this.ociCredentials.tenancy,
                    user: this.ociCredentials.user,
                    region: this.ociCredentials.region,
                    fingerprint: this.ociCredentials.fingerprint,
                    private_key: this.ociCredentials.privateKey
                };

                // Base64 encode the JSON
                this.ociToken = btoa(JSON.stringify(tokenData));
                
                Swal.fire({
                    title: 'Token Generated!',
                    text: 'Your OCI auth token has been generated successfully.',
                    icon: 'success',
                    timer: 2000,
                    showConfirmButton: false
                });
            } catch (error) {
                console.error('Error generating OCI token:', error);
                Swal.fire('Error', 'Failed to generate OCI token', 'error');
            }
        },

        async validateOCIToken() {
            if (!this.ociToken) {
                Swal.fire('Missing Token', 'Please provide an OCI auth token', 'warning');
                return;
            }
            
            this.validatingOCIToken = true;
            try {
                const response = await fetch('/vps/oci/validate-token', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ token: this.ociToken })
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    // Store the token and proceed
                    await this.storeOCIToken();
                    this.currentStep = 3; // Go to OCI instance creation step
                } else {
                    Swal.fire('Invalid Token', data.error || 'The OCI auth token is invalid', 'error');
                }
            } catch (error) {
                console.error('Error validating OCI token:', error);
                Swal.fire('Error', 'Failed to validate OCI token', 'error');
            } finally {
                this.validatingOCIToken = false;
            }
        },

        async storeOCIToken() {
            try {
                const response = await fetch('/vps/oci/store-token', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ oci_token: this.ociToken })
                });
                
                if (!response.ok) {
                    const data = await response.json();
                    throw new Error(data.error || 'Failed to store OCI token');
                }
            } catch (error) {
                console.error('Error storing OCI token:', error);
                Swal.fire('Warning', 'Token validated but storage failed. You may need to re-enter it.', 'warning');
            }
        },

        copyOCIToken() {
            if (!this.ociToken) return;
            
            navigator.clipboard.writeText(this.ociToken).then(() => {
                Swal.fire({
                    title: 'Copied!',
                    text: 'OCI auth token copied to clipboard',
                    icon: 'success',
                    timer: 2000,
                    showConfirmButton: false
                });
            }).catch(() => {
                Swal.fire('Error', 'Failed to copy token to clipboard', 'error');
            });
        },
        
        async loadLocations() {
            this.loading = true;
            this.loadingMessage = 'Loading available locations...';
            
            try {
                const response = await fetch('/vps/locations');
                const data = await response.json();
                
                if (response.ok) {
                    this.locations = data.locations || [];
                } else {
                    throw new Error(data.error || 'Failed to load locations');
                }
            } catch (error) {
                console.error('Error loading locations:', error);
                Swal.fire('Error', 'Failed to load locations', 'error');
            } finally {
                this.loading = false;
            }
        },
        
        selectLocation(location) {
            this.selectedLocation = location;
        },
        
        async loadServerTypes() {
            this.loading = true;
            this.loadingMessage = 'Loading server types and checking availability...';
            
            try {
                const response = await fetch(`/vps/server-types?location=${this.selectedLocation.name}`);
                const data = await response.json();
                
                if (response.ok) {
                    // Process server types to add computed fields
                    this.serverTypes = (data.serverTypes || []).map(type => {
                        // Use Hetzner's actual monthly pricing (not calculated from hourly)
                        const priceData = type.prices.find(p => p.location !== 'monthly_calc');
                        const monthlyPriceGross = priceData?.price_monthly?.gross || '0';
                        const monthlyPriceNet = priceData?.price_monthly?.net || '0';
                        
                        // Calculate VAT percentage
                        const vatPercentage = parseFloat(monthlyPriceNet) > 0 ? 
                            (((parseFloat(monthlyPriceGross) - parseFloat(monthlyPriceNet)) / parseFloat(monthlyPriceNet)) * 100).toFixed(0) : '0';
                        
                        // Check availability for selected location
                        const available = type.available_locations && type.available_locations[this.selectedLocation.name] !== false;
                        
                        return {
                            ...type,
                            monthlyPrice: parseFloat(monthlyPriceGross).toFixed(2),
                            monthlyPriceNet: parseFloat(monthlyPriceNet).toFixed(2),
                            vatPercentage: vatPercentage,
                            available: available
                        };
                    });
                    this.applyFilters();
                } else {
                    throw new Error(data.error || 'Failed to load server types');
                }
            } catch (error) {
                console.error('Error loading server types:', error);
                Swal.fire('Error', 'Failed to load server types', 'error');
            } finally {
                this.loading = false;
            }
        },
        
        selectServerType(serverType) {
            if (serverType.available) {
                this.selectedServerType = serverType;
            }
        },
        
        applyFilters() {
            let filtered = [...this.serverTypes];
            
            // Filter by architecture
            if (this.architectureFilter) {
                filtered = filtered.filter(type => type.architecture === this.architectureFilter);
            }
            
            this.filteredServerTypes = filtered;
            this.applySort();
        },
        
        applySort() {
            this.filteredServerTypes.sort((a, b) => {
                switch (this.sortBy) {
                    case 'price_asc':
                        return parseFloat(a.monthlyPrice) - parseFloat(b.monthlyPrice);
                    case 'price_desc':
                        return parseFloat(b.monthlyPrice) - parseFloat(a.monthlyPrice);
                    case 'cpu_asc':
                        return a.cores - b.cores;
                    case 'cpu_desc':
                        return b.cores - a.cores;
                    case 'memory_asc':
                        return a.memory - b.memory;
                    case 'memory_desc':
                        return b.memory - a.memory;
                    default:
                        return 0;
                }
            });
        },
        
        async nextStep() {
            if (this.currentStep === 1 && this.selectedProvider) {
                this.currentStep = 2;
                if (this.selectedProvider === 'hetzner') {
                    await this.checkExistingKey();
                }
            } else if (this.currentStep === 2) {
                if (this.selectedProvider === 'hetzner' && this.selectedLocation) {
                    this.currentStep = 4; // Skip to server type selection
                    await this.loadServerTypes();
                } else if (this.selectedProvider === 'oci' && this.ociInstanceName && this.ociPublicIP && this.ociUsername) {
                    this.currentStep = 3; // Go to OCI review
                }
            } else if (this.currentStep === 3) {
                if (this.selectedProvider === 'hetzner' && this.selectedLocation) {
                    this.currentStep = 4;
                    await this.loadServerTypes();
                }
            } else if (this.currentStep === 4 && this.selectedProvider === 'hetzner' && this.selectedServerType) {
                this.currentStep = 5; // Hetzner review
                await this.validateName();
            }
        },
        
        previousStep() {
            if (this.currentStep > 1) {
                this.currentStep--;
            }
        },
        
        debounceValidateName() {
            if (this.nameValidationTimeout) {
                clearTimeout(this.nameValidationTimeout);
            }
            
            this.nameValidationTimeout = setTimeout(() => {
                this.validateName();
            }, 500); // Wait 500ms after user stops typing
        },
        
        async validateName() {
            if (!this.serverName || this.serverName.trim().length === 0) {
                this.nameValidationState = '';
                this.nameValidationMessage = '';
                return;
            }
            
            // Basic validation
            if (this.serverName.length < 3) {
                this.nameValidationState = 'invalid';
                this.nameValidationMessage = 'Name must be at least 3 characters long';
                return;
            }
            
            if (this.serverName.length > 63) {
                this.nameValidationState = 'invalid';
                this.nameValidationMessage = 'Name must be less than 64 characters';
                return;
            }
            
            // Check for valid characters (alphanumeric and hyphens)
            if (!/^[a-zA-Z0-9-]+$/.test(this.serverName)) {
                this.nameValidationState = 'invalid';
                this.nameValidationMessage = 'Name can only contain letters, numbers, and hyphens';
                return;
            }
            
            // Cannot start or end with hyphen
            if (this.serverName.startsWith('-') || this.serverName.endsWith('-')) {
                this.nameValidationState = 'invalid';
                this.nameValidationMessage = 'Name cannot start or end with a hyphen';
                return;
            }
            
            this.nameValidationState = 'checking';
            this.nameValidationMessage = 'Checking name availability...';
            
            try {
                const response = await fetch('/vps/validate-name', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    body: `name=${encodeURIComponent(this.serverName)}`
                });
                
                const data = await response.json();
                
                if (response.ok && data.available) {
                    this.nameValidationState = 'valid';
                    this.nameValidationMessage = 'Name is available';
                } else {
                    this.nameValidationState = 'invalid';
                    this.nameValidationMessage = data.error || 'Name is not available';
                }
            } catch (error) {
                console.error('Error validating name:', error);
                this.nameValidationState = 'invalid';
                this.nameValidationMessage = 'Unable to check name availability';
            }
        },
        
        async createVPS() {
            if (!this.serverName || !this.selectedLocation || !this.selectedServerType || this.nameValidationState !== 'valid') return;
            
            this.creating = true;
            this.loading = true;
            this.loadingMessage = `Creating VPS "${this.serverName}"...`;
            
            try {
                const response = await fetch('/vps/create', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    body: `name=${encodeURIComponent(this.serverName)}&location=${encodeURIComponent(this.selectedLocation.name)}&server_type=${encodeURIComponent(this.selectedServerType.name)}`
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    this.loadingMessage = 'VPS created successfully! Redirecting...';
                    
                    // Brief pause to show success message
                    await new Promise(resolve => setTimeout(resolve, 1500));
                    
                    await Swal.fire({
                        title: 'VPS Created Successfully!',
                        html: `
                            <div class="text-left">
                                <p class="mb-2">Your VPS "${this.serverName}" is being created and configured.</p>
                                <p class="mb-2">This process may take 5-10 minutes to complete.</p>
                                <p class="text-sm text-gray-600">You will be redirected to the VPS management page.</p>
                            </div>
                        `,
                        icon: 'success',
                        confirmButtonText: 'Go to VPS Management'
                    });
                    
                    window.location.href = '/vps';
                } else {
                    this.loading = false;
                    Swal.fire('Error', data.error || 'Failed to create VPS', 'error');
                }
            } catch (error) {
                console.error('Error creating VPS:', error);
                this.loading = false;
                Swal.fire('Error', 'Failed to create VPS', 'error');
            } finally {
                this.creating = false;
            }
        },

        async createOCIInstance() {
            if (!this.serverName || !this.ociCredentials.region) return;
            
            this.creating = true;
            this.loading = true;
            this.loadingMessage = `Creating OCI instance "${this.serverName}"...`;
            
            try {
                const response = await fetch('/vps/oci/create', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        name: this.serverName,
                        shape: 'VM.Standard.E2.1.Micro', // Always Free tier
                        region: this.ociCredentials.region,
                        timezone: 'UTC'
                    })
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    this.loadingMessage = 'OCI instance created successfully! Redirecting...';
                    
                    // Brief pause to show success message
                    await new Promise(resolve => setTimeout(resolve, 1500));
                    
                    await Swal.fire({
                        title: 'OCI Instance Created Successfully!',
                        html: `
                            <div class="text-left">
                                <p class="mb-2">Your OCI instance "${this.serverName}" has been created and configured.</p>
                                <p class="mb-2">K3s and Helm are being installed automatically.</p>
                                <p class="mb-2"><strong>Instance ID:</strong> ${data.server.oci_id}</p>
                                <p class="mb-2"><strong>IP Address:</strong> ${data.server.public_net.ipv4.ip}</p>
                                <p class="text-sm text-gray-600">You will be redirected to the VPS management page.</p>
                            </div>
                        `,
                        icon: 'success',
                        confirmButtonText: 'Go to VPS Management'
                    });
                    
                    window.location.href = '/vps';
                } else {
                    this.loading = false;
                    Swal.fire('Error', data.error || 'Failed to create OCI instance', 'error');
                }
            } catch (error) {
                console.error('Error creating OCI instance:', error);
                this.loading = false;
                Swal.fire('Error', 'Failed to create OCI instance', 'error');
            } finally {
                this.creating = false;
            }
        },
        
        async addOCIInstance() {
            if (!this.ociInstanceName || !this.ociPublicIP || !this.ociUsername) return;
            
            this.creating = true;
            this.loading = true;
            this.loadingMessage = `Adding OCI instance "${this.ociInstanceName}"...`;
            
            try {
                const response = await fetch('/vps/add-oci', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        name: this.ociInstanceName,
                        public_ip: this.ociPublicIP,
                        username: this.ociUsername,
                        shape: this.ociShape,
                        provider: 'oci'
                    })
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    this.loadingMessage = 'OCI instance added successfully! Setting up K3s...';
                    
                    // Brief pause to show success message
                    await new Promise(resolve => setTimeout(resolve, 1500));
                    
                    await Swal.fire({
                        title: 'OCI Instance Added Successfully!',
                        html: `
                            <div class="text-left">
                                <p class="mb-2">Your OCI instance "${this.ociInstanceName}" has been added to Xanthus.</p>
                                <p class="mb-2">K3s setup is running in the background and may take 5-10 minutes.</p>
                                <p class="text-sm text-gray-600">You will be redirected to the VPS management page.</p>
                            </div>
                        `,
                        icon: 'success',
                        confirmButtonText: 'Go to VPS Management'
                    });
                    
                    window.location.href = '/vps';
                } else {
                    this.loading = false;
                    Swal.fire('Error', data.error || 'Failed to add OCI instance', 'error');
                }
            } catch (error) {
                console.error('Error adding OCI instance:', error);
                this.loading = false;
                Swal.fire('Error', 'Failed to add OCI instance', 'error');
            } finally {
                this.creating = false;
            }
        }
    }
}