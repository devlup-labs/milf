import { AppLayout } from "@/components/layout";
import { PageHeader } from "@/components/shared";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";

export default function Settings() {
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
          <div className="bg-surface border border-border rounded-md p-6">
            <h3 className="text-sm font-medium mb-4">API Keys</h3>
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
