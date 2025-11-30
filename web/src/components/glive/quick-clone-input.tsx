"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Rocket,
  Github,
  CheckCircle2,
  AlertCircle,
  Loader2,
  Clipboard,
  ArrowRight
} from "lucide-react";
import { useGliveApi } from "@/hooks/use-glive-api";
import { cn } from "@/lib/utils";

// Regex patterns for GitHub URL validation
const GITHUB_URL_PATTERN = /^https?:\/\/(www\.)?github\.com\/[\w.-]+\/[\w.-]+\/?$/i;
const SHORT_PATTERN = /^[\w.-]+\/[\w.-]+$/;

type ValidationState = "idle" | "valid" | "invalid" | "loading";

export function QuickCloneInput() {
  const router = useRouter();
  const { createProject, loading } = useGliveApi();

  const [inputValue, setInputValue] = useState("");
  const [validationState, setValidationState] = useState<ValidationState>("idle");
  const [errorMessage, setErrorMessage] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const [clipboardDetected, setClipboardDetected] = useState(false);

  // Validate input
  const validateInput = useCallback((value: string) => {
    if (!value.trim()) {
      setValidationState("idle");
      setErrorMessage("");
      return false;
    }

    // Check if it's a valid GitHub URL
    if (GITHUB_URL_PATTERN.test(value)) {
      setValidationState("valid");
      setErrorMessage("");
      return true;
    }

    // Check if it's a short format (user/repo)
    if (SHORT_PATTERN.test(value)) {
      setValidationState("valid");
      setErrorMessage("");
      return true;
    }

    // Invalid format
    setValidationState("invalid");
    setErrorMessage("Enter a GitHub URL or user/repo format");
    return false;
  }, []);

  // Handle input change
  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setInputValue(value);
    setClipboardDetected(false);
    validateInput(value);
  };

  // Convert short format to full URL
  const normalizeUrl = (value: string): string => {
    if (SHORT_PATTERN.test(value) && !GITHUB_URL_PATTERN.test(value)) {
      return `https://github.com/${value}`;
    }
    return value;
  };

  // Handle clipboard paste detection
  useEffect(() => {
    const handleFocus = async () => {
      try {
        // Check if clipboard API is available
        if (!navigator.clipboard || !navigator.clipboard.readText) return;

        const clipboardText = await navigator.clipboard.readText();

        // Only auto-fill if input is empty and clipboard contains a GitHub URL
        if (!inputValue && clipboardText) {
          const trimmed = clipboardText.trim();
          if (GITHUB_URL_PATTERN.test(trimmed) || SHORT_PATTERN.test(trimmed)) {
            setInputValue(trimmed);
            setClipboardDetected(true);
            validateInput(trimmed);
          }
        }
      } catch {
        // Clipboard access denied or not available - silently ignore
      }
    };

    // Listen for window focus to detect clipboard
    window.addEventListener("focus", handleFocus);

    // Also check on mount
    handleFocus();

    return () => {
      window.removeEventListener("focus", handleFocus);
    };
  }, [inputValue, validateInput]);

  // Handle form submission
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateInput(inputValue)) return;

    setIsCreating(true);
    setValidationState("loading");

    try {
      const normalizedUrl = normalizeUrl(inputValue);

      // Create project with default settings
      const result = await createProject({
        github_url: normalizedUrl,
        mode: "auto",
        enable_sandbox: true,
        enable_ai_recovery: true,
        force_execution: false,
      });

      if (result && result.id) {
        // Redirect to execution page
        router.push(`/dashboard/projects/${result.id}/execution`);
      } else {
        throw new Error("Failed to create project");
      }
    } catch (error) {
      setValidationState("invalid");
      setErrorMessage(error instanceof Error ? error.message : "Failed to create project");
      setIsCreating(false);
    }
  };

  // Handle paste button click
  const handlePasteClick = async () => {
    try {
      const clipboardText = await navigator.clipboard.readText();
      if (clipboardText) {
        const trimmed = clipboardText.trim();
        setInputValue(trimmed);
        validateInput(trimmed);
      }
    } catch {
      // Clipboard access denied
    }
  };

  const isSubmitDisabled = validationState !== "valid" || isCreating || loading;

  return (
    <div className="w-full max-w-2xl mx-auto">
      <form onSubmit={handleSubmit} className="space-y-3">
        {/* Input Container */}
        <div className="relative">
          <div className={cn(
            "flex items-center gap-2 p-2 rounded-xl border-2 bg-background/80 backdrop-blur transition-all duration-200",
            validationState === "valid" && "border-green-500/50 shadow-lg shadow-green-500/10",
            validationState === "invalid" && "border-red-500/50 shadow-lg shadow-red-500/10",
            validationState === "idle" && "border-border hover:border-primary/50",
            validationState === "loading" && "border-primary/50"
          )}>
            {/* GitHub Icon */}
            <div className="pl-2">
              <Github className={cn(
                "h-5 w-5 transition-colors",
                validationState === "valid" && "text-green-500",
                validationState === "invalid" && "text-red-500",
                validationState === "idle" && "text-muted-foreground"
              )} />
            </div>

            {/* Input Field */}
            <Input
              type="text"
              placeholder="Paste GitHub URL or user/repo..."
              value={inputValue}
              onChange={handleInputChange}
              className="flex-1 border-0 bg-transparent focus-visible:ring-0 focus-visible:ring-offset-0 text-lg placeholder:text-muted-foreground/60"
              disabled={isCreating}
            />

            {/* Paste Button */}
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={handlePasteClick}
              className="text-muted-foreground hover:text-foreground"
              disabled={isCreating}
            >
              <Clipboard className="h-4 w-4" />
            </Button>

            {/* Status Icon */}
            {validationState === "valid" && (
              <CheckCircle2 className="h-5 w-5 text-green-500" />
            )}
            {validationState === "invalid" && (
              <AlertCircle className="h-5 w-5 text-red-500" />
            )}
            {validationState === "loading" && (
              <Loader2 className="h-5 w-5 text-primary animate-spin" />
            )}

            {/* Submit Button */}
            <Button
              type="submit"
              size="lg"
              disabled={isSubmitDisabled}
              className={cn(
                "rounded-lg px-6 font-semibold transition-all",
                validationState === "valid" && "bg-gradient-to-r from-blue-600 to-cyan-600 hover:from-blue-700 hover:to-cyan-700"
              )}
            >
              {isCreating ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Creating...
                </>
              ) : (
                <>
                  <Rocket className="mr-2 h-4 w-4" />
                  GLive It!
                  <ArrowRight className="ml-2 h-4 w-4" />
                </>
              )}
            </Button>
          </div>
        </div>

        {/* Helper Text / Error Message */}
        <div className="flex items-center justify-between px-2 text-sm">
          <div className="flex items-center gap-2">
            {clipboardDetected && validationState === "valid" && (
              <Badge variant="secondary" className="text-xs">
                <Clipboard className="mr-1 h-3 w-3" />
                Pasted from clipboard
              </Badge>
            )}
            {errorMessage && (
              <span className="text-red-500">{errorMessage}</span>
            )}
            {!errorMessage && validationState === "idle" && (
              <span className="text-muted-foreground">
                Example: facebook/react or https://github.com/vercel/next.js
              </span>
            )}
          </div>

          {validationState === "valid" && !isCreating && (
            <span className="text-green-500 text-xs">
              Ready to clone with auto mode
            </span>
          )}
        </div>
      </form>

      {/* Features badges */}
      <div className="flex items-center justify-center gap-2 mt-4 flex-wrap">
        <Badge variant="outline" className="text-xs">
          Auto Dependencies
        </Badge>
        <Badge variant="outline" className="text-xs">
          AI Recovery
        </Badge>
        <Badge variant="outline" className="text-xs">
          Sandboxed
        </Badge>
        <Badge variant="outline" className="text-xs">
          One Click
        </Badge>
      </div>
    </div>
  );
}
