import { useState, useCallback } from "react";
import { Upload, FolderOpen, File, MoreVertical, Download, Trash2, Eye, Search } from "lucide-react";
import { AppLayout } from "@/components/layout";
import { PageHeader, DataTable, EmptyState } from "@/components/shared";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Progress } from "@/components/ui/progress";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { cn } from "@/lib/utils";

// Mock data
const files = [
  { id: "1", name: "config.json", size: "2.4 KB", type: "application/json", modified: "2 hours ago", path: "/config" },
  { id: "2", name: "handler.go", size: "4.1 KB", type: "text/x-go", modified: "1 day ago", path: "/src" },
  { id: "3", name: "data.csv", size: "156 KB", type: "text/csv", modified: "3 days ago", path: "/data" },
  { id: "4", name: "model.pkl", size: "2.3 MB", type: "application/octet-stream", modified: "1 week ago", path: "/models" },
  { id: "5", name: "README.md", size: "1.2 KB", type: "text/markdown", modified: "2 weeks ago", path: "/" },
];

const filePreview = `{
  "version": "1.0.0",
  "environment": "production",
  "database": {
    "host": "db.example.com",
    "port": 5432,
    "ssl": true
  },
  "features": {
    "caching": true,
    "logging": "verbose"
  }
}`;

export default function Files() {
  const [searchQuery, setSearchQuery] = useState("");
  const [isDragging, setIsDragging] = useState(false);
  const [uploadProgress, setUploadProgress] = useState<number | null>(null);
  const [selectedFile, setSelectedFile] = useState<typeof files[0] | null>(null);

  const filteredFiles = files.filter((file) =>
    file.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
    
    // Simulate upload
    setUploadProgress(0);
    const interval = setInterval(() => {
      setUploadProgress((prev) => {
        if (prev === null || prev >= 100) {
          clearInterval(interval);
          setTimeout(() => setUploadProgress(null), 500);
          return 100;
        }
        return prev + 10;
      });
    }, 100);
  }, []);

  const getFileIcon = (type: string) => {
    if (type.includes("folder")) return FolderOpen;
    return File;
  };

  const columns = [
    {
      key: "name",
      header: "Name",
      render: (file: typeof files[0]) => {
        const Icon = getFileIcon(file.type);
        return (
          <button
            onClick={() => setSelectedFile(file)}
            className="flex items-center gap-2 text-foreground hover:text-primary"
          >
            <Icon className="h-4 w-4 text-muted-foreground" />
            <span className="font-mono text-sm">{file.name}</span>
          </button>
        );
      },
    },
    {
      key: "path",
      header: "Path",
      render: (file: typeof files[0]) => (
        <span className="text-muted-foreground font-mono text-xs">{file.path}</span>
      ),
    },
    {
      key: "size",
      header: "Size",
      className: "text-right",
      render: (file: typeof files[0]) => (
        <span className="text-muted-foreground">{file.size}</span>
      ),
    },
    {
      key: "modified",
      header: "Modified",
      className: "text-right",
      render: (file: typeof files[0]) => (
        <span className="text-muted-foreground">{file.modified}</span>
      ),
    },
    {
      key: "actions",
      header: "",
      className: "w-10",
      render: (file: typeof files[0]) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" className="h-7 w-7">
              <MoreVertical className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => setSelectedFile(file)}>
              <Eye className="h-3.5 w-3.5 mr-2" />
              Preview
            </DropdownMenuItem>
            <DropdownMenuItem>
              <Download className="h-3.5 w-3.5 mr-2" />
              Download
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="text-destructive">
              <Trash2 className="h-3.5 w-3.5 mr-2" />
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
    },
  ];

  return (
    <AppLayout>
      <PageHeader
        title="Files"
        description="Manage your uploaded files and assets"
        actions={
          <Button>
            <Upload className="h-4 w-4 mr-2" />
            Upload
          </Button>
        }
      />

      {/* Upload drop zone */}
      <div
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        className={cn(
          "border-2 border-dashed rounded-md p-8 mb-6 text-center micro-transition",
          isDragging ? "border-primary bg-primary/5" : "border-border",
          uploadProgress !== null && "pointer-events-none"
        )}
      >
        {uploadProgress !== null ? (
          <div className="space-y-2 max-w-xs mx-auto">
            <p className="text-sm text-muted-foreground">Uploading...</p>
            <Progress value={uploadProgress} className="h-1" />
          </div>
        ) : (
          <>
            <Upload className="h-8 w-8 text-muted-foreground mx-auto mb-2" />
            <p className="text-sm text-muted-foreground">
              Drag and drop files here, or click to browse
            </p>
          </>
        )}
      </div>

      {/* Search */}
      <div className="flex items-center gap-4 mb-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
          <Input
            type="search"
            placeholder="Search files..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-8 h-8 bg-background"
          />
        </div>
      </div>

      {/* Files Table */}
      <div className="bg-surface border border-border rounded-md">
        {filteredFiles.length === 0 && searchQuery === "" ? (
          <EmptyState
            icon={<FolderOpen className="h-6 w-6 text-muted-foreground" />}
            title="No files yet"
            description="Upload your first file to get started"
            action={<Button>Upload File</Button>}
          />
        ) : (
          <DataTable
            columns={columns}
            data={filteredFiles}
            emptyMessage="No files match your search"
          />
        )}
      </div>

      {/* File Preview Sheet */}
      <Sheet open={!!selectedFile} onOpenChange={() => setSelectedFile(null)}>
        <SheetContent className="sm:max-w-lg">
          <SheetHeader>
            <SheetTitle className="font-mono">{selectedFile?.name}</SheetTitle>
          </SheetHeader>
          <div className="mt-6 space-y-4">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <p className="text-muted-foreground">Size</p>
                <p className="mt-1">{selectedFile?.size}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Modified</p>
                <p className="mt-1">{selectedFile?.modified}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Type</p>
                <p className="mt-1 font-mono text-xs">{selectedFile?.type}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Path</p>
                <p className="mt-1 font-mono text-xs">{selectedFile?.path}</p>
              </div>
            </div>

            {selectedFile?.name.endsWith(".json") && (
              <div className="mt-6">
                <p className="text-sm font-medium mb-2">Preview</p>
                <pre className="bg-terminal border border-border rounded-md p-4 overflow-auto text-xs font-mono">
                  {filePreview}
                </pre>
              </div>
            )}
          </div>
        </SheetContent>
      </Sheet>
    </AppLayout>
  );
}
