import { writable } from 'svelte/store';
import { browser } from '$app/environment';
import { jwtDecode } from 'jwt-decode';

function createAuthStore() {
  const { subscribe, set, update } = writable({
    token: null,
    user: null,
    isAuthenticated: false,
    isLoading: false
  });

  return {
    subscribe,
    
    async login(cloudflareToken) {
      update(state => ({ ...state, isLoading: true }));
      
      try {
        const response = await fetch('/api/auth/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ token: cloudflareToken })
        });
        
        if (!response.ok) throw new Error('Login failed');
        
        const data = await response.json();
        const decoded = jwtDecode(data.data.jwt_token);
        
        // Store token in localStorage
        if (browser) {
          localStorage.setItem('jwt_token', data.data.jwt_token);
        }
        
        set({
          token: data.data.jwt_token,
          user: decoded,
          isAuthenticated: true,
          isLoading: false
        });
        
        return true;
      } catch (error) {
        set({
          token: null,
          user: null,
          isAuthenticated: false,
          isLoading: false
        });
        throw error;
      }
    },
    
    logout() {
      if (browser) {
        localStorage.removeItem('jwt_token');
      }
      set({
        token: null,
        user: null,
        isAuthenticated: false,
        isLoading: false
      });
    },
    
    checkAuth() {
      if (!browser) return;
      
      const token = localStorage.getItem('jwt_token');
      if (!token) return;
      
      try {
        const decoded = jwtDecode(token);
        
        // Check if token is expired
        if (decoded.exp * 1000 < Date.now()) {
          this.logout();
          return;
        }
        
        set({
          token,
          user: decoded,
          isAuthenticated: true,
          isLoading: false
        });
      } catch (error) {
        this.logout();
      }
    }
  };
}

export const auth = createAuthStore();