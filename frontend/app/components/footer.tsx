export default function Footer() {
    return (
      <footer className="bg-white border-t py-10 px-6">
        <div className="max-w-7xl mx-auto grid grid-cols-2 md:grid-cols-4 gap-8 text-sm text-gray-600">
          <div>
            <h4 className="font-bold mb-2">DistStore</h4>
            <p>A secure distributed file storage system built for reliability, security, and performance.</p>
          </div>
          <div>
            <h4 className="font-bold mb-2">Company</h4>
            <ul>
              <li>About</li>
              <li>Careers</li>
              <li>Blog</li>
            </ul>
          </div>
          <div>
            <h4 className="font-bold mb-2">Help Center</h4>
            <ul>
              <li>Documentation</li>
              <li>Support</li>
              <li>Contact Us</li>
            </ul>
          </div>
          <div>
            <h4 className="font-bold mb-2">Legal</h4>
            <ul>
              <li>Privacy Policy</li>
              <li>Terms of Service</li>
              <li>Licensing</li>
            </ul>
          </div>
        </div>
        <p className="text-center text-xs text-gray-400 mt-6">© 2025 DistStore. All rights reserved.</p>
      </footer>
    )
  }
  