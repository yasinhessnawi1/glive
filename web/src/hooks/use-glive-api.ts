// React hook for GLive API
import { useState, useCallback } from 'react';
import { gliveAPI, APIError } from '@/lib/glive-api-client';
import type { Project, CreateProjectRequest, Config, ConfigUpdateRequest } from '@/types/glive';

export function useGliveAPI() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<APIError | null>(null);

  const handleError = useCallback((err: unknown) => {
    if (err instanceof APIError) {
      setError(err);
    } else {
      setError(new APIError('UNKNOWN_ERROR', 'An unknown error occurred'));
    }
  }, []);

  const listProjects = useCallback(async (): Promise<Project[]> => {
    setLoading(true);
    setError(null);
    try {
      const projects = await gliveAPI.listProjects();
      return projects;
    } catch (err) {
      handleError(err);
      return [];
    } finally {
      setLoading(false);
    }
  }, [handleError]);

  const getProject = useCallback(async (id: string): Promise<Project | null> => {
    setLoading(true);
    setError(null);
    try {
      const project = await gliveAPI.getProject(id);
      return project;
    } catch (err) {
      handleError(err);
      return null;
    } finally {
      setLoading(false);
    }
  }, [handleError]);

  const createProject = useCallback(async (request: CreateProjectRequest): Promise<Project | null> => {
    setLoading(true);
    setError(null);
    try {
      const project = await gliveAPI.createProject(request);
      return project;
    } catch (err) {
      handleError(err);
      return null;
    } finally {
      setLoading(false);
    }
  }, [handleError]);

  const deleteProject = useCallback(async (id: string): Promise<boolean> => {
    setLoading(true);
    setError(null);
    try {
      await gliveAPI.deleteProject(id);
      return true;
    } catch (err) {
      handleError(err);
      return false;
    } finally {
      setLoading(false);
    }
  }, [handleError]);

  const startProject = useCallback(async (id: string): Promise<boolean> => {
    setLoading(true);
    setError(null);
    try {
      await gliveAPI.startProject(id);
      return true;
    } catch (err) {
      handleError(err);
      return false;
    } finally {
      setLoading(false);
    }
  }, [handleError]);

  const stopProject = useCallback(async (id: string): Promise<boolean> => {
    setLoading(true);
    setError(null);
    try {
      await gliveAPI.stopProject(id);
      return true;
    } catch (err) {
      handleError(err);
      return false;
    } finally {
      setLoading(false);
    }
  }, [handleError]);

  const cleanupProject = useCallback(async (id: string): Promise<boolean> => {
    setLoading(true);
    setError(null);
    try {
      await gliveAPI.cleanupProject(id);
      return true;
    } catch (err) {
      handleError(err);
      return false;
    } finally {
      setLoading(false);
    }
  }, [handleError]);

  const getConfig = useCallback(async (): Promise<Config | null> => {
    setLoading(true);
    setError(null);
    try {
      const config = await gliveAPI.getConfig();
      return config;
    } catch (err) {
      handleError(err);
      return null;
    } finally {
      setLoading(false);
    }
  }, [handleError]);

  const updateConfig = useCallback(async (config: ConfigUpdateRequest): Promise<Config | null> => {
    setLoading(true);
    setError(null);
    try {
      const updated = await gliveAPI.updateConfig(config);
      return updated;
    } catch (err) {
      handleError(err);
      return null;
    } finally {
      setLoading(false);
    }
  }, [handleError]);

  // Get download ZIP URL
  const getDownloadZipURL = useCallback((id: string): string => {
    return gliveAPI.getDownloadZipURL(id);
  }, []);

  // Get VS Code URLs
  const getVSCodeURLs = useCallback(async (id: string) => {
    setLoading(true);
    setError(null);
    try {
      const urls = await gliveAPI.getVSCodeURLs(id);
      return urls;
    } catch (err) {
      handleError(err);
      return null;
    } finally {
      setLoading(false);
    }
  }, [handleError]);

  // Get execution report
  const getExecutionReport = useCallback(async (id: string) => {
    setLoading(true);
    setError(null);
    try {
      const report = await gliveAPI.getExecutionReport(id);
      return report;
    } catch (err) {
      handleError(err);
      return null;
    } finally {
      setLoading(false);
    }
  }, [handleError]);

  // Get execution report markdown URL
  const getExecutionReportMarkdownURL = useCallback((id: string): string => {
    return gliveAPI.getExecutionReportMarkdownURL(id);
  }, []);

  return {
    loading,
    error,
    listProjects,
    getProject,
    createProject,
    deleteProject,
    startProject,
    stopProject,
    cleanupProject,
    getConfig,
    updateConfig,
    getDownloadZipURL,
    getVSCodeURLs,
    getExecutionReport,
    getExecutionReportMarkdownURL,
  };
}

// Alias for backwards compatibility
export const useGliveApi = useGliveAPI;

