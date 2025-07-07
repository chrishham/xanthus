// API Client Utility - Common HTTP request helpers
export class ApiClient {
    static async get(url, options = {}) {
        return this.request(url, {
            method: 'GET',
            ...options
        });
    }

    static async post(url, data, options = {}) {
        const isFormData = data instanceof FormData;
        const body = isFormData ? data : JSON.stringify(data);
        const headers = isFormData ? {} : { 'Content-Type': 'application/json' };

        return this.request(url, {
            method: 'POST',
            body,
            headers: {
                ...headers,
                ...options.headers
            },
            ...options
        });
    }

    static async postForm(url, formData, options = {}) {
        const body = new URLSearchParams(formData).toString();
        
        return this.request(url, {
            method: 'POST',
            body,
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
                ...options.headers
            },
            ...options
        });
    }

    static async delete(url, options = {}) {
        return this.request(url, {
            method: 'DELETE',
            ...options
        });
    }

    static async request(url, options = {}) {
        try {
            const response = await fetch(url, {
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                ...options
            });

            // Handle authentication errors
            if (response.status === 401 || response.status === 403) {
                console.log('Authentication failed, redirecting to login');
                window.location.href = '/login';
                return;
            }

            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.error || `HTTP error! status: ${response.status}`);
            }

            return data;
        } catch (error) {
            console.error('API request failed:', error);
            throw error;
        }
    }

    // Helper method for handling common error responses
    static handleError(error, defaultMessage = 'An error occurred') {
        if (error.name === 'TypeError' && error.message.includes('NetworkError')) {
            return 'Network error. Please check your connection.';
        }
        
        return error.message || defaultMessage;
    }
}

// Export for backward compatibility
export default ApiClient;