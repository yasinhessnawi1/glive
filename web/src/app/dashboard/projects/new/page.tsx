'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Switch } from '@/components/ui/switch';
import { useGliveAPI } from '@/hooks/use-glive-api';
import type { ExecutionMode } from '@/types/glive';
import { ArrowLeft, Loader2, CheckCircle2 } from 'lucide-react';
import Link from 'next/link';

export default function NewProjectPage() {
  const router = useRouter();
  const { createProject, loading, error } = useGliveAPI();
  const [step, setStep] = useState(1);
  const [formData, setFormData] = useState({
    github_url: '',
    mode: 'auto' as ExecutionMode,
    enable_sandbox: false,
    enable_ai_recovery: true,
  });
  const [createdProject, setCreatedProject] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (step < 4) {
      setStep(step + 1);
      return;
    }

    // Final submission
    const project = await createProject({
      github_url: formData.github_url,
      mode: formData.mode,
      enable_sandbox: formData.enable_sandbox,
      enable_ai_recovery: formData.enable_ai_recovery,
    });

    if (project) {
      setCreatedProject(project.id);
      setTimeout(() => {
        router.push(`/dashboard/projects/${project.id}`);
      }, 2000);
    }
  };

  const validateURL = (url: string): boolean => {
    try {
      const githubPattern = /^https?:\/\/(www\.)?(github\.com|gitlab\.com|bitbucket\.org)\/.+\/.+/;
      return githubPattern.test(url);
    } catch {
      return false;
    }
  };

  if (createdProject) {
    return (
      <div className="min-h-screen bg-slate-50 dark:bg-slate-900 flex items-center justify-center">
        <Card className="max-w-md w-full">
          <CardHeader className="text-center">
            <CheckCircle2 className="h-16 w-16 text-green-600 mx-auto mb-4" />
            <CardTitle>Project Created!</CardTitle>
            <CardDescription>
              Redirecting to project page...
            </CardDescription>
          </CardHeader>
        </Card>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900">
      <header className="bg-white dark:bg-slate-800 shadow-sm border-b border-slate-200 dark:border-slate-700">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center h-16 gap-4">
            <Link href="/dashboard">
              <Button variant="ghost" size="sm">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back
              </Button>
            </Link>
            <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-100">
              Create New Project
            </h1>
          </div>
        </div>
      </header>

      <main className="max-w-4xl mx-auto py-8 px-4 sm:px-6 lg:px-8">
        {/* Progress Steps */}
        <div className="mb-8">
          <div className="flex items-center justify-between">
            {[1, 2, 3, 4].map((s) => (
              <div key={s} className="flex items-center flex-1">
                <div className="flex flex-col items-center flex-1">
                  <div
                    className={`w-10 h-10 rounded-full flex items-center justify-center font-semibold ${
                      step >= s
                        ? 'bg-blue-600 text-white'
                        : 'bg-slate-200 dark:bg-slate-700 text-slate-600'
                    }`}
                  >
                    {step > s ? <CheckCircle2 className="h-5 w-5" /> : s}
                  </div>
                  <div className="mt-2 text-xs text-center text-slate-600 dark:text-slate-400">
                    {s === 1 && 'URL'}
                    {s === 2 && 'Mode'}
                    {s === 3 && 'Options'}
                    {s === 4 && 'Review'}
                  </div>
                </div>
                {s < 4 && (
                  <div
                    className={`h-1 flex-1 mx-2 ${
                      step > s ? 'bg-blue-600' : 'bg-slate-200 dark:bg-slate-700'
                    }`}
                  />
                )}
              </div>
            ))}
          </div>
        </div>

        <form onSubmit={handleSubmit}>
          <Card>
            <CardHeader>
              <CardTitle>
                {step === 1 && 'Step 1: GitHub Repository URL'}
                {step === 2 && 'Step 2: Execution Mode'}
                {step === 3 && 'Step 3: Advanced Options'}
                {step === 4 && 'Step 4: Review & Create'}
              </CardTitle>
              <CardDescription>
                {step === 1 && 'Enter the GitHub repository URL you want to set up'}
                {step === 2 && 'Choose how GLive should handle the project setup'}
                {step === 3 && 'Configure sandbox and AI recovery options'}
                {step === 4 && 'Review your settings and create the project'}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              {/* Step 1: URL Input */}
              {step === 1 && (
                <div className="space-y-4">
                  <div>
                    <Label htmlFor="github_url">GitHub Repository URL</Label>
                    <Input
                      id="github_url"
                      type="url"
                      placeholder="https://github.com/username/repository"
                      value={formData.github_url}
                      onChange={(e) =>
                        setFormData({ ...formData, github_url: e.target.value })
                      }
                      required
                    />
                    <p className="text-sm text-muted-foreground mt-2">
                      Enter a valid GitHub, GitLab, or Bitbucket repository URL
                    </p>
                  </div>
                  {formData.github_url && !validateURL(formData.github_url) && (
                    <p className="text-sm text-red-600">
                      Please enter a valid repository URL
                    </p>
                  )}
                </div>
              )}

              {/* Step 2: Mode Selection */}
              {step === 2 && (
                <RadioGroup
                  value={formData.mode}
                  onValueChange={(value) =>
                    setFormData({ ...formData, mode: value as ExecutionMode })
                  }
                >
                  <div className="flex items-start space-x-2 p-4 border rounded-lg hover:bg-accent cursor-pointer">
                    <RadioGroupItem value="auto" id="auto" className="mt-1" />
                    <div className="flex-1">
                      <Label htmlFor="auto" className="font-semibold cursor-pointer">
                        Auto Mode
                      </Label>
                      <p className="text-sm text-muted-foreground">
                        Fully automatic setup. GLive will analyze and run the project without
                        user intervention.
                      </p>
                    </div>
                  </div>
                  <div className="flex items-start space-x-2 p-4 border rounded-lg hover:bg-accent cursor-pointer">
                    <RadioGroupItem value="assisted" id="assisted" className="mt-1" />
                    <div className="flex-1">
                      <Label htmlFor="assisted" className="font-semibold cursor-pointer">
                        Assisted Mode
                      </Label>
                      <p className="text-sm text-muted-foreground">
                        GLive will prompt you for approval before executing each step.
                      </p>
                    </div>
                  </div>
                  <div className="flex items-start space-x-2 p-4 border rounded-lg hover:bg-accent cursor-pointer">
                    <RadioGroupItem value="manual" id="manual" className="mt-1" />
                    <div className="flex-1">
                      <Label htmlFor="manual" className="font-semibold cursor-pointer">
                        Manual Mode
                      </Label>
                      <p className="text-sm text-muted-foreground">
                        GLive will only provide instructions. You execute commands manually.
                      </p>
                    </div>
                  </div>
                </RadioGroup>
              )}

              {/* Step 3: Options */}
              {step === 3 && (
                <div className="space-y-6">
                  <div className="flex items-center justify-between p-4 border rounded-lg">
                    <div className="flex-1">
                      <Label htmlFor="sandbox" className="font-semibold">
                        Enable Sandbox
                      </Label>
                      <p className="text-sm text-muted-foreground">
                        Run commands in an isolated sandbox environment for safety
                      </p>
                    </div>
                    <Switch
                      id="sandbox"
                      checked={formData.enable_sandbox}
                      onCheckedChange={(checked) =>
                        setFormData({ ...formData, enable_sandbox: checked })
                      }
                    />
                  </div>
                  <div className="flex items-center justify-between p-4 border rounded-lg">
                    <div className="flex-1">
                      <Label htmlFor="ai_recovery" className="font-semibold">
                        Enable AI Recovery
                      </Label>
                      <p className="text-sm text-muted-foreground">
                        Automatically attempt to recover from errors using AI
                      </p>
                    </div>
                    <Switch
                      id="ai_recovery"
                      checked={formData.enable_ai_recovery}
                      onCheckedChange={(checked) =>
                        setFormData({ ...formData, enable_ai_recovery: checked })
                      }
                    />
                  </div>
                </div>
              )}

              {/* Step 4: Review */}
              {step === 4 && (
                <div className="space-y-4">
                  <div className="p-4 bg-slate-50 dark:bg-slate-800 rounded-lg">
                    <h3 className="font-semibold mb-2">Repository URL</h3>
                    <p className="text-sm text-muted-foreground">{formData.github_url}</p>
                  </div>
                  <div className="p-4 bg-slate-50 dark:bg-slate-800 rounded-lg">
                    <h3 className="font-semibold mb-2">Execution Mode</h3>
                    <p className="text-sm text-muted-foreground capitalize">{formData.mode}</p>
                  </div>
                  <div className="p-4 bg-slate-50 dark:bg-slate-800 rounded-lg">
                    <h3 className="font-semibold mb-2">Options</h3>
                    <div className="space-y-1 text-sm text-muted-foreground">
                      <p>Sandbox: {formData.enable_sandbox ? 'Enabled' : 'Disabled'}</p>
                      <p>AI Recovery: {formData.enable_ai_recovery ? 'Enabled' : 'Disabled'}</p>
                    </div>
                  </div>
                </div>
              )}

              {error && (
                <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
                  <p className="text-sm text-red-600 dark:text-red-400">
                    {error.message}
                  </p>
                </div>
              )}

              <div className="flex justify-between pt-4">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setStep(Math.max(1, step - 1))}
                  disabled={step === 1 || loading}
                >
                  Previous
                </Button>
                <Button type="submit" disabled={loading || (step === 1 && !validateURL(formData.github_url))}>
                  {loading ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      Creating...
                    </>
                  ) : step === 4 ? (
                    'Create Project'
                  ) : (
                    'Next'
                  )}
                </Button>
              </div>
            </CardContent>
          </Card>
        </form>
      </main>
    </div>
  );
}

