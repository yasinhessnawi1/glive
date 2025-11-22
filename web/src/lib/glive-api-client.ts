// GLive API Client
import type {
  Project,
  CreateProjectRequest,
  APIResponse,
  APIErrorResponse,
  Config,
  ConfigUpdateRequest,
} from '@/types/glive';

const DEFAULT_API_URL = process.env.NEXT_PUBLIC_GLIVE_API_URL || 'http://localhost:8080';

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

