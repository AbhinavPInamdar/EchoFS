"use client"
import { useState, useEffect } from 'react';
import { File, Download, Eye, Calendar, HardDrive } from 'lucide-react';

interface FileItem {
    file_id: string;
    name: string;
    size: number;
    uploaded: string;
    type: string;
}

export default function FilesPage() {
    const [files, setFiles] = useState<FileItem[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        fetchFiles();
    }, []);

    const fetchFiles = async () => {
        try {
            const response = await fetch('http://localhost:8080/api/v1/files');
            if (!response.ok) {
                throw new Error('Failed to fetch files');
            }
            const result = await response.json();
            setFiles(result.data || []);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load files');
        } finally {
            setLoading(false);
        }
    };

    const handleDownload = async (fileId: string, fileName: string) => {
        try {
            const response = await fetch(`http://localhost:8080/api/v1/files/${fileId}/download`);
            if (!response.ok) {
                throw new Error('Download failed');
            }

            const blob = await response.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = fileName;
            document.body.appendChild(a);
            a.click();
            window.URL.revokeObjectURL(url);
            document.body.removeChild(a);
        } catch (err) {
            alert('Download failed: ' + (err instanceof Error ? err.message : 'Unknown error'));
        }
    };

    const formatFileSize = (bytes: number) => {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    };

    const formatDate = (dateString: string) => {
        return new Date(dateString).toLocaleString();
    };

    const getFileIcon = (type: string) => {
        switch (type.toLowerCase()) {
            case '.pdf':
                return '📄';
            case '.xlsx':
            case '.xls':
                return '📊';
            case '.docx':
            case '.doc':
                return '📝';
            case '.jpg':
            case '.jpeg':
            case '.png':
            case '.gif':
                return '🖼️';
            default:
                return '📁';
        }
    };

    if (loading) {
        return (
            <div className="min-h-screen bg-gray-50 dark:bg-gray-900 py-12 px-4">
                <div className="max-w-4xl mx-auto">
                    <div className="text-center">
                        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
                        <p className="mt-4 text-gray-600 dark:text-gray-400">Loading files...</p>
                    </div>
                </div>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-gray-50 dark:bg-gray-900 py-12 px-4">
            <div className="max-w-4xl mx-auto">
                <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-8">
                    <div className="flex items-center justify-between mb-8">
                        <h1 className="text-3xl font-bold text-gray-900 dark:text-white flex items-center">
                            <HardDrive className="mr-3" />
                            Uploaded Files
                        </h1>
                        <button
                            onClick={fetchFiles}
                            className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors"
                        >
                            Refresh
                        </button>
                    </div>

                    {error && (
                        <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/20 rounded-lg">
                            <p className="text-red-700 dark:text-red-400">{error}</p>
                        </div>
                    )}

                    {files.length === 0 ? (
                        <div className="text-center py-12">
                            <File className="h-16 w-16 text-gray-400 mx-auto mb-4" />
                            <p className="text-gray-600 dark:text-gray-400 text-lg">No files uploaded yet</p>
                            <p className="text-gray-500 dark:text-gray-500 text-sm mt-2">
                                Go to the Upload Demo to add some files
                            </p>
                        </div>
                    ) : (
                        <div className="space-y-4">
                            {files.map((file) => (
                                <div
                                    key={file.file_id}
                                    className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
                                >
                                    <div className="flex items-center justify-between">
                                        <div className="flex items-center space-x-4">
                                            <div className="text-2xl">{getFileIcon(file.type)}</div>
                                            <div>
                                                <h3 className="font-semibold text-gray-900 dark:text-white">
                                                    {file.name}
                                                </h3>
                                                <div className="flex items-center space-x-4 text-sm text-gray-500 dark:text-gray-400">
                                                    <span>{formatFileSize(file.size)}</span>
                                                    <span className="flex items-center">
                                                        <Calendar className="h-4 w-4 mr-1" />
                                                        {formatDate(file.uploaded)}
                                                    </span>
                                                    <span className="font-mono text-xs bg-gray-100 dark:bg-gray-600 px-2 py-1 rounded">
                                                        ID: {file.file_id.substring(0, 8)}...
                                                    </span>
                                                </div>
                                            </div>
                                        </div>
                                        <div className="flex space-x-2">
                                            <button
                                                onClick={() => handleDownload(file.file_id, file.name)}
                                                className="flex items-center space-x-1 bg-green-600 text-white px-3 py-2 rounded-lg hover:bg-green-700 transition-colors"
                                            >
                                                <Download className="h-4 w-4" />
                                                <span>Download</span>
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}

                    <div className="mt-8 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                        <h3 className="text-sm font-medium text-blue-900 dark:text-blue-100 mb-2">
                            File Storage Info
                        </h3>
                        <p className="text-xs text-blue-700 dark:text-blue-300">
                            Files are stored in: <code>./storage/uploads/[file_id]/</code>
                        </p>
                        <p className="text-xs text-blue-700 dark:text-blue-300 mt-1">
                            Each file is compressed and chunked for distributed storage
                        </p>
                    </div>
                </div>
            </div>
        </div>
    );
}