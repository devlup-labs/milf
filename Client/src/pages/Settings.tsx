import { useState, useEffect } from "react";
import { AppLayout } from "@/components/layout";
import { PageHeader } from "@/components/shared";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { useToast } from "@/hooks/use-toast";
import { useTestCopilotKey } from "@/hooks/useQueries";
import { Sparkles, Trash2, Loader2, Check } from "lucide-react";

export default function Settings() {
  const { toast } = useToast();
  const [apiKey, setApiKey] = useState(localStorage.getItem('milf_user_gemini_key') || '');
  const [keyInput, setKeyInput] = useState('');
  const [isConnecting, setIsConnecting] = useState(false);
  
  const testKeyMutation = useTestCopilotKey();

  const handleConnectKey = async () => {
    const trimmedKey = keyInput.trim();
    if (!trimmedKey) return;

    setIsConnecting(true);
    try {
      await testKeyMutation.mutateAsync(trimmedKey);
      localStorage.setItem('milf_user_gemini_key', trimmedKey);
      setApiKey(trimmedKey);
      setKeyInput('');
      toast({
        title: "Gemini Connected",
        description: "Your custom Gemini API Key has been configured successfully!",
      });
    } catch (err) {
      toast({
        title: "Connection Failed",
        description: (err as Error).message || "Could not validate your Gemini Key. Please check the key and try again.",
        variant: "destructive"
      });
    } finally {
      setIsConnecting(false);
    }
  };

  const handleDisconnectKey = () => {
    localStorage.removeItem('milf_user_gemini_key');
    setApiKey('');
    toast({
      title: "Key Disconnected",
      description: "Your custom Gemini API Key has been removed.",
    });
  };

  return (
    <AppLayout>
      <PageHeader
        title="Settings"
        description="Manage your account and project settings"
      />

      <Tabs defaultValue="general" className="space-y-6">
        <TabsList className="bg-transparent border-b border-border rounded-none w-full justify-start gap-0 p-0 h-auto">
          {["General", "API Keys", "Notifications", "Team"].map((tab) => (
            <TabsTrigger
              key={tab}
              value={tab.toLowerCase().replace(" ", "-")}
              className={cn(
                "px-4 py-2 rounded-none border-b-2 border-transparent",
                "data-[state=active]:border-primary data-[state=active]:bg-transparent",
                "text-muted-foreground data-[state=active]:text-foreground"
              )}
            >
              {tab}
            </TabsTrigger>
          ))}
        </TabsList>

        {/* General Tab */}
        <TabsContent value="general">
          <div className="bg-surface border border-border rounded-md">
            <div className="p-6 space-y-6">
              <div>
                <h3 className="text-sm font-medium mb-4">Project Information</h3>
                <div className="grid gap-4 max-w-md">
                  <div className="space-y-2">
                    <Label htmlFor="project-name">Project Name</Label>
                    <Input
                      id="project-name"
                      defaultValue="my-project"
                      className="bg-background"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="project-id">Project ID</Label>
                    <Input
                      id="project-id"
                      value="prj_abc123def456"
                      disabled
                      className="bg-background font-mono text-sm"
                    />
                  </div>
                </div>
              </div>

              <Separator />

              <div>
                <h3 className="text-sm font-medium mb-4">Default Settings</h3>
                <div className="space-y-4">
                  <div className="flex items-center justify-between max-w-md">
                    <div>
                      <p className="text-sm font-medium">Auto-deploy on push</p>
                      <p className="text-xs text-muted-foreground">
                        Automatically deploy when code is pushed
                      </p>
                    </div>
                    <Switch defaultChecked />
                  </div>
                  <div className="flex items-center justify-between max-w-md">
                    <div>
                      <p className="text-sm font-medium">Log retention</p>
                      <p className="text-xs text-muted-foreground">
                        Keep logs for 30 days
                      </p>
                    </div>
                    <Switch defaultChecked />
                  </div>
                </div>
              </div>

              <Separator />

              <div className="flex justify-end">
                <Button>Save Changes</Button>
              </div>
            </div>
          </div>
        </TabsContent>

        {/* API Keys Tab */}
        <TabsContent value="api-keys">
          <div className="space-y-6">
            <div className="bg-surface border border-border rounded-md p-6">
              <h3 className="text-sm font-medium mb-4">Project API Keys</h3>
              <div className="space-y-4">
                <div className="flex items-center justify-between p-3 bg-background rounded-md border border-border">
                  <div>
                    <p className="text-sm font-medium">Production Key</p>
                    <p className="text-xs text-muted-foreground font-mono">sk_live_***************abc</p>
                  </div>
                  <Button variant="secondary" size="sm">Regenerate</Button>
                </div>
                <div className="flex items-center justify-between p-3 bg-background rounded-md border border-border">
                  <div>
                    <p className="text-sm font-medium">Development Key</p>
                    <p className="text-xs text-muted-foreground font-mono">sk_test_***************xyz</p>
                  </div>
                  <Button variant="secondary" size="sm">Regenerate</Button>
                </div>
              </div>
            </div>

            <div className="bg-surface border border-border rounded-md p-6">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <h3 className="text-sm font-medium">Gemini AI Copilot Configuration</h3>
                  <p className="text-xs text-muted-foreground mt-1">
                    Connect your own Gemini API Key to enable the AI Copilot for code generation and refactoring.
                  </p>
                </div>
                <span className="text-xs text-muted-foreground bg-primary/10 text-primary px-2 py-1 rounded font-semibold flex items-center gap-1">
                  <Sparkles className="h-3 w-3" />
                  BYOK Enabled
                </span>
              </div>

              <div className="p-4 bg-background rounded-md border border-border/80 space-y-4">
                {apiKey ? (
                  <div className="flex items-center justify-between">
                    <div>
                      <span className="text-xs text-green-500 font-semibold uppercase tracking-wider block mb-1">Status: Connected</span>
                      <p className="text-sm font-medium font-mono text-foreground/90">
                        {apiKey.substring(0, 6)}...{apiKey.substring(apiKey.length - 4)}
                      </p>
                    </div>
                    <div className="flex gap-2">
                      <Button
                        type="button"
                        variant="destructive"
                        onClick={handleDisconnectKey}
                        className="h-8 text-xs gap-1.5"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                        Disconnect Key
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="space-y-4">
                    <div className="space-y-1">
                      <p className="text-sm font-medium text-foreground/90">No API key connected</p>
                      <p className="text-xs text-muted-foreground leading-relaxed max-w-md">
                        Get your API key free from{" "}
                        <a
                          href="https://aistudio.google.com/app/apikey"
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-primary hover:underline font-semibold"
                        >
                          Google AI Studio &rarr;
                        </a>{" "}
                        then paste it below to connect.
                      </p>
                    </div>
                    <div className="flex flex-col sm:flex-row gap-2 max-w-md">
                      <Input
                        type="password"
                        placeholder="Paste your Gemini API Key here..."
                        value={keyInput}
                        onChange={(e) => setKeyInput(e.target.value)}
                        className="bg-background text-sm font-mono"
                        disabled={isConnecting}
                      />
                      <Button
                        type="button"
                        onClick={handleConnectKey}
                        disabled={isConnecting || !keyInput.trim()}
                        className="bg-blue-600 hover:bg-blue-700 text-white font-medium gap-1.5 shrink-0"
                      >
                        {isConnecting ? (
                          <>
                            <Loader2 className="h-4 w-4 animate-spin" />
                            Connecting...
                          </>
                        ) : (
                          <>
                            <Sparkles className="h-4 w-4" />
                            Connect Key
                          </>
                        )}
                      </Button>
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        </TabsContent>

        {/* Notifications Tab */}
        <TabsContent value="notifications">
          <div className="bg-surface border border-border rounded-md p-6">
            <h3 className="text-sm font-medium mb-4">Notification Preferences</h3>
            <div className="space-y-4 max-w-md">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium">Error alerts</p>
                  <p className="text-xs text-muted-foreground">Get notified on function errors</p>
                </div>
                <Switch defaultChecked />
              </div>
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium">Weekly digest</p>
                  <p className="text-xs text-muted-foreground">Summary of usage and costs</p>
                </div>
                <Switch defaultChecked />
              </div>
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium">Deployment notifications</p>
                  <p className="text-xs text-muted-foreground">Get notified on deployments</p>
                </div>
                <Switch />
              </div>
            </div>
          </div>
        </TabsContent>

        {/* Team Tab */}
        <TabsContent value="team">
          <div className="bg-surface border border-border rounded-md p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-medium">Team Members</h3>
              <Button size="sm">Invite Member</Button>
            </div>
            <div className="space-y-2">
              {[
                { name: "John Doe", email: "john@example.com", role: "Owner" },
                { name: "Jane Smith", email: "jane@example.com", role: "Admin" },
                { name: "Bob Wilson", email: "bob@example.com", role: "Developer" },
              ].map((member) => (
                <div
                  key={member.email}
                  className="flex items-center justify-between p-3 bg-background rounded-md border border-border"
                >
                  <div className="flex items-center gap-3">
                    <div className="h-8 w-8 rounded-full bg-primary/20 flex items-center justify-center">
                      <span className="text-xs font-medium text-primary">
                        {member.name.split(" ").map((n) => n[0]).join("")}
                      </span>
                    </div>
                    <div>
                      <p className="text-sm font-medium">{member.name}</p>
                      <p className="text-xs text-muted-foreground">{member.email}</p>
                    </div>
                  </div>
                  <span className="text-xs text-muted-foreground">{member.role}</span>
                </div>
              ))}
            </div>
          </div>
        </TabsContent>
      </Tabs>
    </AppLayout>
  );
}
