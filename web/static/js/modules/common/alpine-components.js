// Alpine.js Common Components - Shared component utilities
export class AlpineComponents {
    // Common loading state management
    static createLoadingState() {
        return {
            loading: false,
            loadingTitle: 'Processing...',
            loadingMessage: 'Please wait while the operation completes.',
            
            setLoadingState(title, message) {
                this.loadingTitle = title;
                this.loadingMessage = message;
                this.loading = true;
            },

            clearLoadingState() {
                this.loading = false;
            }
        };
    }

    // Common auto-refresh functionality
    static createAutoRefresh(refreshFunction, interval = 30000) {
        return {
            autoRefreshEnabled: true,
            refreshInterval: interval,
            intervalId: null,
            countdownInterval: null,
            countdown: 0,
            isRefreshing: false,

            startAutoRefresh() {
                if (this.autoRefreshEnabled && !this.intervalId) {
                    this.startCountdown();
                    
                    this.intervalId = setInterval(() => {
                        if (!document.hidden && !this.isRefreshing) {
                            refreshFunction.call(this);
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
                if (this.countdownInterval) {
                    clearInterval(this.countdownInterval);
                }
                
                this.countdown = Math.floor(this.refreshInterval / 1000);
                this.countdownInterval = setInterval(() => {
                    this.countdown--;
                    if (this.countdown <= 0) {
                        clearInterval(this.countdownInterval);
                        this.countdownInterval = null;
                    }
                }, 1000);
            },

            setupVisibilityHandling() {
                document.addEventListener('visibilitychange', () => {
                    if (document.hidden) {
                        this.stopAutoRefresh();
                    } else {
                        this.startAutoRefresh();
                        // Refresh immediately when page becomes visible
                        if (typeof this.refreshQuietly === 'function') {
                            this.refreshQuietly();
                        }
                    }
                });
            }
        };
    }

    // Common form validation utilities
    static createFormValidation() {
        return {
            errors: {},
            
            validateRequired(field, value, fieldName) {
                if (!value || (typeof value === 'string' && value.trim() === '')) {
                    this.errors[field] = `${fieldName} is required`;
                    return false;
                } else {
                    delete this.errors[field];
                    return true;
                }
            },

            validateEmail(field, value) {
                const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
                if (value && !emailRegex.test(value)) {
                    this.errors[field] = 'Please enter a valid email address';
                    return false;
                } else {
                    delete this.errors[field];
                    return true;
                }
            },

            validateMinLength(field, value, minLength, fieldName) {
                if (value && value.length < minLength) {
                    this.errors[field] = `${fieldName} must be at least ${minLength} characters long`;
                    return false;
                } else {
                    delete this.errors[field];
                    return true;
                }
            },

            hasErrors() {
                return Object.keys(this.errors).length > 0;
            },

            getError(field) {
                return this.errors[field] || '';
            },

            clearErrors() {
                this.errors = {};
            }
        };
    }

    // Common pagination utilities
    static createPagination(itemsPerPage = 10) {
        return {
            currentPage: 1,
            itemsPerPage,

            get totalPages() {
                return Math.ceil(this.totalItems / this.itemsPerPage);
            },

            get startIndex() {
                return (this.currentPage - 1) * this.itemsPerPage;
            },

            get endIndex() {
                return Math.min(this.startIndex + this.itemsPerPage, this.totalItems);
            },

            get paginatedItems() {
                return this.items.slice(this.startIndex, this.endIndex);
            },

            goToPage(page) {
                if (page >= 1 && page <= this.totalPages) {
                    this.currentPage = page;
                }
            },

            nextPage() {
                if (this.currentPage < this.totalPages) {
                    this.currentPage++;
                }
            },

            previousPage() {
                if (this.currentPage > 1) {
                    this.currentPage--;
                }
            },

            getPageNumbers() {
                const pages = [];
                const maxVisible = 5;
                const start = Math.max(1, this.currentPage - Math.floor(maxVisible / 2));
                const end = Math.min(this.totalPages, start + maxVisible - 1);

                for (let i = start; i <= end; i++) {
                    pages.push(i);
                }

                return pages;
            }
        };
    }

    // Common search/filter utilities
    static createSearch() {
        return {
            searchTerm: '',
            searchFields: [],

            get filteredItems() {
                if (!this.searchTerm.trim()) {
                    return this.items;
                }

                const term = this.searchTerm.toLowerCase();
                return this.items.filter(item => {
                    return this.searchFields.some(field => {
                        const value = this.getNestedValue(item, field);
                        return value && value.toString().toLowerCase().includes(term);
                    });
                });
            },

            getNestedValue(obj, path) {
                return path.split('.').reduce((current, key) => {
                    return current && current[key] !== undefined ? current[key] : null;
                }, obj);
            },

            clearSearch() {
                this.searchTerm = '';
            }
        };
    }

    // Debounced function utility
    static debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func.apply(this, args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    }
}

export default AlpineComponents;