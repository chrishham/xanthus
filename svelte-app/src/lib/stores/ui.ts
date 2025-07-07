import { writable } from 'svelte/store';

export interface UIState {
	loading: boolean;
	loadingTitle: string;
	loadingMessage: string;
	currentPage: string;
	modals: {
		[key: string]: boolean;
	};
}

const initialState: UIState = {
	loading: false,
	loadingTitle: 'Processing...',
	loadingMessage: 'Please wait while the operation completes.',
	currentPage: '',
	modals: {}
};

export const uiStore = writable<UIState>(initialState);

export const setLoading = (title: string, message: string) => {
	uiStore.update(state => ({
		...state,
		loading: true,
		loadingTitle: title,
		loadingMessage: message
	}));
};

export const clearLoading = () => {
	uiStore.update(state => ({
		...state,
		loading: false
	}));
};

export const showModal = (modalKey: string) => {
	uiStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			[modalKey]: true
		}
	}));
};

export const hideModal = (modalKey: string) => {
	uiStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			[modalKey]: false
		}
	}));
};

export const toggleModal = (modalKey: string) => {
	uiStore.update(state => ({
		...state,
		modals: {
			...state.modals,
			[modalKey]: !state.modals[modalKey]
		}
	}));
};

export const setCurrentPage = (page: string) => {
	uiStore.update(state => ({
		...state,
		currentPage: page
	}));
};