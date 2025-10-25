"use client"
import { useState } from 'react';
import { Upload, File, CheckCircle, AlertCircle, Download } from 'lucide-react';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export default function UploadPage() {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [uploadResult, setUploadResult] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      setSelectedFile(file);
      setUploadResult(null);
      setError(null);
    }
  };

  const handleUpload = async () => {
    if (!selectedFile) return;

    setUploading(true);
    setError(null);

    try {
      const formData = new FormData();
      formData.append('file', selectedFile);
      formData.append('user_id', 'demo-user');

      const response = await fetch(`${API_URL}/api/v1/files/upload`, {
        method: 'POST',
        body: formData,
      });

      if (!response.ok) {
        throw new Error(`Upload failed: ${response.statusText}`);
      }

      const result = await response.json();
      setUploadResult(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upload failed');
    } finally {
      setUploading(false);
    }
  };

  const handleDownload = async (fileId: string) => {
    try {
      const response = await fetch(`${API_URL}/api/v1/files/${fileId}/download`);
      
      if (!response.ok) {
        throw new Error('Download failed');
      }

      const result = await response.json();
      alert(`Download response: ${JSON.stringify(result, null, 2)}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Download failed');
    }
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 py-12 px-4">
      <div className="max-w-2xl mx-auto">
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-8">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-8 text-center">
            EchoFS File Upload Demo
          </h1>

          {}
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Select File
            </label>
            <div className="border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg p-6 text-center">
              <input
                type="file"
                onChange={handleFileSelect}
                className="hidden"
                id="file-upload"
              />
              <label
                htmlFor="file-upload"
                className="cursor-pointer flex flex-col items-center"
              >
                <Upload className="h-12 w-12 text-gray-400 mb-4" />
                <span className="text-sm text-gray-600 dark:text-gray-400">
                  Click to select a file or drag and drop
                </span>
              </label>
            </div>
          </div>

          {}
          {selectedFile && (
            <div className="mb-6 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
              <div className="flex items-center">
                <File className="h-5 w-5 text-blue-500 mr-2" />
                <div>
                  <p className="text-sm font-medium text-gray-900 dark:text-white">
                    {selectedFile.name}
                  </p>
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    {(selectedFile.size / 1024 / 1024).toFixed(2)} MB
                  </p>
                </div>
              </div>
            </div>
          )}

          {}
          <button
            onClick={handleUpload}
            disabled={!selectedFile || uploading}
            className="w-full bg-blue-600 text-white py-3 px-4 rounded-lg font-semibold disabled:opacity-50 disabled:cursor-not-allowed hover:bg-blue-700 transition-colors"
          >
            {uploading ? 'Uploading...' : 'Upload File'}
          </button>

          {}
          {error && (
            <div className="mt-4 p-4 bg-red-50 dark:bg-red-900/20 rounded-lg">
              <div className="flex items-center">
                <AlertCircle className="h-5 w-5 text-red-500 mr-2" />
                <p className="text-sm text-red-700 dark:text-red-400">{error}</p>
              </div>
            </div>
          )}

          {}
          {uploadResult && (
            <div className="mt-4 p-4 bg-green-50 dark:bg-green-900/20 rounded-lg">
              <div className="flex items-center mb-2">
                <CheckCircle className="h-5 w-5 text-green-500 mr-2" />
                <p className="text-sm font-medium text-green-700 dark:text-green-400">
                  Upload Successful!
                </p>
              </div>
              <div className="text-xs text-gray-600 dark:text-gray-400 space-y-1">
                <p><strong>File ID:</strong> {uploadResult.data?.file_id}</p>
                <p><strong>Session ID:</strong> {uploadResult.data?.session_id}</p>
                <p><strong>Chunks:</strong> {uploadResult.data?.chunks}</p>
                <p><strong>Compressed:</strong> {uploadResult.data?.compressed ? 'Yes' : 'No'}</p>
              </div>
              
              {uploadResult.data?.file_id && (
                <button
                  onClick={() => handleDownload(uploadResult.data.file_id)}
                  className="mt-3 flex items-center text-sm text-blue-600 hover:text-blue-800"
                >
                  <Download className="h-4 w-4 mr-1" />
                  Test Download
                </button>
              )}
            </div>
          )}

          {}
          <div className="mt-8 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
            <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
              Backend Status
            </h3>
            <p className="text-xs text-gray-600 dark:text-gray-400">
              Make sure your backend is running on http:
            </p>
            <code className="text-xs text-gray-500 dark:text-gray-400">
              ./run_master.sh
            </code>
          </div>
        </div>
      </div>
    </div>
  );
}