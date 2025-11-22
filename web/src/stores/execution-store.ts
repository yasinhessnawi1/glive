// Zustand store for execution state
import { create } from 'zustand';
import type { Project, ExecutionEvent, RecoveryPlan } from '@/types/glive';

interface ExecutionState {
  // Current project
  currentProject: Project | null;
  
  // Execution events
  executionEvents: ExecutionEvent[];
  
  // Recovery plans
  recoveryPlans: RecoveryPlan[];
  
  // Connection state
  isLive: boolean;
  isConnected: boolean;

  // Actions
  setProject: (project: Project | null) => void;
  addEvent: (event: ExecutionEvent) => void;
  addEvents: (events: ExecutionEvent[]) => void;
  addRecoveryPlan: (plan: RecoveryPlan) => void;
  updateRecoveryPlan: (planId: string, updates: Partial<RecoveryPlan>) => void;
  setLive: (isLive: boolean) => void;
  setConnected: (isConnected: boolean) => void;
  clearEvents: () => void;
  clearRecoveryPlans: () => void;
  reset: () => void;
}

export const useExecutionStore = create<ExecutionState>((set) => ({
  currentProject: null,
  executionEvents: [],
  recoveryPlans: [],
  isLive: false,
  isConnected: false,

  setProject: (project) => set({ currentProject: project }),
  
  addEvent: (event) =>
    set((state) => ({
      executionEvents: [...state.executionEvents, event],
    })),
  
  addEvents: (events) =>
    set((state) => ({
      executionEvents: [...state.executionEvents, ...events],
    })),
  
  addRecoveryPlan: (plan) =>
    set((state) => ({
      recoveryPlans: [...state.recoveryPlans, plan],
    })),
  
  updateRecoveryPlan: (planId, updates) =>
    set((state) => ({
      recoveryPlans: state.recoveryPlans.map((plan) =>
        plan.id === planId ? { ...plan, ...updates } : plan
      ),
    })),
  
  setLive: (isLive) => set({ isLive }),
  setConnected: (isConnected) => set({ isConnected }),
  
  clearEvents: () => set({ executionEvents: [] }),
  clearRecoveryPlans: () => set({ recoveryPlans: [] }),
  
  reset: () =>
    set({
      currentProject: null,
      executionEvents: [],
      recoveryPlans: [],
      isLive: false,
      isConnected: false,
    }),
}));

