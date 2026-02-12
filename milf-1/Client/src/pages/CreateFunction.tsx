import { useState, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { Upload, FileCode, Container, FileUp, FolderUp, X, AlertTriangle, CheckCircle2, Maximize2, Minimize2, Settings, Loader2 } from "lucide-react";
import { Editor } from "@monaco-editor/react";
import { AppLayout } from "@/components/layout";
import { PageHeader, FileExplorer } from "@/components/shared";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Slider } from "@/components/ui/slider";
import { useToast } from "@/hooks/use-toast";
import { cn } from "@/lib/utils";
import { useCreateFunction } from "@/hooks/useQueries";
import { parseZipFile, parseDirectoryFiles } from "@/lib/mock/api";
import { FunctionEntity } from "@/lib/mock/types";

const sourceTypes = [
  { id: "inline", name: "Inline Code", icon: FileCode, description: "Write code directly" },
  { id: "zip", name: "Zip / Folder", icon: Upload, description: "Upload .zip or folder" },
  { id: "docker", name: "Docker Image", icon: Container, description: "Use a container" },
];

const runtimes = [
  { value: "go1.21", label: "Go 1.21" },
  { value: "node18", label: "Node.js 18" },
  { value: "node20", label: "Node.js 20" },
  { value: "python3.10", label: "Python 3.10" },
  { value: "python3.11", label: "Python 3.11" },
  { value: "java17", label: "Java 17" },
  { value: "dotnet6", label: ".NET 6" },
];

const STORAGE_KEY = "dark-canvas-create-function-draft";

