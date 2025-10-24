import { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Architecture - EchoFS',
  description: 'High-level architecture diagram of the EchoFS adaptive consistency system',
};

export default function Page() {
    return (
      <div className="p-4">
        <h1 className="text-xl font-bold mb-4 text-black">Mermaid Diagram</h1>
        <img 
          src="https://www.mermaidchart.com/raw/b4080b9a-bfe7-4016-a3fe-32f437cfe2b2?theme=light&version=v0.1&format=svg" 
          alt="EchoFS Diagram" 
          className="max-w-full h-auto"
        />
      </div>
    );
  }