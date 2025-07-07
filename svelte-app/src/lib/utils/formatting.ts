export const formatDate = (dateString: string): string => {
	return new Date(dateString).toLocaleDateString('en-US', {
		year: 'numeric',
		month: 'short',
		day: 'numeric',
		hour: '2-digit',
		minute: '2-digit'
	});
};

export const formatRelativeTime = (dateString: string): string => {
	const now = new Date();
	const date = new Date(dateString);
	const diffMs = now.getTime() - date.getTime();
	const diffMins = Math.floor(diffMs / 60000);
	const diffHours = Math.floor(diffMins / 60);
	const diffDays = Math.floor(diffHours / 24);

	if (diffMins < 1) {
		return 'Just now';
	} else if (diffMins < 60) {
		return `${diffMins}m ago`;
	} else if (diffHours < 24) {
		return `${diffHours}h ago`;
	} else if (diffDays < 7) {
		return `${diffDays}d ago`;
	} else {
		return formatDate(dateString);
	}
};

export const formatFileSize = (bytes: number): string => {
	if (bytes === 0) return '0 B';
	
	const k = 1024;
	const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	
	return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
};

export const formatUptime = (uptimeSeconds: number): string => {
	const days = Math.floor(uptimeSeconds / 86400);
	const hours = Math.floor((uptimeSeconds % 86400) / 3600);
	const minutes = Math.floor((uptimeSeconds % 3600) / 60);

	if (days > 0) {
		return `${days}d ${hours}h ${minutes}m`;
	} else if (hours > 0) {
		return `${hours}h ${minutes}m`;
	} else {
		return `${minutes}m`;
	}
};

export const formatStatus = (status: string): string => {
	return status.charAt(0).toUpperCase() + status.slice(1);
};

export const getStatusColor = (status: string): string => {
	switch (status.toLowerCase()) {
		case 'running':
			return 'text-green-600 bg-green-100';
		case 'stopped':
			return 'text-red-600 bg-red-100';
		case 'pending':
			return 'text-yellow-600 bg-yellow-100';
		case 'error':
			return 'text-red-600 bg-red-100';
		case 'starting':
			return 'text-blue-600 bg-blue-100';
		case 'stopping':
			return 'text-orange-600 bg-orange-100';
		default:
			return 'text-gray-600 bg-gray-100';
	}
};

export const getStatusIcon = (status: string): string => {
	switch (status.toLowerCase()) {
		case 'running':
			return '✅';
		case 'stopped':
			return '⏹️';
		case 'pending':
			return '⏳';
		case 'error':
			return '❌';
		case 'starting':
			return '🚀';
		case 'stopping':
			return '⏸️';
		default:
			return '❓';
	}
};

export const truncateText = (text: string, maxLength: number): string => {
	if (text.length <= maxLength) {
		return text;
	}
	return text.slice(0, maxLength) + '...';
};

export const capitalizeFirst = (text: string): string => {
	return text.charAt(0).toUpperCase() + text.slice(1);
};

export const kebabToTitle = (text: string): string => {
	return text
		.split('-')
		.map(word => capitalizeFirst(word))
		.join(' ');
};

export const extractDomain = (url: string): string => {
	try {
		const urlObj = new URL(url);
		const parts = urlObj.hostname.split('.');
		return parts.slice(1).join('.');
	} catch (error) {
		console.error('Error extracting domain from URL:', url, error);
		return '';
	}
};

export const debounce = <T extends (...args: any[]) => any>(
	func: T,
	wait: number
): (...args: Parameters<T>) => void => {
	let timeout: NodeJS.Timeout;
	return (...args: Parameters<T>) => {
		clearTimeout(timeout);
		timeout = setTimeout(() => func.apply(this, args), wait);
	};
};