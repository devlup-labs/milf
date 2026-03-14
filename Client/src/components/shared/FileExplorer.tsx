
import React, { useState, useEffect } from 'react';
import JSZip from 'jszip';
import { Editor } from '@monaco-editor/react';
import {
    File,
    FileJson,
    FileCode,
    FileText,
    ChevronRight,
    ChevronDown,
    LayoutList,
    LayoutGrid,
    Search,
    X,
    AlertCircle,
    Loader2
} from 'lucide-react';
import { cn } from "@/lib/utils";

interface FileExplorerProps {
    file: File | null;
    files?: FileList | null;
    onClose: () => void;
    onFilesParsed?: (fileNames: string[]) => void;
    className?: string; // Support custom styling
}

interface FileNode {
    name: string;
    path: string;
    isDir: boolean;
    children?: FileNode[];
    contentPromise?: () => Promise<string | ArrayBuffer>;
    size?: number;
}

const FileExplorer: React.FC<FileExplorerProps> = ({ file, files, onClose, onFilesParsed, className }) => {
    const [createError, setCreateError] = useState<string>('');
    const [fileStructure, setFileStructure] = useState<FileNode[]>([]);
    const [flatStructure, setFlatStructure] = useState<FileNode[]>([]);
    const [selectedFile, setSelectedFile] = useState<FileNode | null>(null);
    const [fileContent, setFileContent] = useState<string>('');
    const [viewMode, setViewMode] = useState<'tree' | 'flat'>('tree');
    const [loading, setLoading] = useState(false);
    const [expandedFolders, setExpandedFolders] = useState<Set<string>>(new Set());
    const [language, setLanguage] = useState('plaintext');

    useEffect(() => {
        if (file) {
            processZipFile(file);
        } else if (files && files.length > 0) {
            processFileList(files);
        }
    }, [file, files]);

    const processZipFile = async (zipFile: File) => {
        setLoading(true);
        try {
            const zip = new JSZip();
            const contents = await zip.loadAsync(zipFile);

            const flatNodes: FileNode[] = [];

            // Helper to get logic for content
            const getContent = (zipEntry: any) => async () => {
                return await zipEntry.async('string');
            };

            contents.forEach((relativePath, zipEntry) => {
                const isDir = zipEntry.dir || relativePath.endsWith('/');
                const name = relativePath.split('/').filter(p => p).pop() || relativePath;

                const node: FileNode = {
                    name,
                    path: relativePath,
                    isDir,
                    contentPromise: !isDir ? getContent(zipEntry) : undefined,
                    size: (zipEntry as any)._data?.uncompressedSize || 0
                };

                if (!isDir) flatNodes.push(node);
            });

            setFlatStructure(flatNodes);
            setFileStructure(buildTree(flatNodes));

            if (onFilesParsed) {
                onFilesParsed(flatNodes.map(n => n.name));
            }
        } catch (err) {
            setCreateError('Failed to parse ZIP file');
            console.error(err);
        } finally {
            setLoading(false);
        }
    };

    const processFileList = (fileList: FileList) => {
        setLoading(true);
        const flatNodes: FileNode[] = [];

        Array.from(fileList).forEach(f => {
            const relativePath = f.webkitRelativePath || f.name;
            const pathParts = relativePath.split('/');
            const name = pathParts[pathParts.length - 1];

            const node: FileNode = {
                name,
                path: relativePath,
                isDir: false,
                size: f.size,
                contentPromise: async () => {
                    return await f.text();
                }
            };
            flatNodes.push(node);
        });

        setFlatStructure(flatNodes);
        setFileStructure(buildTree(flatNodes));
        setLoading(false);

        if (onFilesParsed) {
            onFilesParsed(flatNodes.map(n => n.name));
        }
    };

    const buildTree = (flatFiles: FileNode[]) => {
        const root: FileNode[] = [];

        flatFiles.forEach(fileNode => {
            const parts = fileNode.path.split('/').filter(p => p);
            let currentLevel = root;
            let currentPath = '';

            parts.forEach((part, index) => {
                currentPath = currentPath ? `${currentPath}/${part}` : part;
                const isFile = index === parts.length - 1;

                let existingNode = currentLevel.find(n => n.name === part);

                if (existingNode) {
                    if (existingNode.isDir && !existingNode.children) {
                        existingNode.children = [];
                    }
                    currentLevel = existingNode.children!;
                } else {
                    const newNode: FileNode = {
                        name: part,
                        path: currentPath,
                        isDir: !isFile,
                        children: !isFile ? [] : undefined,
                        contentPromise: isFile ? fileNode.contentPromise : undefined,
                        size: isFile ? fileNode.size : 0
                    };
                    currentLevel.push(newNode);
                    if (!isFile) {
                        currentLevel = newNode.children!;
                    }
                }
            });
        });

        const sortNodes = (nodes: FileNode[]) => {
            nodes.sort((a, b) => {
                if (a.isDir === b.isDir) return a.name.localeCompare(b.name);
                return a.isDir ? -1 : 1;
            });
            nodes.forEach(n => {
                if (n.children) sortNodes(n.children);
            });
        };
        sortNodes(root);

        return root;
    };

    const handleFileClick = async (node: FileNode) => {
        if (node.isDir) {
            toggleFolder(node.path);
            return;
        }

        setSelectedFile(node);
        const ext = node.name.split('.').pop()?.toLowerCase();
        switch (ext) {
            case 'js': setLanguage('javascript'); break;
            case 'jsx': setLanguage('javascript'); break;
            case 'ts': setLanguage('typescript'); break;
            case 'tsx': setLanguage('typescript'); break;
            case 'py': setLanguage('python'); break;
            case 'html': setLanguage('html'); break;
            case 'css': setLanguage('css'); break;
            case 'json': setLanguage('json'); break;
            case 'go': setLanguage('go'); break;
            case 'java': setLanguage('java'); break;
            default: setLanguage('plaintext');
        }

        if (node.contentPromise) {
            try {
                const content = await node.contentPromise();
                setFileContent(typeof content === 'string' ? content : 'Binary file');
            } catch (e) {
                setFileContent('Error reading file');
            }
        }
    };

    const toggleFolder = (path: string) => {
        const newExpanded = new Set(expandedFolders);
        if (newExpanded.has(path)) {
            newExpanded.delete(path);
        } else {
            newExpanded.add(path);
        }
        setExpandedFolders(newExpanded);
    };

    const renderTree = (nodes: FileNode[], level = 0) => {
        return nodes.map(node => (
            <div key={node.path}>
                <div
                    className={cn(
                        "flex items-center py-1 px-2 cursor-pointer hover:bg-muted/50 text-sm transition-colors rounded-sm mx-1",
                        selectedFile?.path === node.path ? 'bg-primary/20 text-primary' : 'text-muted-foreground'
                    )}
                    style={{ paddingLeft: `${level * 12 + 8}px` }}
                    onClick={() => handleFileClick(node)}
                >
                    <span className="mr-1.5 opacity-70">
                        {node.isDir ? (
                            expandedFolders.has(node.path) ? <ChevronDown size={14} /> : <ChevronRight size={14} />
                        ) : (
                            getFileIcon(node.name)
                        )}
                    </span>

                    <span className="truncate">{node.name}</span>
                </div>
                {node.isDir && expandedFolders.has(node.path) && node.children && (
                    <div>
                        {renderTree(node.children, level + 1)}
                    </div>
                )}
            </div>
        ));
    };

    const renderFlat = () => {
        return flatStructure.map(node => (
            <div
                key={node.path}
                className={cn(
                    "flex items-center py-1 px-2 cursor-pointer hover:bg-muted/50 text-sm transition-colors",
                    selectedFile?.path === node.path ? 'bg-primary/20 text-primary' : 'text-muted-foreground'
                )}
                onClick={() => handleFileClick(node)}
            >
                <span className="mr-2 opacity-70">{getFileIcon(node.name)}</span>
                <span className="truncate">{node.path}</span>
            </div>
        ));
    };

    const getFileIcon = (filename: string) => {
        if (filename.endsWith('.js') || filename.endsWith('.ts') || filename.endsWith('.py')) return <FileCode size={14} />;
        if (filename.endsWith('.json')) return <FileJson size={14} />;
        if (filename.endsWith('.txt') || filename.endsWith('.md')) return <FileText size={14} />;
        return <File size={14} />;
    };

    return (
        <div className={cn("flex bg-black/20 border border-border/50 rounded-lg overflow-hidden h-[450px] shadow-sm", className)}>
            {/* Sidebar */}
            <div className="w-64 bg-card/30 border-r border-border/50 flex flex-col backdrop-blur-sm">
                <div className="p-3 border-b border-border/50 flex items-center justify-between">
                    <span className="font-semibold text-xs tracking-wider uppercase text-muted-foreground">Explorer</span>
                    <div className="flex gap-1">
                        <button
                            onClick={() => setViewMode('tree')}
                            className={cn("p-1 rounded hover:bg-muted transition-colors", viewMode === 'tree' ? 'text-primary bg-muted/50' : 'text-muted-foreground')}
                            title="Tree View"
                        >
                            <LayoutList size={14} />
                        </button>
                        <button
                            onClick={() => setViewMode('flat')}
                            className={cn("p-1 rounded hover:bg-muted transition-colors", viewMode === 'flat' ? 'text-primary bg-muted/50' : 'text-muted-foreground')}
                            title="Flat View"
                        >
                            <LayoutGrid size={14} />
                        </button>
                    </div>
                </div>

                <div className="flex-1 overflow-y-auto py-2 custom-scrollbar">
                    {createError && (
                        <div className="px-3 py-2 text-xs text-destructive bg-destructive/10 flex items-center gap-2 mb-2 mx-2 rounded">
                            <AlertCircle size={12} />
                            {createError}
                        </div>
                    )}
                    {loading ? (
                        <div className="flex flex-col items-center justify-center h-40 text-muted-foreground text-sm gap-2">
                            <Loader2 className="animate-spin" size={16} />
                            <span>Parsing files...</span>
                        </div>
                    ) : (
                        viewMode === 'tree' ? renderTree(fileStructure) : renderFlat()
                    )}
                </div>
            </div>

            {/* Main Area */}
            <div className="flex-1 flex flex-col bg-background/50 backdrop-blur-sm relative">
                {selectedFile ? (
                    <>
                        <div className="h-9 border-b border-border/50 flex items-center px-4 bg-muted/10 justify-between">
                            <div className="flex items-center gap-2 text-sm text-muted-foreground">
                                {getFileIcon(selectedFile.name)}
                                <span className="font-mono text-xs">{selectedFile.path}</span>
                            </div>
                            <span className="text-xs text-muted-foreground">{selectedFile.size ? (selectedFile.size / 1024).toFixed(1) + ' KB' : ''}</span>
                        </div>
                        <div className="flex-1 relative">
                            <Editor
                                height="100%"
                                language={language}
                                value={fileContent}
                                theme="vs-dark"
                                options={{
                                    readOnly: true,
                                    minimap: { enabled: false },
                                    scrollBeyondLastLine: false,
                                    fontSize: 13,
                                    fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
                                    padding: { top: 16 }
                                }}
                            />
                        </div>
                    </>
                ) : (
                    <div className="flex-1 flex flex-col items-center justify-center text-muted-foreground opacity-30 gap-4">
                        <div className="p-4 rounded-full bg-muted/20">
                            <Search size={32} />
                        </div>
                        <p className="text-sm">Select a file to preview content</p>
                    </div>
                )}

                <button
                    onClick={onClose}
                    className="absolute top-3 right-3 p-1.5 bg-background/80 text-muted-foreground hover:text-foreground rounded-full hover:bg-muted transition-colors border border-border/50 z-50 shadow-sm"
                    title="Close Explorer"
                >
                    <X size={14} />
                </button>
            </div>
        </div>
    );
};

export default FileExplorer;
