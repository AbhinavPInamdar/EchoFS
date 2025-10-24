const features = [
    { title: "Reliable Storage", desc: "Files are distributed...", icon: "🛡️" },
    { title: "End-to-End Encryption", desc: "Encrypted before storage...", icon: "🔐" },
    { title: "High Performance", desc: "Parallel processing & caching", icon: "⚡" },
    { title: "Simple File Upload", desc: "Drag & drop UI with tracking", icon: "⬆️" },
    { title: "Secure Downloads", desc: "Malware scanning built-in", icon: "⬇️" },
    { title: "Controlled Sharing", desc: "Custom permissions & expiry", icon: "📤" }
  ]
  
  export default function Features() {
    return (
      <section className="bg-blue-50 py-20 px-6">
        <div className="max-w-7xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-8">Powerful Features for Your Files</h2>
          <div className="grid md:grid-cols-3 gap-8">
            {features.map(({ title, desc, icon }) => (
              <div key={title} className="bg-white p-6 rounded-xl shadow-md">
                <div className="text-3xl mb-4">{icon}</div>
                <h3 className="font-semibold text-xl">{title}</h3>
                <p className="text-gray-600 mt-2">{desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>
    )
  }