'use client';

import { useEffect, useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { useGliveAPI } from '@/hooks/use-glive-api';
import { useSettingsStore } from '@/stores/settings-store';
import type { Config } from '@/types/glive';
import { Save, Loader2 } from 'lucide-react';

export default function SettingsPage() {
  const { getConfig, updateConfig, loading } = useGliveAPI();
  const { config: localConfig, setConfig } = useSettingsStore();
  const [config, setConfigState] = useState<Config | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    const loadConfig = async () => {
      const data = await getConfig();
      if (data) {
        setConfigState(data);
        setConfig(data);
      } else if (localConfig) {
        setConfigState(localConfig);
      }
    };
    loadConfig();
  }, [getConfig, localConfig, setConfig]);

  const handleSave = async () => {
    if (!config) return;
    setSaving(true);
    const updated = await updateConfig({
      api_provider: config.api_provider,
      default_mode: config.default_mode,
      workspace_dir: config.workspace_dir,
      agent_port: config.agent_port,
      enable_sandbox: config.enable_sandbox,
      enable_ai_recovery: config.enable_ai_recovery,
      ai_recovery_confidence_threshold: config.ai_recovery_confidence_threshold,
    });
    if (updated) {
      setConfigState(updated);
      setConfig(updated);
    }
    setSaving(false);
  };

  if (loading && !config) {
    return (
      <div className="min-h-screen bg-slate-50 dark:bg-slate-900 flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-blue-600" />
      </div>
    );
  }

  if (!config) {
    return (
      <div className="min-h-screen bg-slate-50 dark:bg-slate-900">
        <div className="max-w-4xl mx-auto py-8 px-4">
          <Card>
            <CardHeader>
              <CardTitle>Settings</CardTitle>
              <CardDescription>Unable to load settings</CardDescription>
            </CardHeader>
          </Card>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900">
      <header className="bg-white dark:bg-slate-800 shadow-sm border-b border-slate-200 dark:border-slate-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center h-16">
            <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-100">
              Settings
            </h1>
          </div>
        </div>
      </header>

      <main className="max-w-4xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0 space-y-6">
          {/* API Settings */}
          <Card>
            <CardHeader>
              <CardTitle>API Settings</CardTitle>
              <CardDescription>Configure AI provider and API settings</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label htmlFor="api_provider">AI Provider</Label>
                <Select
                  value={config.api_provider}
                  onValueChange={(value: Config['api_provider']) =>
                    setConfigState({ ...config, api_provider: value })
                  }
                >
                  <SelectTrigger id="api_provider">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="deepseek">DeepSeek</SelectItem>
                    <SelectItem value="openai">OpenAI</SelectItem>
                    <SelectItem value="claude">Claude (Anthropic)</SelectItem>
                    <SelectItem value="ollama">Ollama</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </CardContent>
          </Card>

          {/* Execution Settings */}
          <Card>
            <CardHeader>
              <CardTitle>Execution Settings</CardTitle>
              <CardDescription>Configure default execution behavior</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label htmlFor="default_mode">Default Execution Mode</Label>
                <Select
                  value={config.default_mode}
                  onValueChange={(value: Config['default_mode']) =>
                    setConfigState({ ...config, default_mode: value })
                  }
                >
                  <SelectTrigger id="default_mode">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="auto">Auto</SelectItem>
                    <SelectItem value="assisted">Assisted</SelectItem>
                    <SelectItem value="manual">Manual</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div>
                <Label htmlFor="workspace_dir">Workspace Directory</Label>
                <Input
                  id="workspace_dir"
                  value={config.workspace_dir}
                  onChange={(e) =>
                    setConfigState({ ...config, workspace_dir: e.target.value })
                  }
                />
              </div>
              <div>
                <Label htmlFor="agent_port">Agent Port</Label>
                <Input
                  id="agent_port"
                  type="number"
                  value={config.agent_port}
                  onChange={(e) =>
                    setConfigState({ ...config, agent_port: parseInt(e.target.value) || 8080 })
                  }
                />
              </div>
            </CardContent>
          </Card>

          {/* Security Settings */}
          <Card>
            <CardHeader>
              <CardTitle>Security Settings</CardTitle>
              <CardDescription>Configure sandbox and security options</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <Label htmlFor="sandbox">Enable Sandbox</Label>
                  <p className="text-sm text-muted-foreground">
                    Run commands in an isolated sandbox environment
                  </p>
                </div>
                <Switch
                  id="sandbox"
                  checked={config.enable_sandbox}
                  onCheckedChange={(checked) =>
                    setConfigState({ ...config, enable_sandbox: checked })
                  }
                />
              </div>
            </CardContent>
          </Card>

          {/* AI Recovery Settings */}
          <Card>
            <CardHeader>
              <CardTitle>AI Recovery Settings</CardTitle>
              <CardDescription>Configure AI-powered error recovery</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <Label htmlFor="ai_recovery">Enable AI Recovery</Label>
                  <p className="text-sm text-muted-foreground">
                    Automatically attempt to recover from errors using AI
                  </p>
                </div>
                <Switch
                  id="ai_recovery"
                  checked={config.enable_ai_recovery || false}
                  onCheckedChange={(checked) =>
                    setConfigState({ ...config, enable_ai_recovery: checked })
                  }
                />
              </div>
              {config.enable_ai_recovery && (
                <div>
                  <Label htmlFor="confidence">Confidence Threshold</Label>
                  <Input
                    id="confidence"
                    type="number"
                    min="0"
                    max="1"
                    step="0.1"
                    value={config.ai_recovery_confidence_threshold || 0.7}
                    onChange={(e) =>
                      setConfigState({
                        ...config,
                        ai_recovery_confidence_threshold: parseFloat(e.target.value) || 0.7,
                      })
                    }
                  />
                  <p className="text-sm text-muted-foreground mt-1">
                    Minimum confidence level (0.0 - 1.0) for AI recovery attempts
                  </p>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Save Button */}
          <div className="flex justify-end">
            <Button onClick={handleSave} disabled={saving}>
              {saving ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Saving...
                </>
              ) : (
                <>
                  <Save className="h-4 w-4 mr-2" />
                  Save Settings
                </>
              )}
            </Button>
          </div>
        </div>
      </main>
    </div>
  );
}

