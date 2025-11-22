// Zustand store for settings
import { create } from 'zustand';
import type { Config } from '@/types/glive';

interface SettingsState {
  config: Config | null;
  setConfig: (config: Config) => void;
  updateConfig: (updates: Partial<Config>) => void;
  reset: () => void;
}

const defaultConfig: Config = {
  api_provider: 'deepseek',
  default_mode: 'auto',
  workspace_dir: '~/glive-workspace',
  agent_port: 8080,
  enable_sandbox: false,
  enable_ai_recovery: true,
  ai_recovery_confidence_threshold: 0.7,
};

export const useSettingsStore = create<SettingsState>((set) => ({
  config: defaultConfig,
  setConfig: (config) => set({ config }),
  updateConfig: (updates) =>
    set((state) => ({
      config: state.config ? { ...state.config, ...updates } : null,
    })),
  reset: () => set({ config: defaultConfig }),
}));

