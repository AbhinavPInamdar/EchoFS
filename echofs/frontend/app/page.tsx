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
  Sparkles as SparklesIcon
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
          </div>
        </nav>
      </header>

      <main>
        {page === 'home' && <HomePage />}
        {page === 'hld' && <HighLevelDesignPage />}
        {page === 'file-manager' && <FileManagementPage />}
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

export default App;
