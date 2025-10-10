"use client"
import { useState, useEffect } from 'react';
import {
  Home as HomeIcon,
  Layout as LayoutIcon,
  Shield as ShieldIcon,
  Lock as LockIcon,
  Zap as ZapIcon,
  ArrowUp as ArrowUpIcon,
  ArrowDown as ArrowDownIcon,
  Share2 as ShareIcon,
  FileText as FileTextIcon,
  Sparkles as SparklesIcon,
  BarChart3 as BarChart3Icon,
  Activity as ActivityIcon,
  Users as UsersIcon,
  HardDrive as HardDriveIcon,
  Clock as ClockIcon,
  TrendingUp as TrendingUpIcon
} from 'lucide-react';


function App() {
  const [page, setPage] = useState('home');

  return (
    <>
      <style>
        {/*
          This CSS is included directly to make the app self-contained.
          It includes the Inter font and a simple light/dark theme.
          The colors have been adjusted for better contrast.
        */}
        {`
        @import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;600;700&display=swap');
        
        :root {
          --background: #f8f9fa;
          --foreground: #212529;
          --blue-primary: #1e40af; /* A darker, more accessible blue */
          --blue-light: #dbeafe; /* A lighter, more accessible blue */
          --shadow-color: rgba(0, 0, 0, 0.1);
        }

        @media (prefers-color-scheme: dark) {
          :root {
            --background: #1a202c;
            --foreground: #e2e8f0;
            --blue-primary: #3b82f6; /* A brighter blue for contrast on dark bg */
            --blue-light: #1e40af;
            --shadow-color: rgba(255, 255, 255, 0.1);
          }
        }

        body {
          font-family: 'Inter', sans-serif;
          background-color: var(--background);
          color: var(--foreground);
        }
        
        .shadow-md {
          box-shadow: 0 4px 6px -1px var(--shadow-color), 0 2px 4px -1px var(--shadow-color);
        }
        
        /* Correcting the visibility issues on the light background */
        .text-gray-600 {
            color: #4a5568;
        }

        .dark .text-gray-900 {
            color: var(--foreground);
        }
        .dark .text-gray-600 {
            color: #d1d5db; /* Lighter gray for better visibility in dark mode */
        }
        `}
      </style>
      <header className="bg-white dark:bg-gray-800 shadow-md">
        <nav className="flex items-center justify-between p-6 max-w-7xl mx-auto">
          <div className="flex items-center space-x-4">
            <LayoutIcon className="text-blue-primary" />
            <h1 className="text-xl font-bold text-gray-900 dark:text-white">EchoFS</h1>
          </div>
          <div className="flex space-x-6">
            <button
              onClick={() => setPage('home')}
              className={`font-semibold transition-colors duration-200 hover:text-blue-primary ${page === 'home' ? 'text-blue-primary' : 'text-gray-600 dark:text-gray-300'}`}
            >
              Home
            </button>
            <button
              onClick={() => setPage('upload')}
              className={`font-semibold transition-colors duration-200 hover:text-blue-primary ${page === 'upload' ? 'text-blue-primary' : 'text-gray-600 dark:text-gray-300'}`}
            >
              Upload Demo
            </button>
            <button
              onClick={() => setPage('files')}
              className={`font-semibold transition-colors duration-200 hover:text-blue-primary ${page === 'files' ? 'text-blue-primary' : 'text-gray-600 dark:text-gray-300'}`}
            >
              My Files
            </button>
            <button
              onClick={() => setPage('hld')}
              className={`font-semibold transition-colors duration-200 hover:text-blue-primary ${page === 'hld' ? 'text-blue-primary' : 'text-gray-600 dark:text-gray-300'}`}
            >
              High-Level Design
            </button>
            <button
              onClick={() => setPage('file-manager')}
              className={`font-semibold transition-colors duration-200 hover:text-blue-primary ${page === 'file-manager' ? 'text-blue-primary' : 'text-gray-600 dark:text-gray-300'}`}
            >
              File Manager
            </button>
            <button
              onClick={() => setPage('metrics')}
              className={`font-semibold transition-colors duration-200 hover:text-blue-primary ${page === 'metrics' ? 'text-blue-primary' : 'text-gray-600 dark:text-gray-300'}`}
            >
              Metrics
            </button>
          </div>
        </nav>
      </header>

      <main>
        {page === 'home' && <HomePage />}
        {page === 'upload' && <UploadDemoPage />}
        {page === 'files' && <FilesPage />}
        {page === 'hld' && <HighLevelDesignPage />}
        {page === 'file-manager' && <FileManagementPage />}
        {page === 'metrics' && <MetricsPage />}
      </main>
    </>
  );
}

