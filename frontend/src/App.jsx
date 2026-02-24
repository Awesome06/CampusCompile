import { BrowserRouter, Routes, Route } from 'react-router-dom';

function App() {
  return (
    <BrowserRouter>
      <div className="min-h-screen bg-dark-bg text-dark-text font-sans">
        {/* Simple Navbar */}
        <nav className="p-4 bg-dark-surface border-b border-dark-border flex justify-between items-center">
          <h1 className="text-xl font-bold text-dark-accent">CampusCompile</h1>
          <div className="space-x-4">
            <span className="text-sm">Welcome, Knight</span>
          </div>
        </nav>

        {/* Route Configuration */}
        <Routes>
          <Route path="/" element={
            <div className="p-8 flex flex-col items-center justify-center">
              <h2 className="text-3xl font-bold mb-4">The Arena is Ready</h2>
              <p className="text-gray-400">Tailwind CSS and React Router are successfully wired up.</p>
            </div>
          } />
        </Routes>
      </div>
    </BrowserRouter>
  );
}

export default App;