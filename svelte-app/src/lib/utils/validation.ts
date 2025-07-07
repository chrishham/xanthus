export interface ValidationErrors {
	[field: string]: string;
}

export class FormValidator {
	private errors: ValidationErrors = {};

	validateRequired(field: string, value: any, fieldName: string): boolean {
		if (!value || (typeof value === 'string' && value.trim() === '')) {
			this.errors[field] = `${fieldName} is required`;
			return false;
		} else {
			delete this.errors[field];
			return true;
		}
	}

	validateEmail(field: string, value: string): boolean {
		const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
		if (value && !emailRegex.test(value)) {
			this.errors[field] = 'Please enter a valid email address';
			return false;
		} else {
			delete this.errors[field];
			return true;
		}
	}

	validateMinLength(field: string, value: string, minLength: number, fieldName: string): boolean {
		if (value && value.length < minLength) {
			this.errors[field] = `${fieldName} must be at least ${minLength} characters long`;
			return false;
		} else {
			delete this.errors[field];
			return true;
		}
	}

	validateMaxLength(field: string, value: string, maxLength: number, fieldName: string): boolean {
		if (value && value.length > maxLength) {
			this.errors[field] = `${fieldName} must be no more than ${maxLength} characters long`;
			return false;
		} else {
			delete this.errors[field];
			return true;
		}
	}

	validatePattern(field: string, value: string, pattern: RegExp, errorMessage: string): boolean {
		if (value && !pattern.test(value)) {
			this.errors[field] = errorMessage;
			return false;
		} else {
			delete this.errors[field];
			return true;
		}
	}

	validateSubdomain(field: string, value: string): boolean {
		const subdomainRegex = /^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/;
		if (value && !subdomainRegex.test(value)) {
			this.errors[field] = 'Subdomain must contain only lowercase letters, numbers, and hyphens (no spaces or special characters)';
			return false;
		} else {
			delete this.errors[field];
			return true;
		}
	}

	validatePort(field: string, value: string | number): boolean {
		const port = typeof value === 'string' ? parseInt(value) : value;
		if (isNaN(port) || port < 1 || port > 65535) {
			this.errors[field] = 'Port must be a number between 1 and 65535';
			return false;
		} else {
			delete this.errors[field];
			return true;
		}
	}

	hasErrors(): boolean {
		return Object.keys(this.errors).length > 0;
	}

	getError(field: string): string {
		return this.errors[field] || '';
	}

	getErrors(): ValidationErrors {
		return { ...this.errors };
	}

	clearErrors(): void {
		this.errors = {};
	}

	clearError(field: string): void {
		delete this.errors[field];
	}
}

export const createValidator = (): FormValidator => new FormValidator();