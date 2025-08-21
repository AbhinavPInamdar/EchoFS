export default function Hero() {
    return (
      <section className="bg-white px-6 py-20">
        <div className="max-w-7xl mx-auto grid grid-cols-1 md:grid-cols-2 gap-12 items-center">
          <div>
            <h1 className="text-5xl font-bold text-gray-900">
              Secure <span className="text-blue-600">Distributed</span> File Storage
            </h1>
            <p className="mt-4 text-lg text-gray-600">
              Store, share, and access your files with unparalleled security and reliability...
            </p>
            <div className="mt-6 flex gap-4">
              <button className="bg-blue-600 text-white px-6 py-2 rounded-lg">Get Started →</button>
              <button className="border px-6 py-2 rounded-lg">Learn More</button>
            </div>
            <div className="mt-10 flex gap-6">
              {/* Icons with labels */}
              <FeatureIcon label="Easy Upload" />
              <FeatureIcon label="Secure Storage" />
              <FeatureIcon label="Simple Sharing" />
            </div>
          </div>
          <div className="shadow-lg rounded-lg bg-gray-100 h-64"></div>
        </div>
      </section>
    )
  }
  
  function FeatureIcon({ label }: { label: string }) {
    return (
      <div className="flex flex-col items-center text-sm text-gray-700">
        <div className="bg-blue-100 p-3 rounded-full mb-2">🔒</div>
        {label}
      </div>
    )
  }
  