const HomePage = () => (
  <>
    <HeroComponent />
    <FeaturesComponent />
    <FooterComponent />
  </>
);

const HeroComponent = () => (
  <section className="bg-white dark:bg-gray-900 px-6 py-20 rounded-b-xl">
    <div className="max-w-7xl mx-auto grid grid-cols-1 md:grid-cols-2 gap-12 items-center">
      <div>
        <h1 className="text-5xl font-bold text-gray-900 dark:text-white">
          Secure <span className="text-blue-primary">Distributed</span> File Storage
        </h1>
        <p className="mt-4 text-lg text-gray-600 dark:text-gray-400">
          Store, share, and access your files with unparalleled security and reliability.
        </p>
        <div className="mt-6 flex gap-4">
          <button className="bg-blue-primary text-white px-6 py-3 rounded-lg font-semibold transition-all duration-300 transform hover:scale-105 shadow-lg">
            Get Started →
          </button>
          <button className="border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-white px-6 py-3 rounded-lg font-semibold transition-all duration-300 transform hover:bg-gray-100 dark:hover:bg-gray-700">
            Learn More
          </button>
        </div>
        <div className="mt-10 flex gap-6">
          <FeatureIcon label="Easy Upload" icon={<ArrowUpIcon size={24} />} />
          <FeatureIcon label="Secure Storage" icon={<LockIcon size={24} />} />
          <FeatureIcon label="Simple Sharing" icon={<ShareIcon size={24} />} />
        </div>
      </div>
      <div className="shadow-lg rounded-lg bg-gray-100 dark:bg-gray-800 h-64 flex items-center justify-center text-gray-400">
        [Image placeholder for hero section]
      </div>
    </div>
  </section>
);

type FeatureIconProps = {
  label: string;
  icon: React.ReactNode;
};

const FeatureIcon = ({ label, icon }: FeatureIconProps) => (
  <div className="flex flex-col items-center text-sm text-gray-700 dark:text-gray-300">
    <div className="bg-blue-light dark:bg-blue-primary p-3 rounded-full mb-2 text-blue-primary dark:text-white">
      {icon}
    </div>
    {label}
  </div>
);

