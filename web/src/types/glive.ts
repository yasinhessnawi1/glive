// GLive Type Definitions

// Execution Modes
export type ExecutionMode = 'auto' | 'assisted' | 'manual';

// Project Types
export type ProjectType = 
  | 'nodejs' 
  | 'python' 
  | 'go' 
  | 'rust' 
  | 'java' 
  | 'docker' 
  | 'unknown' 
  | 'polyglot';

// Project Status
export type ProjectStatus = 
  | 'pending' 
  | 'cloning' 
  | 'analyzing' 
  | 'installing' 
  | 'running' 
  | 'ready' 
  | 'failed' 
  | 'stopped';

// Project Entity
export interface Project {
  id: string;
  name: string;
  github_url: string;
  local_path: string;
  type: ProjectType;
  detected_types?: ProjectType[]; // Optional - may not always be present
  status: ProjectStatus;
  created_at: string;
  updated_at: string;
}

// Project Request
export interface CreateProjectRequest {
  github_url: string;
  mode: ExecutionMode;
  enable_sandbox?: boolean;
  enable_ai_recovery?: boolean;
}

// API Response Wrapper
export interface APIResponse<T> {
  success: boolean;
  data: T;
  meta?: {
    request_id?: string;
    duration?: string;
  };
}

export interface APIError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}

export interface APIErrorResponse {
  success: false;
  error: APIError;
  meta?: {
    request_id?: string;
    duration?: string;
  };
}

// Configuration
export interface Config {
  api_provider: 'deepseek' | 'openai' | 'claude' | 'ollama';
  default_mode: ExecutionMode;
  workspace_dir: string;
  agent_port: number;
  enable_sandbox: boolean;
  enable_ai_recovery?: boolean;
  ai_recovery_confidence_threshold?: number;
}

export interface ConfigUpdateRequest {
  api_provider?: Config['api_provider'];
  default_mode?: ExecutionMode;
  workspace_dir?: string;
  agent_port?: number;
  enable_sandbox?: boolean;
  enable_ai_recovery?: boolean;
  ai_recovery_confidence_threshold?: number;
}

// WebSocket Message Types
export type WSMessageType = 
  | 'project.status' 
  | 'command.output' 
  | 'command.complete' 
  | 'command.started'
  | 'recovery.triggered'
  | 'recovery.plan'
  | 'recovery.step'
  | 'execution.completed'
  | 'error';

// WebSocket Message
export interface WSMessage {
  type: WSMessageType;
  payload: unknown;
}

// WebSocket Payloads
export interface ProjectStatusPayload {
  project_id: string;
  status: ProjectStatus;
  message?: string;
}

export interface CommandOutputPayload {
  project_id: string;
  command_id?: string;
  stream: 'stdout' | 'stderr';
  output: string;
  timestamp?: number;
}

export interface CommandCompletePayload {
  project_id: string;
  command_id?: string;
  exit_code: number;
  success: boolean;
  message?: string;
  duration?: number;
}

export interface CommandStartedPayload {
  project_id: string;
  command_id: string;
  command: string;
  timestamp: number;
}

export interface RecoveryTriggeredPayload {
  project_id: string;
  command_id: string;
  reason: string;
  error_output?: string;
}

export interface RecoveryPlanPayload {
  project_id: string;
  plan_id: string;
  analysis: string;
  steps: RecoveryStep[];
  confidence: number;
}

export interface RecoveryStep {
  id: string;
  description: string;
  command: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
}

export interface RecoveryStepPayload {
  project_id: string;
  plan_id: string;
  step_id: string;
  status: RecoveryStep['status'];
  output?: string;
}

export interface ExecutionCompletedPayload {
  project_id: string;
  status: 'success' | 'failed';
  message?: string;
}

export interface ErrorPayload {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}

// Execution Event (unified type for frontend)
export type ExecutionEvent = 
  | { type: 'command_started'; command: string; command_id: string; timestamp: number }
  | { type: 'output_line'; stream: 'stdout' | 'stderr'; line: string; timestamp?: number }
  | { type: 'command_completed'; exit_code: number; success: boolean; duration?: number }
  | { type: 'recovery_triggered'; reason: string; error_output?: string }
  | { type: 'recovery_plan'; plan: RecoveryPlanPayload }
  | { type: 'recovery_step'; step: RecoveryStepPayload }
  | { type: 'execution_completed'; status: 'success' | 'failed'; message?: string }
  | { type: 'error'; error: ErrorPayload };

// Recovery Plan (for display)
export interface RecoveryPlan {
  id: string;
  project_id: string;
  analysis: string;
  steps: RecoveryStep[];
  confidence: number;
  created_at: string;
}

// Dashboard Stats
export interface DashboardStats {
  total_projects: number;
  active_projects: number;
  success_rate: number;
  recent_activity: ActivityItem[];
}

export interface ActivityItem {
  project_id: string;
  project_name: string;
  action: string;
  timestamp: string;
  status: ProjectStatus;
}

