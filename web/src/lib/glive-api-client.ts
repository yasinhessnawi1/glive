// GLive API Client
import type {
  Project,
  CreateProjectRequest,
  APIResponse,
  APIErrorResponse,
  Config,
  ConfigUpdateRequest,
} from '@/types/glive';

const DEFAULT_API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export class GliveAPIClient {
  private baseURL: string;

  constructor(baseURL: string = DEFAULT_API_URL) {
    this.baseURL = baseURL.replace(/\/$/, ''); // Remove trailing slash
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<APIResponse<T>> {
    const url = `${this.baseURL}${endpoint}`;

    const defaultHeaders: HeadersInit = {
      'Content-Type': 'application/json',
    };

    const response = await fetch(url, {
      ...options,
      headers: {
        ...defaultHeaders,
        ...options.headers,
      },
    });

    const data = await response.json();

    if (!response.ok) {
      const error = data as APIErrorResponse;
      throw new APIError(
        error.error?.code || 'UNKNOWN_ERROR',
        error.error?.message || 'An unknown error occurred',
        error.error?.details
      );
    }

    return data as APIResponse<T>;
  }

  private get<T>(endpoint: string): Promise<APIResponse<T>> {
    return this.request<T>(endpoint, { method: 'GET' });
  }

  private post<T>(endpoint: string, body?: unknown): Promise<APIResponse<T>> {
    return this.request<T>(endpoint, {
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  private put<T>(endpoint: string, body?: unknown): Promise<APIResponse<T>> {
    return this.request<T>(endpoint, {
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  private delete<T>(endpoint: string): Promise<APIResponse<T>> {
    return this.request<T>(endpoint, { method: 'DELETE' });
  }

  // Health Check
  async healthCheck(): Promise<{ status: string; version: string }> {
    const response = await fetch(`${this.baseURL}/health`);
    return response.json();
  }

  // Projects
  async listProjects(): Promise<Project[]> {
    const response = await this.get<{ projects: Project[] }>('/api/v1/projects');
    return response.data.projects || [];
  }

  async getProject(id: string): Promise<Project> {
    const response = await this.get<Project>(`/api/v1/projects/${id}`);
    return response.data;
  }

  async createProject(request: CreateProjectRequest): Promise<Project> {
    const response = await this.post<Project>('/api/v1/projects', request);
    return response.data;
  }

  async deleteProject(id: string): Promise<void> {
    await this.delete(`/api/v1/projects/${id}`);
  }

  async startProject(id: string): Promise<void> {
    await this.post(`/api/v1/projects/${id}/start`);
  }

  async stopProject(id: string): Promise<void> {
    await this.post(`/api/v1/projects/${id}/stop`);
  }

  async cleanupProject(id: string): Promise<void> {
    await this.post(`/api/v1/projects/${id}/cleanup`);
  }

  // Download project as ZIP
  getDownloadZipURL(id: string): string {
    return `${this.baseURL}/api/v1/projects/${id}/download`;
  }

  // Get VS Code URLs for opening project
  async getVSCodeURLs(id: string): Promise<{
    local_vscode_url: string;
    web_vscode_url: string;
    github_dev_url: string;
    project_path: string;
  }> {
    const response = await this.get<{
      local_vscode_url: string;
      web_vscode_url: string;
      github_dev_url: string;
      project_path: string;
    }>(`/api/v1/projects/${id}/vscode`);
    return response.data;
  }

  // Get execution report
  async getExecutionReport(id: string): Promise<{
    project_id: string;
    project_name: string;
    github_url: string;
    local_path: string;
    project_type: string;
    status: string;
    created_at: string;
    total_duration: string;
    steps: Array<{
      step: number;
      stage: string;
      command?: string;
      description: string;
      output?: string;
      duration?: string;
      timestamp: string;
      success: boolean;
    }>;
    summary: string;
    next_steps: string[];
  }> {
    const response = await this.get<{
      project_id: string;
      project_name: string;
      github_url: string;
      local_path: string;
      project_type: string;
      status: string;
      created_at: string;
      total_duration: string;
      steps: Array<{
        step: number;
        stage: string;
        command?: string;
        description: string;
        output?: string;
        duration?: string;
        timestamp: string;
        success: boolean;
      }>;
      summary: string;
      next_steps: string[];
    }>(`/api/v1/projects/${id}/report`);
    return response.data;
  }

  // Get execution report as markdown download URL
  getExecutionReportMarkdownURL(id: string): string {
    return `${this.baseURL}/api/v1/projects/${id}/report?format=markdown`;
  }

  // Configuration
  async getConfig(): Promise<Config> {
    const response = await this.get<Config>('/api/v1/config');
    return response.data;
  }

  async updateConfig(config: ConfigUpdateRequest): Promise<Config> {
    const response = await this.put<Config>('/api/v1/config', config);
    return response.data;
  }
}

// Custom Error Class
export class APIError extends Error {
  code: string;
  details?: Record<string, unknown>;

  constructor(code: string, message: string, details?: Record<string, unknown>) {
    super(message);
    this.name = 'APIError';
    this.code = code;
    this.details = details;
  }
}

// Singleton instance
export const gliveAPI = new GliveAPIClient();