const FeaturesComponent = () => {
  const features = [
    { title: "Reliable Storage", desc: "Files are distributed across multiple nodes for fault tolerance.", icon: <ShieldIcon size={24} /> },
    { title: "End-to-End Encryption", desc: "Files are encrypted client-side before being stored.", icon: <LockIcon size={24} /> },
    { title: "High Performance", desc: "Parallel chunking and upload/download for fast operations.", icon: <ZapIcon size={24} /> },
    { title: "Simple File Upload", desc: "Drag & drop UI with progress tracking for a smooth experience.", icon: <ArrowUpIcon size={24} /> },
    { title: "Secure Downloads", desc: "Chunks are downloaded and reassembled securely with integrity checks.", icon: <ArrowDownIcon size={24} /> },
    { title: "Controlled Sharing", desc: "Share files with custom permissions and expiry dates.", icon: <ShareIcon size={24} /> }
  ];

  return (
    <section className="bg-blue-light dark:bg-gray-800 py-20 px-6">
      <div className="max-w-7xl mx-auto">
        <h2 className="text-3xl font-bold text-center mb-12 text-gray-900 dark:text-white">Powerful Features for Your Files</h2>
        <div className="grid md:grid-cols-3 gap-8">
          {features.map(({ title, desc, icon }) => (
            <div key={title} className="bg-white dark:bg-gray-700 p-8 rounded-xl shadow-md transition-transform duration-300 hover:scale-105">
              <div className="text-3xl mb-4 text-blue-primary dark:text-blue-300">{icon}</div>
              <h3 className="font-semibold text-xl text-gray-900 dark:text-white">{title}</h3>
              <p className="text-gray-900 dark:text-gray-400 mt-2">{desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

const FooterComponent = () => (
  <footer className="bg-white dark:bg-gray-900 border-t border-gray-200 dark:border-gray-700 py-10 px-6">
    <div className="max-w-7xl mx-auto grid grid-cols-2 md:grid-cols-4 gap-8 text-sm text-gray-600 dark:text-gray-400">
      <div>
        <h4 className="font-bold mb-2 text-gray-900 dark:text-white">EchoFS</h4>
        <p>A secure distributed file storage system built for reliability, security, and performance.</p>
      </div>
      <div>
        <h4 className="font-bold mb-2 text-gray-900 dark:text-white">Company</h4>
        <ul className="space-y-1">
          <li>About</li>
          <li>Careers</li>
          <li>Blog</li>
        </ul>
      </div>
      <div>
        <h4 className="font-bold mb-2 text-gray-900 dark:text-white">Help Center</h4>
        <ul className="space-y-1">
          <li>Documentation</li>
          <li>Support</li>
          <li>Contact Us</li>
        </ul>
      </div>
      <div>
        <h4 className="font-bold mb-2 text-gray-900 dark:text-white">Legal</h4>
        <ul className="space-y-1">
          <li>Privacy Policy</li>
          <li>Terms of Service</li>
          <li>Licensing</li>
        </ul>
      </div>
    </div>
    <p className="text-center text-xs text-gray-400 dark:text-gray-500 mt-6">© 2025 EchoFS. All rights reserved.</p>
  </footer>
);

const HighLevelDesignPage = () => (
  <div className="p-8 max-w-7xl mx-auto">
    <h1 className="text-3xl font-bold mb-6 text-gray-900 dark:text-white">High-Level Design</h1>
    <p className="text-lg text-gray-700 dark:text-gray-300 mb-6">
      This diagram illustrates the architecture of the EchoFS system, showing how the frontend interacts with the backend services.
    </p>
    <img
      src="https://www.mermaidchart.com/raw/b4080b9a-bfe7-4016-a3fe-32f437cfe2b2?theme=light&version=v0.1&format=svg"
      alt="EchoFS High-Level Design Diagram"
      className="max-w-full h-auto rounded-lg shadow-lg"
    />
  </div>
);

const FilesPage = () => {
  const [files, setFiles] = useState<any[]>([]);
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

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900 py-12 px-4">
        <div className="max-w-4xl mx-auto text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600 dark:text-gray-400">Loading files...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 py-12 px-4">
      <div className="max-w-4xl mx-auto">
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-8">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-8">My Files</h1>

          {error && (
            <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/20 rounded-lg">
              <p className="text-red-700 dark:text-red-400">{error}</p>
            </div>
          )}

          {files.length === 0 ? (
            <div className="text-center py-12">
              <FileTextIcon className="h-16 w-16 text-gray-400 mx-auto mb-4" />
              <p className="text-gray-600 dark:text-gray-400">No files uploaded yet</p>
            </div>
          ) : (
            <div className="space-y-4">
              {files.map((file: any) => (
                <div key={file.file_id} className="border rounded-lg p-4 flex items-center justify-between">
                  <div>
                    <h3 className="font-semibold text-gray-900 dark:text-white">{file.name}</h3>
                    <p className="text-sm text-gray-500">{formatFileSize(file.size)} • {new Date(file.uploaded).toLocaleString()}</p>
                  </div>
                  <button
                    onClick={() => handleDownload(file.file_id, file.name)}
                    className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700"
                  >
                    Download
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

const UploadDemoPage = () => {
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

      const response = await fetch('http://localhost:8080/api/v1/files/upload', {
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

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 py-12 px-4">
      <div className="max-w-2xl mx-auto">
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-8">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-8 text-center">
            EchoFS Upload Demo
          </h1>

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
              <label htmlFor="file-upload" className="cursor-pointer flex flex-col items-center">
                <ArrowUpIcon className="h-12 w-12 text-gray-400 mb-4" />
                <span className="text-sm text-gray-600 dark:text-gray-400">
                  Click to select a file
                </span>
              </label>
            </div>
          </div>

          {selectedFile && (
            <div className="mb-6 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
              <div className="flex items-center">
                <FileTextIcon className="h-5 w-5 text-blue-500 mr-2" />
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

          <button
            onClick={handleUpload}
            disabled={!selectedFile || uploading}
            className="w-full bg-blue-600 text-white py-3 px-4 rounded-lg font-semibold disabled:opacity-50 disabled:cursor-not-allowed hover:bg-blue-700 transition-colors"
          >
            {uploading ? 'Uploading...' : 'Upload File'}
          </button>

          {error && (
            <div className="mt-4 p-4 bg-red-50 dark:bg-red-900/20 rounded-lg">
              <p className="text-sm text-red-700 dark:text-red-400">{error}</p>
            </div>
          )}

          {uploadResult && (
            <div className="mt-4 p-4 bg-green-50 dark:bg-green-900/20 rounded-lg">
              <p className="text-sm font-medium text-green-700 dark:text-green-400 mb-2">
                Upload Successful!
              </p>
              <div className="text-xs text-gray-600 dark:text-gray-400 space-y-1">
                <p><strong>File ID:</strong> {uploadResult.data?.file_id}</p>
                <p><strong>Chunks:</strong> {uploadResult.data?.chunks}</p>
                <p><strong>Compressed:</strong> {uploadResult.data?.compressed ? 'Yes' : 'No'}</p>
              </div>
            </div>
          )}

          <div className="mt-8 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
            <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
              Backend Status
            </h3>
            <p className="text-xs text-gray-600 dark:text-gray-400">
              Make sure your backend is running: <code>./run_master.sh</code>
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

const FileManagementPage = () => {
  const [fileContent, setFileContent] = useState('');
  const [summary, setSummary] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchSummary = async () => {
    setLoading(true);
    setSummary('');
    setError(null);

    const prompt = `Summarize the following text concisely. The summary should be a few sentences long.
    
    Text:
    """
    ${fileContent}
    """
    `;

    const chatHistory = [];
    chatHistory.push({ role: "user", parts: [{ text: prompt }] });
    const payload = { contents: chatHistory };
    const apiKey = ""
    const apiUrl = `https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-preview-05-20:generateContent?key=${apiKey}`;
    
    let response = null;
    let retries = 0;
    const maxRetries = 5;
    const initialDelay = 1000; 

    while (retries < maxRetries) {
      try {
        response = await fetch(apiUrl, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload)
        });

        if (response.status !== 429) { 
          break;
        }

        retries++;
        const delay = initialDelay * Math.pow(2, retries - 1);
        await new Promise(resolve => setTimeout(resolve, delay));
      } catch (e) {
        throw e;
      }
    }

    if (!response || !response.ok) {
        setError(`API error: ${response ? response.status : 'No'} ${response ? response.statusText : 'response'}`);
        setLoading(false);
        return;
    }

    try {
      const result = await response.json();
      if (result.candidates && result.candidates.length > 0 &&
          result.candidates[0].content && result.candidates[0].content.parts &&
          result.candidates[0].content.parts.length > 0) {
        const text = result.candidates[0].content.parts[0].text;
        setSummary(text);
      } else {
        setSummary("Could not generate a summary. The API response was empty or malformed.");
      }
    } catch (e) {
      console.error(e);
      setError("An error occurred while parsing the API response. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className="bg-white dark:bg-gray-900 py-20 px-6">
      <div className="max-w-7xl mx-auto rounded-lg shadow-md p-8 bg-gray-100 dark:bg-gray-800">
        <h2 className="text-3xl font-bold mb-6 text-gray-900 dark:text-white">
          <FileTextIcon className="inline-block mr-2" size={32} />
          File Manager
        </h2>
        <p className="mb-6 text-gray-700 dark:text-gray-300">
          Paste your document content below and use the power of AI to generate a quick summary.
        </p>

        <textarea
          className="w-full h-64 p-4 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring focus:ring-blue-300 focus:outline-none"
          placeholder="Paste your document content here..."
          value={fileContent}
          onChange={(e) => setFileContent(e.target.value)}
        />
        
        <div className="mt-4 flex justify-center">
          <button
            onClick={fetchSummary}
            disabled={loading || !fileContent}
            className="flex items-center space-x-2 bg-blue-primary text-white px-6 py-3 rounded-lg font-semibold transition-all duration-300 transform hover:scale-105 disabled:opacity-50 disabled:hover:scale-100 shadow-lg"
          >
            {loading ? (
              <svg className="animate-spin h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
            ) : (
              <>
                <SparklesIcon size={20} />
                <span>Summarize Content</span>
              </>
            )}
          </button>
        </div>

        {summary && (
          <div className="mt-8 p-6 bg-white dark:bg-gray-700 rounded-lg shadow-md">
            <h3 className="font-semibold text-lg mb-2 text-gray-900 dark:text-white flex items-center space-x-2">
              <SparklesIcon size={20} />
              <span>Summary</span>
            </h3>
            <p className="text-gray-900 dark:text-gray-300">{summary}</p>
          </div>
        )}

        {error && (
          <div className="mt-8 p-4 bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-200 rounded-lg shadow-md">
            <p>{error}</p>
          </div>
        )}
      </div>
    </section>
  );
};

const MetricsPage = () => {
  const [metrics, setMetrics] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date>(new Date());

  useEffect(() => {
    fetchMetrics();
    const interval = setInterval(fetchMetrics, 5000); // Update every 5 seconds
    return () => clearInterval(interval);
  }, []);

  const fetchMetrics = async () => {
    try {
      const response = await fetch('http://localhost:8080/metrics/dashboard');
      if (!response.ok) {
        throw new Error('Failed to fetch metrics');
      }
      const data = await response.json();
      setMetrics(data);
      setLastUpdated(new Date());
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load metrics');
    } finally {
      setLoading(false);
    }
  };

  const formatNumber = (num: number) => {
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
    return num.toString();
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatTime = (seconds: number) => {
    if (seconds < 1) return (seconds * 1000).toFixed(0) + 'ms';
    return seconds.toFixed(3) + 's';
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900 py-12 px-4">
        <div className="max-w-7xl mx-auto text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600 dark:text-gray-400">Loading metrics...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 py-12 px-4">
      <div className="max-w-7xl mx-auto">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2 flex items-center">
            <BarChart3Icon className="mr-3" size={32} />
            EchoFS Metrics Dashboard
          </h1>
          <p className="text-gray-600 dark:text-gray-400">
            Real-time system performance and usage statistics
          </p>
          <p className="text-sm text-gray-500 dark:text-gray-500 mt-2">
            Last updated: {lastUpdated.toLocaleTimeString()}
          </p>
        </div>

        {error && (
          <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/20 rounded-lg">
            <p className="text-red-700 dark:text-red-400">{error}</p>
            <button 
              onClick={fetchMetrics}
              className="mt-2 text-sm text-red-600 dark:text-red-400 hover:underline"
            >
              Retry
            </button>
          </div>
        )}

        {metrics && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {/* File Operations */}
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">File Operations</h3>
                <FileTextIcon className="text-blue-500" size={24} />
              </div>
              <div className="space-y-3">
                <div className="flex justify-between">
                  <span className="text-gray-600 dark:text-gray-400">Uploads:</span>
                  <span className="font-semibold text-green-600 dark:text-green-400">
                    {formatNumber(metrics.file_operations.total_uploads)}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600 dark:text-gray-400">Downloads:</span>
                  <span className="font-semibold text-blue-600 dark:text-blue-400">
                    {formatNumber(metrics.file_operations.total_downloads)}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600 dark:text-gray-400">Deletes:</span>
                  <span className="font-semibold text-red-600 dark:text-red-400">
                    {formatNumber(metrics.file_operations.total_deletes)}
                  </span>
                </div>
              </div>
            </div>

            {/* Performance */}
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Performance</h3>
                <ClockIcon className="text-purple-500" size={24} />
              </div>
              <div className="space-y-3">
                <div className="flex justify-between">
                  <span className="text-gray-600 dark:text-gray-400">Avg Upload Time:</span>
                  <span className="font-semibold text-purple-600 dark:text-purple-400">
                    {formatTime(metrics.performance.avg_upload_time_seconds)}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600 dark:text-gray-400">Avg Download Time:</span>
                  <span className="font-semibold text-purple-600 dark:text-purple-400">
                    {formatTime(metrics.performance.avg_download_time_seconds)}
                  </span>
                </div>
              </div>
            </div>

            {/* System Status */}
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">System Status</h3>
                <ActivityIcon className="text-green-500" size={24} />
              </div>
              <div className="space-y-3">
                <div className="flex justify-between">
                  <span className="text-gray-600 dark:text-gray-400">Active Connections:</span>
                  <span className="font-semibold text-green-600 dark:text-green-400">
                    {metrics.system.active_connections}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600 dark:text-gray-400">Storage Usage:</span>
                  <span className="font-semibold text-orange-600 dark:text-orange-400">
                    {formatBytes(metrics.system.storage_usage_bytes)}
                  </span>
                </div>
              </div>
            </div>

            {/* gRPC Metrics */}
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">gRPC Communication</h3>
                <TrendingUpIcon className="text-indigo-500" size={24} />
              </div>
              <div className="space-y-3">
                <div className="flex justify-between">
                  <span className="text-gray-600 dark:text-gray-400">Total Requests:</span>
                  <span className="font-semibold text-indigo-600 dark:text-indigo-400">
                    {formatNumber(metrics.grpc.total_requests)}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600 dark:text-gray-400">Errors:</span>
                  <span className="font-semibold text-red-600 dark:text-red-400">
                    {formatNumber(metrics.grpc.total_errors)}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600 dark:text-gray-400">Success Rate:</span>
                  <span className="font-semibold text-green-600 dark:text-green-400">
                    {metrics.grpc.total_requests > 0 
                      ? ((metrics.grpc.total_requests - metrics.grpc.total_errors) / metrics.grpc.total_requests * 100).toFixed(1) + '%'
                      : 'N/A'
                    }
                  </span>
                </div>
              </div>
            </div>

            {/* HTTP Metrics */}
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">HTTP API</h3>
                <TrendingUpIcon className="text-cyan-500" size={24} />
              </div>
              <div className="space-y-3">
                <div className="flex justify-between">
                  <span className="text-gray-600 dark:text-gray-400">Total Requests:</span>
                  <span className="font-semibold text-cyan-600 dark:text-cyan-400">
                    {formatNumber(metrics.http.total_requests)}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600 dark:text-gray-400">Errors:</span>
                  <span className="font-semibold text-red-600 dark:text-red-400">
                    {formatNumber(metrics.http.total_errors)}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600 dark:text-gray-400">Success Rate:</span>
                  <span className="font-semibold text-green-600 dark:text-green-400">
                    {metrics.http.total_requests > 0 
                      ? ((metrics.http.total_requests - metrics.http.total_errors) / metrics.http.total_requests * 100).toFixed(1) + '%'
                      : 'N/A'
                    }
                  </span>
                </div>
              </div>
            </div>

            {/* Quick Actions */}
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Quick Actions</h3>
                <HardDriveIcon className="text-gray-500" size={24} />
              </div>
              <div className="space-y-3">
                <button
                  onClick={fetchMetrics}
                  className="w-full bg-blue-600 text-white py-2 px-4 rounded-lg hover:bg-blue-700 transition-colors"
                >
                  Refresh Metrics
                </button>
                <a
                  href="http://localhost:3001"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="block w-full bg-gray-600 text-white py-2 px-4 rounded-lg hover:bg-gray-700 transition-colors text-center"
                >
                  Open Grafana Dashboard
                </a>
                <a
                  href="http://localhost:9090"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="block w-full bg-orange-600 text-white py-2 px-4 rounded-lg hover:bg-orange-700 transition-colors text-center"
                >
                  Open Prometheus
                </a>
              </div>
            </div>
          </div>
        )}

        {/* Status Indicator */}
        <div className="mt-8 text-center">
          <div className="inline-flex items-center space-x-2 px-4 py-2 bg-green-100 dark:bg-green-900/20 rounded-full">
            <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
            <span className="text-sm text-green-700 dark:text-green-400">
              System Online • Auto-refresh every 5s
            </span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default App;
