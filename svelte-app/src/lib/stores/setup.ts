import { writable } from 'svelte/store';
import { api } from '$lib/services/api';
import { errorHandler } from '$lib/services/errorHandler';

export interface SetupConfig {
  hetznerToken?: string;
  cloudflareToken?: string;
  step: number;
  isComplete: boolean;
}

export interface SetupStatus {
  isSetupRequired: boolean;
  hasHetznerToken: boolean;
  hasCloudflareToken: boolean;
  currentStep: number;
}

class SetupStore {
  private store = writable<SetupConfig>({
    step: 1,
    isComplete: false
  });

  private statusStore = writable<SetupStatus>({
    isSetupRequired: true,
    hasHetznerToken: false,
    hasCloudflareToken: false,
    currentStep: 1
  });

  subscribe = this.store.subscribe;
  subscribeStatus = this.statusStore.subscribe;

  async checkSetupStatus(): Promise<SetupStatus> {
    try {
      const response = await api.get('/setup/status');
      const status = response.data;
      
      this.statusStore.set(status);
      
      // Update main store with current step
      this.store.update(config => ({
        ...config,
        step: status.currentStep,
        isComplete: !status.isSetupRequired
      }));
      
      return status;
    } catch (error) {
      console.error('Failed to check setup status:', error);
      await errorHandler.handleAPIError(error, 'setup status check');
      
      // Default to requiring setup on error
      const defaultStatus: SetupStatus = {
        isSetupRequired: true,
        hasHetznerToken: false,
        hasCloudflareToken: false,
        currentStep: 1
      };
      
      this.statusStore.set(defaultStatus);
      return defaultStatus;
    }
  }

  async configureHetzner(hetznerToken: string): Promise<boolean> {
    try {
      const response = await api.post('/setup/hetzner', {
        hetzner_token: hetznerToken
      });

      if (response.success) {
        this.store.update(config => ({
          ...config,
          hetznerToken,
          step: 2
        }));

        this.statusStore.update(status => ({
          ...status,
          hasHetznerToken: true,
          currentStep: 2
        }));

        errorHandler.showSuccess('Hetzner Setup Complete', 'Hetzner API token configured successfully');
        return true;
      } else {
        throw new Error(response.message || 'Failed to configure Hetzner token');
      }
    } catch (error) {
      console.error('Hetzner setup failed:', error);
      await errorHandler.handleAPIError(error, 'Hetzner setup');
      return false;
    }
  }

  async configureCloudflare(cloudflareToken: string): Promise<boolean> {
    try {
      // Use existing auth login endpoint since Cloudflare token setup is done via login
      const response = await api.post('/auth/login', {
        cf_token: cloudflareToken
      });

      if (response.access_token) {
        this.store.update(config => ({
          ...config,
          cloudflareToken,
          step: 3,
          isComplete: true
        }));

        this.statusStore.update(status => ({
          ...status,
          hasCloudflareToken: true,
          currentStep: 3,
          isSetupRequired: false
        }));

        errorHandler.showSuccess('Setup Complete', 'Platform setup completed successfully');
        return true;
      } else {
        throw new Error('Invalid Cloudflare token');
      }
    } catch (error) {
      console.error('Cloudflare setup failed:', error);
      await errorHandler.handleAPIError(error, 'Cloudflare setup');
      return false;
    }
  }

  async completeSetup(config: { hetznerToken: string; cloudflareToken: string }): Promise<boolean> {
    try {
      // First configure Hetzner
      const hetznerSuccess = await this.configureHetzner(config.hetznerToken);
      if (!hetznerSuccess) {
        return false;
      }

      // Then configure Cloudflare (which also logs the user in)
      const cloudflareSuccess = await this.configureCloudflare(config.cloudflareToken);
      if (!cloudflareSuccess) {
        return false;
      }

      return true;
    } catch (error) {
      console.error('Setup completion failed:', error);
      await errorHandler.handleAPIError(error, 'setup completion');
      return false;
    }
  }

  reset(): void {
    this.store.set({
      step: 1,
      isComplete: false
    });

    this.statusStore.set({
      isSetupRequired: true,
      hasHetznerToken: false,
      hasCloudflareToken: false,
      currentStep: 1
    });
  }

  setStep(step: number): void {
    this.store.update(config => ({
      ...config,
      step
    }));
  }

  nextStep(): void {
    this.store.update(config => ({
      ...config,
      step: Math.min(config.step + 1, 3)
    }));
  }

  previousStep(): void {
    this.store.update(config => ({
      ...config,
      step: Math.max(config.step - 1, 1)
    }));
  }
}

export const setupStore = new SetupStore();