export default function CreateFunction() {
  const navigate = useNavigate();
  const { toast } = useToast();
  const folderInputRef = useRef<HTMLInputElement>(null);
  const isEditorFullscreen = useRef(false);
  const [fullscreenState, setFullscreenState] = useState(false);

  const createFunction = useCreateFunction();

  // Form state
  // ... (previous state code)
  const [formData, setFormData] = useState({
    name: "",
    sourceType: "inline",
    code: `package main\n\nimport(\n    "context"\n    "encoding/json"\n) \n\nfunc Handler(ctx context.Context, event json.RawMessage)(interface{}, error) { \n    return map[string]string{ "message": "Hello, World!" }, nil\n } `,
    runtime: "go1.21",
    memory: 128,
    timeout: 30,
    envVars: [{ key: "", value: "" }],
    tags: "",
    file: null as File | null,
    files: null as FileList | null,
    status: "active" as "active" | "inactive",
  });

  const [validationStatus, setValidationStatus] = useState<{ valid: boolean, message: string } | null>(null);
  const [dragActive, setDragActive] = useState(false);

  // Persistence Logic
  useEffect(() => {
    // ... (previous persistence code)
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      try {
        const parsed = JSON.parse(saved);
        const { file, files, ...rest } = parsed;
        setFormData(prev => ({ ...prev, ...rest }));
      } catch (e) {
        console.error("Failed to restore draft", e);
      }
    }
  }, []);

  useEffect(() => {
    const { file, files, ...rest } = formData;
    localStorage.setItem(STORAGE_KEY, JSON.stringify(rest));
  }, [formData]);

  const updateFormData = (key: string, value: any) => {
    setFormData((prev) => ({ ...prev, [key]: value }));
  };

  // ... (drag and drop handlers)
  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true);
    } else if (e.type === 'dragleave') {
      setDragActive(false);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);

    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      handleFileSelection(e.dataTransfer.files);
    }
  };

  const handleFileSelection = (selectedFiles: FileList) => {
    setValidationStatus(null);
    if (selectedFiles.length === 1 && (selectedFiles[0].name.endsWith('.zip') || selectedFiles[0].type === 'application/zip')) {
      updateFormData("file", selectedFiles[0]);
      updateFormData("files", null);
    } else {
      updateFormData("files", selectedFiles);
      updateFormData("file", null);
    }
  };

  const handleFilesParsed = (fileNames: string[]) => {
    validateStructure(fileNames);
  };

  const validateStructure = (fileNames: string[]) => {
    let requiredFile = '';
    const r = formData.runtime;

    if (r.startsWith('node')) requiredFile = 'package.json';
    else if (r.startsWith('python')) requiredFile = 'requirements.txt';
    else if (r.startsWith('go')) requiredFile = 'go.mod';
    else if (r.startsWith('java')) requiredFile = 'pom.xml';
    else if (r.startsWith('dotnet')) requiredFile = '.csproj';

    if (!requiredFile) return;

    const found = fileNames.some(name => name.includes(requiredFile));
    if (found) {
      setValidationStatus({
        valid: true,
        message: `Found ${requiredFile} for ${r}`
      });
    } else {
      setValidationStatus({
        valid: false,
        message: `Missing ${requiredFile}. Required for ${r} dependencies.`
      });
    }
  };

  const isFormValid = () => {
    if (!formData.name) return false;
    if (formData.sourceType === "inline" && !formData.code) return false;
    if (formData.sourceType === "zip" && (!formData.file && !formData.files)) return false;
    return true;
  };

  const handleCreate = async (status: "active" | "inactive") => {
    try {
      let source: FunctionEntity["source"];

      if (formData.sourceType === "inline") {
        source = { type: "inline", code: formData.code };
      } else if (formData.sourceType === "zip") {
        if (formData.file) {
          // Zip file
          const parsed = await parseZipFile(formData.file);
          source = {
            type: "zip",
            fileName: formData.file.name,
            files: parsed.files,
            entryPath: parsed.entryPath
          };
        } else if (formData.files) {
          // Directory
          const parsed = await parseDirectoryFiles(formData.files);
          source = {
            type: "directory",
            rootName: parsed.rootName,
            files: parsed.files,
            entryPath: parsed.entryPath
          };
        } else {
          throw new Error("No files selected");
        }
      } else {
        source = { type: "inline", code: "// Docker not implemented yet" };
      }

      await createFunction.mutateAsync({
        name: formData.name,
        runtime: formData.runtime as any,
        status: status,
        memory: formData.memory,
        timeout: formData.timeout,
        tags: formData.tags.split(",").map(t => t.trim()).filter(Boolean),
        envVars: formData.envVars.filter(e => e.key),
        source: source,
      });

      localStorage.removeItem(STORAGE_KEY);
      toast({
        title: "Function created",
        description: `${formData.name} has been successfully created.`,
      });
      navigate("/functions");
    } catch (err) {
      toast({
        title: "Error creating function",
        description: (err as Error).message,
        variant: "destructive",
      });
    }
  };

  const estimatedCost = ((formData.memory / 1024) * (formData.timeout / 1000) * 0.0000166667).toFixed(6);

  const getEditorLanguage = () => {
    if (formData.runtime.startsWith("go")) return "go";
    if (formData.runtime.startsWith("node")) return "javascript";
    if (formData.runtime.startsWith("python")) return "python";
    if (formData.runtime.startsWith("java")) return "java";
    return "plaintext";
  };

  const toggleFullscreen = () => {
    isEditorFullscreen.current = !isEditorFullscreen.current;
    setFullscreenState(isEditorFullscreen.current);
  };

  return (
    <AppLayout>
      <PageHeader
        title="Create Function"
        description="Set up a new serverless function"
        actions={
          <div className="flex gap-2">
            <Button variant="ghost" onClick={() => navigate("/functions")}>
              Cancel
            </Button>
            <Button variant="outline" onClick={() => handleCreate('inactive')} disabled={!isFormValid() || createFunction.isPending}>
              {createFunction.isPending ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : null}
              Save Draft
            </Button>
            <Button onClick={() => handleCreate('active')} disabled={!isFormValid() || createFunction.isPending}>
              {createFunction.isPending ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : null}
              Deploy Function
            </Button>
          </div>
        }
      />

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-12">
        {/* Left Column: Main Configuration */}
        <div className="lg:col-span-2 space-y-6">
          {/* Basic Info */}
          <div className="bg-surface border border-border rounded-md p-6">
            <h3 className="text-lg font-medium mb-4 flex items-center gap-2">
              <Settings className="w-5 h-5 text-primary" />
              Function Details
            </h3>

            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="name">Function Name</Label>
                <Input
                  id="name"
                  placeholder="my-function"
                  value={formData.name}
                  onChange={(e) => updateFormData("name", e.target.value)}
                  className="font-mono"
                />
              </div>

              <div className="space-y-2">
                <Label>Runtime</Label>
                <Select
                  value={formData.runtime}
                  onValueChange={(value) => {
                    updateFormData("runtime", value);
                    setValidationStatus(null);
                  }}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {runtimes.map((runtime) => (
                      <SelectItem key={runtime.value} value={runtime.value}>
                        {runtime.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>

          {/* Source Code */}
          <div className="bg-surface border border-border rounded-md p-6">
            <h3 className="text-lg font-medium mb-4">Source Code</h3>

            <div className="space-y-4">
              <div className="grid grid-cols-3 gap-3">
                {sourceTypes.map((type) => (
                  <button
                    key={type.id}
                    type="button"
                    onClick={() => updateFormData("sourceType", type.id)}
                    className={cn(
                      "flex flex-col items-center gap-2 p-3 rounded-md border text-center micro-transition relative overflow-hidden",
                      formData.sourceType === type.id
                        ? "border-primary bg-primary/5 ring-1 ring-primary/20"
                        : "border-border hover:border-muted-foreground/50 hover:bg-muted/30"
                    )}
                  >
                    <type.icon className={cn("h-4 w-4", formData.sourceType === type.id ? "text-primary" : "text-muted-foreground")} />
                    <span className="text-xs font-medium">{type.name}</span>
                  </button>
                ))}
              </div>

              {formData.sourceType === "inline" && (
                <div className={cn("space-y-2 transition-all", fullscreenState ? "fixed inset-0 z-50 bg-background p-4 flex flex-col" : "")}>
                  <div className="flex items-center justify-between mb-2">
                    <Label htmlFor="code" className={fullscreenState ? "text-lg font-semibold" : ""}>Code Editor</Label>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={toggleFullscreen}
                      title={fullscreenState ? "Exit Fullscreen" : "Fullscreen"}
                    >
                      {fullscreenState ? <Minimize2 size={16} /> : <Maximize2 size={16} />}
                    </Button>
                  </div>
                  <div className={cn("border border-border/50 rounded-md overflow-hidden", fullscreenState ? "flex-1" : "h-[500px]")}>
                    <Editor
                      height="100%"
                      language={getEditorLanguage()}
                      value={formData.code}
                      onChange={(value) => updateFormData("code", value || "")}
                      theme="vs-dark"
                      options={{
                        minimap: { enabled: true },
                        fontSize: 14,
                        scrollBeyondLastLine: false,
                        mouseWheelZoom: true,
                        automaticLayout: true,
                        fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
                        padding: { top: 16 }
                      }}
                    />
                  </div>
                </div>
              )}

              {formData.sourceType === "zip" && (
                <div className="space-y-4">
                  <div className="border-2 border-dashed border-border rounded-lg p-6 flex flex-col items-center justify-center text-center hover:bg-muted/20 transition-colors"
                    onDragEnter={handleDrag}
                    onDragLeave={handleDrag}
                    onDragOver={handleDrag}
                    onDrop={handleDrop}
                  >
                    {/* ... Existing Upload Logic ... */}
                    {formData.file || formData.files ? (
                      <div className="w-full space-y-4">
                        <div className="bg-card p-3 rounded border border-border flex items-center justify-between">
                          <span className="text-sm truncate max-w-[150px]">
                            {formData.file ? formData.file.name : `${formData.files?.length} files`}
                          </span>
                          <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => {
                            updateFormData("file", null);
                            updateFormData("files", null);
                            setValidationStatus(null);
                          }}>
                            <X className="h-4 w-4" />
                          </Button>
                        </div>
                        {validationStatus && (
                          <div className={cn("text-xs p-2 rounded border flex items-start gap-2 text-left",
                            validationStatus.valid ? "bg-green-500/10 border-green-500/20 text-green-500" : "bg-yellow-500/10 border-yellow-500/20 text-yellow-500"
                          )}>
                            {validationStatus.valid ? <CheckCircle2 size={14} className="mt-0.5 shrink-0" /> : <AlertTriangle size={14} className="mt-0.5 shrink-0" />}
                            <span className="leading-tight">{validationStatus.message}</span>
                          </div>
                        )}
                      </div>
                    ) : (
                      <>
                        <Upload className="h-8 w-8 text-muted-foreground/50 mb-3" />
                        <div className="space-y-1 mb-4">
                          <p className="text-sm font-medium">Drag ZIP or Folder</p>
                          <p className="text-xs text-muted-foreground">Max 50MB</p>
                        </div>
                        <div className="flex gap-2">
                          <div className="relative">
                            <Button variant="outline" size="sm" className="gap-2">
                              <FileUp size={14} /> Select ZIP
                            </Button>
                            <input
                              type="file"
                              accept=".zip"
                              className="absolute inset-0 opacity-0 cursor-pointer"
                              onChange={(e) => {
                                if (e.target.files && e.target.files.length > 0) handleFileSelection(e.target.files);
                              }}
                            />
                          </div>
                          <div className="relative">
                            <Button variant="secondary" size="sm" className="gap-2">
                              <FolderUp size={14} /> Select Folder
                            </Button>
                            <input
                              type="file"
                              ref={folderInputRef}
                              {...({ webkitdirectory: "", directory: "" } as any)}
                              className="absolute inset-0 opacity-0 cursor-pointer"
                              onChange={(e) => {
                                if (e.target.files && e.target.files.length > 0) handleFileSelection(e.target.files);
                              }}
                            />
                          </div>
                        </div>
                      </>
                    )}
                  </div>

                  {(formData.file || formData.files) && (
                    <div className="h-[300px] bg-card border border-border/50 rounded-lg overflow-hidden">
                      <FileExplorer
                        file={formData.file}
                        files={formData.files}
                        onClose={() => { }} // No close from here
                        onFilesParsed={handleFilesParsed}
                        className="h-full border-none rounded-none"
                      />
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Right Column: Settings & Metadata */}
        <div className="space-y-6">
          <div className="bg-surface border border-border rounded-md p-6 sticky top-6">
            <h3 className="text-sm font-medium mb-4 text-muted-foreground uppercase tracking-wider">Resources</h3>

            <div className="space-y-6">
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <Label>Memory</Label>
                  <span className="text-sm font-mono">{formData.memory} MB</span>
                </div>
                <Slider
                  value={[formData.memory]}
                  onValueChange={([value]) => updateFormData("memory", value)}
                  min={128}
                  max={3008}
                  step={64}
                  className="w-full"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="timeout">Timeout (s)</Label>
                <Input
                  id="timeout"
                  type="number"
                  min={1}
                  max={900}
                  value={formData.timeout}
                  onChange={(e) => updateFormData("timeout", parseInt(e.target.value) || 30)}
                />
              </div>

              <div className="p-3 rounded bg-muted/30 border border-border/50">
                <p className="text-xs text-muted-foreground">Est. Cost/Invocation</p>
                <p className="text-lg font-mono text-foreground">${estimatedCost}</p>
              </div>
            </div>

            <div className="border-t border-border my-6" />

            <h3 className="text-sm font-medium mb-4 text-muted-foreground uppercase tracking-wider">Configuration</h3>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label>Environment Variables</Label>
                {formData.envVars.map((env, index) => (
                  <div key={index} className="flex gap-2">
                    <Input
                      placeholder="KEY"
                      value={env.key}
                      onChange={(e) => {
                        const newEnvVars = [...formData.envVars];
                        newEnvVars[index].key = e.target.value;
                        updateFormData("envVars", newEnvVars);
                      }}
                      className="flex-1 text-xs font-mono"
                    />
                    <Input
                      placeholder="VAL"
                      value={env.value}
                      onChange={(e) => {
                        const newEnvVars = [...formData.envVars];
                        newEnvVars[index].value = e.target.value;
                        updateFormData("envVars", newEnvVars);
                      }}
                      className="flex-1 text-xs"
                    />
                  </div>
                ))}
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="w-full text-xs"
                  onClick={() => updateFormData("envVars", [...formData.envVars, { key: "", value: "" }])}
                >
                  + Add Variable
                </Button>
              </div>

              <div className="space-y-2">
                <Label htmlFor="tags">Tags</Label>
                <Input
                  id="tags"
                  placeholder="production, api"
                  value={formData.tags}
                  onChange={(e) => updateFormData("tags", e.target.value)}
                  className="text-xs"
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </AppLayout>
  );
}
