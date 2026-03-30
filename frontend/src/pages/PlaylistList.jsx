import React, { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import api from '../services/api';
import Button from '../components/ui/Button';
import { useAuth } from '../context/AuthContext';

export default function PlaylistList() {
  const [playlists, setPlaylists] = useState([]);
  const [loading, setLoading] = useState(true);
  const { currentUser } = useAuth();
  const navigate = useNavigate();

  const isElevated = currentUser?.role === 'admin' || currentUser?.role === 'professor';

  // "public" or "faculty"
  const [viewMode, setViewMode] = useState(isElevated ? 'faculty' : 'public');
  const [offset, setOffset] = useState(0);
  const limit = 10;
  const [hasMore, setHasMore] = useState(true);

  const fetchPlaylists = (currentOffset, mode) => {
    if (currentOffset === 0) setLoading(true);
    api.get('/playlists', { params: { limit, offset: currentOffset, view_mode: mode } })
      .then((response) => {
        const data = response.data || [];
        setPlaylists(data);
        setHasMore(data.length === limit);
        setOffset(currentOffset);
        setLoading(false);
      })
      .catch((error) => {
        console.error("Error fetching playlists:", error);
        setLoading(false);
      });
  };

  useEffect(() => {
    setOffset(0);
    setHasMore(true);
    fetchPlaylists(0, viewMode);
  }, [viewMode]);

  const handlePrev = () => {
    if (offset >= limit) {
      const nextOffset = offset - limit;
      fetchPlaylists(nextOffset, viewMode);
    }
  };

  const handleNext = () => {
    if (hasMore) {
      const nextOffset = offset + limit;
      fetchPlaylists(nextOffset, viewMode);
    }
  };

  const getDifficultyColor = (diff) => {
    if (diff === 'Easy') return 'text-green-400 bg-green-900/20 border-green-700/50';
    if (diff === 'Medium') return 'text-yellow-400 bg-yellow-900/20 border-yellow-700/50';
    if (diff === 'Hard') return 'text-red-400 bg-red-900/20 border-red-700/50';
    return 'text-gray-400 bg-gray-900/20 border-gray-700/50';
  };

  return (
    <div className="p-8 max-w-6xl mx-auto">
      {/* Header & Dynamic Tabs */}
      <div className="relative flex justify-center items-center mb-8 h-10 w-full">
        <div className="flex bg-[#1e1e1e] rounded-lg p-1 border border-dark-border shadow-lg">
          {isElevated && (
            <>
              <button onClick={() => setViewMode('faculty')} className={`px-6 py-2 text-sm font-bold rounded-md transition-all ${viewMode === 'faculty' ? 'bg-dark-accent text-white shadow' : 'text-gray-500 hover:text-gray-300'}`}>
                My Playlists
              </button>
              <button onClick={() => setViewMode('public')} className={`px-6 py-2 text-sm font-bold rounded-md transition-all ${viewMode === 'public' ? 'bg-dark-accent text-white shadow' : 'text-gray-500 hover:text-gray-300'}`}>
                Community Playlists
              </button>
            </>
          )}
          {!isElevated && (
            <button className="px-6 py-2 text-sm font-bold rounded-md transition-all bg-dark-accent text-white shadow cursor-default">
              Community Playlists
            </button>
          )}
        </div>
        
        {/* Right-Pinned Create Button */}
        {isElevated && (
          <div className="absolute right-0">
            <Button onClick={() => navigate('/add-playlist')} variant="success" size="sm" className="border border-green-600 hover:border-green-500">
              <span>+</span> Create Playlist
            </Button>
          </div>
        )}
      </div>

      {/* Playlist List Render */}
      <div className="bg-[#1e1e1e] border border-dark-border rounded-lg p-4 shadow-lg">
        {loading && <div className="text-center py-8 text-gray-400 animate-pulse font-mono">Fetching collections...</div>}
        
        {!loading && playlists.length === 0 && (
          <div className="text-center py-10 text-gray-500 italic border-b border-dark-border last:border-0">
            No playlists found.
          </div>
        )}
        
        {!loading && playlists.map((playlist) => (
          <Link key={playlist.playlist_id} to={`/playlists/${playlist.playlist_id}`} className="block">
            <div className="flex justify-between items-center py-5 text-white border-b border-dark-border last:border-0 hover:bg-[#252525] px-4 rounded transition group cursor-pointer relative">
              
              <div className="w-2/3 flex flex-col items-start gap-2">
                <div className="flex items-center gap-3">
                  <span className="font-bold text-xl group-hover:text-blue-400 transition">{playlist.title}</span>
                  <span className={`text-[10px] px-2 py-0.5 rounded font-bold tracking-wider border ${getDifficultyColor(playlist.overall_difficulty)}`}>
                    {playlist.overall_difficulty.toUpperCase()}
                  </span>
                  {!playlist.is_public && (
                    <span className="bg-yellow-900/20 text-yellow-500 border border-yellow-700/50 text-[10px] px-2 py-0.5 rounded font-bold tracking-wider">
                      DRAFT
                    </span>
                  )}
                </div>
                <div className="text-sm text-gray-400">
                  {playlist.description || 'No description provided.'}
                </div>
                <div className="text-xs text-gray-500 font-mono mt-1 flex gap-4 items-center">
                  <span>👤 By {playlist.author_name}</span>
                  {playlist.tags && playlist.tags.length > 0 && (
                    <div className="flex gap-1">
                      {playlist.tags.map(tag => (
                        <span key={tag} className="bg-dark-surface px-1.5 py-0.5 rounded border border-dark-border">
                          {tag}
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              </div>

              <div className="w-1/3 text-right flex flex-col items-end gap-2">
                <div className="flex flex-col items-end">
                  <span className="text-sm text-gray-400 font-semibold mb-1">Completion Progress</span>
                  <div className="flex items-center gap-3 w-full justify-end">
                    <span className="text-green-400 font-bold font-mono bg-green-900/20 px-2 py-1 border border-green-800/50 rounded shadow-sm">
                      {playlist.solved_count} / {playlist.total_count} Solved
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </Link>
        ))}

        {!loading && (hasMore || offset > 0) && (
          <div className="flex justify-between items-center mt-6 pt-4 border-t border-dark-border px-4">
            <Button onClick={handlePrev} variant="outline" disabled={offset === 0} className="w-1/5 text-gray-300 border-dark-border hover:bg-[#2a2a2a] disabled:opacity-30 disabled:cursor-not-allowed">
              &larr; Prev
            </Button>
            <span className="w-3/5 text-center text-gray-400 text-sm font-mono">
              Showing {offset + 1} - {offset + playlists.length}
            </span>
            <Button onClick={handleNext} variant="outline" disabled={!hasMore} className="w-1/5 text-gray-300 border-dark-border hover:bg-[#2a2a2a] disabled:opacity-30 disabled:cursor-not-allowed">
              Next &rarr;
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
