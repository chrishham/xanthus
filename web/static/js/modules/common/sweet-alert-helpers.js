// SweetAlert2 Helper Functions - Common dialog configurations
export class SweetAlertHelpers {
    // Standard success dialog
    static success(title, text, confirmButtonText = 'OK') {
        return Swal.fire({
            title,
            text,
            icon: 'success',
            confirmButtonText,
            confirmButtonColor: '#059669'
        });
    }

    // Standard error dialog
    static error(title, text = 'An error occurred', confirmButtonText = 'OK') {
        return Swal.fire({
            title,
            text,
            icon: 'error',
            confirmButtonText,
            confirmButtonColor: '#dc2626'
        });
    }

    // Standard warning dialog
    static warning(title, text, confirmButtonText = 'OK') {
        return Swal.fire({
            title,
            text,
            icon: 'warning',
            confirmButtonText,
            confirmButtonColor: '#d97706'
        });
    }

    // Standard info dialog
    static info(title, text, confirmButtonText = 'OK') {
        return Swal.fire({
            title,
            text,
            icon: 'info',
            confirmButtonText,
            confirmButtonColor: '#2563eb'
        });
    }

    // Confirmation dialog
    static confirm(title, text, confirmButtonText = 'Yes', cancelButtonText = 'Cancel') {
        return Swal.fire({
            title,
            text,
            icon: 'question',
            showCancelButton: true,
            confirmButtonColor: '#2563eb',
            cancelButtonColor: '#6b7280',
            confirmButtonText,
            cancelButtonText
        });
    }

    // Destructive confirmation dialog (for delete operations)
    static confirmDelete(title, text, confirmButtonText = 'Yes, delete it!', cancelButtonText = 'Cancel') {
        return Swal.fire({
            title,
            text,
            icon: 'warning',
            showCancelButton: true,
            confirmButtonColor: '#dc2626',
            cancelButtonColor: '#6b7280',
            confirmButtonText,
            cancelButtonText
        });
    }

    // Loading dialog
    static loading(title, text) {
        return Swal.fire({
            title,
            text,
            allowOutsideClick: false,
            allowEscapeKey: false,
            showConfirmButton: false,
            willOpen: () => {
                Swal.showLoading();
            }
        });
    }

    // Input dialog
    static input(title, inputPlaceholder, inputType = 'text', inputValue = '') {
        return Swal.fire({
            title,
            input: inputType,
            inputPlaceholder,
            inputValue,
            showCancelButton: true,
            confirmButtonColor: '#2563eb',
            inputValidator: (value) => {
                if (!value) {
                    return 'Input is required';
                }
            }
        });
    }

    // Password input dialog
    static passwordInput(title, placeholder = 'Enter password') {
        return Swal.fire({
            title,
            input: 'password',
            inputPlaceholder: placeholder,
            showCancelButton: true,
            confirmButtonColor: '#2563eb',
            inputValidator: (value) => {
                if (!value) {
                    return 'Password is required';
                }
                if (value.length < 8) {
                    return 'Password must be at least 8 characters long';
                }
            }
        });
    }

    // Progress dialog with custom content
    static progressDialog(title, htmlContent, width = 600) {
        return Swal.fire({
            title,
            html: htmlContent,
            width,
            showCloseButton: true,
            showConfirmButton: false,
            allowOutsideClick: false
        });
    }

    // Custom HTML dialog
    static customDialog(title, htmlContent, options = {}) {
        const defaultOptions = {
            title,
            html: htmlContent,
            showCancelButton: true,
            confirmButtonColor: '#2563eb',
            cancelButtonColor: '#6b7280',
            width: 600
        };

        return Swal.fire({
            ...defaultOptions,
            ...options
        });
    }

    // Toast notification (small popup)
    static toast(message, icon = 'success', position = 'top-end') {
        const Toast = Swal.mixin({
            toast: true,
            position,
            showConfirmButton: false,
            timer: 3000,
            timerProgressBar: true,
            didOpen: (toast) => {
                toast.addEventListener('mouseenter', Swal.stopTimer);
                toast.addEventListener('mouseleave', Swal.resumeTimer);
            }
        });

        return Toast.fire({
            icon,
            title: message
        });
    }

    // Network error handler
    static networkError() {
        return this.error(
            'Network Error',
            'Unable to connect to the server. Please check your internet connection and try again.'
        );
    }

    // Authentication error handler
    static authError() {
        return this.error(
            'Authentication Error',
            'Your session has expired. Please log in again.'
        ).then(() => {
            window.location.href = '/login';
        });
    }
}

export default SweetAlertHelpers;