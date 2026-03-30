import React, { useState, useEffect } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import api from '../services/api';
import Button from '../components/ui/Button';
import { useAuth } from '../context/AuthContext';

export default function PlaylistView() {
  const { id } = useParams();
  const [playlist, setPlaylist] = useState(null);
  const [problems, setProblems] = useState([]);
  const [loading, setLoading] = useState(true);
  const { currentUser } = useAuth();
  const navigate = useNavigate();

  const isElevated = currentUser?.role === 'admin' || currentUser?.role === 'professor';
  const canEdit = currentUser?.role === 'admin' || (playlist && playlist.author_id === currentUser?.id);

  useEffect(() => {
    setLoading(true);
    Promise.all([
      api.get(`/playlists/${id}`),
      api.get(`/playlists/${id}/problems`)
    ])
      .then(([playlistRes, problemsRes]) => {
        setPlaylist(playlistRes.data);
        setProblems(problemsRes.data || []);
        setLoading(false);
      })
      .catch((err) => {
        console.error("Error loading playlist:", err);
        setLoading(false);
      });
  }, [id]);

  if (loading) {
    return <div className="p-8 text-center font-mono text-gray-400 animate-pulse">Loading playlist...</div>;
  }

  if (!playlist) {
    return <div className="p-8 text-center font-mono text-red-500">Playlist not found</div>;
  }

  const getStatusColor = (status) => {
    if (status === 'Accepted') return 'bg-green-900/20 border-green-700/50 text-green-400';
    if (status === 'Attempted/WA') return 'bg-yellow-900/20 border-yellow-700/50 text-yellow-500';
    return 'bg-gray-800 border-gray-600 text-gray-400';
  };

  return (
    <div className="p-8 max-w-6xl mx-auto">
      {/* Header */}
      <div className="bg-[#1e1e1e] p-6 rounded-lg border border-dark-border shadow-md mb-8 relative">
        <h1 className="text-3xl font-bold text-white mb-2">{playlist.title}</h1>
        <p className="text-gray-400 mb-4">{playlist.description}</p>
        <div className="flex items-center gap-4 text-sm font-mono text-gray-500">
          <span>By {playlist.author_name}</span>
          <span>•</span>
          <span className="bg-dark-surface px-2 py-1 rounded border border-dark-border">
            Difficulty: <span className="text-yellow-400">{playlist.overall_difficulty}</span>
          </span>
          {playlist.tags && playlist.tags.length > 0 && (
            <>
              <span>•</span>
              <div className="flex gap-2">
                {playlist.tags.map(t => (
                  <span key={t} className="bg-[#2a2a2a] px-2 py-1 rounded text-gray-300">
                    {t}
                  </span>
                ))}
              </div>
            </>
          )}
        </div>

        {/* Action Buttons for Faculty */}
        {canEdit && (
          <div className="absolute top-6 right-6 flex items-center gap-2">
            <Button onClick={() => navigate(`/playlists/${id}/edit`)} variant="secondary" className="border border-dark-border hover:bg-gray-700">
              ⚙️ Edit
            </Button>
            <Button onClick={() => navigate(`/playlists/${id}/analytics`)} variant="primary" className="border border-blue-600">
              📊 View Analytics
            </Button>
          </div>
        )}
      </div>

      {/* Problem Progression */}
      <div className="bg-[#1e1e1e] p-6 rounded-lg border border-dark-border shadow-md">
        <h2 className="text-xl font-bold text-white mb-4 border-b border-dark-border pb-2">Problem Sequence</h2>
        
        {problems.length === 0 ? (
          <p className="text-gray-500 italic">No problems have been added to this playlist yet.</p>
        ) : (
          <div className="space-y-4">
            {problems.map((prob) => {
              let rowColor = 'hover:bg-[#2a2a2a] bg-[#222]';
              if (prob.status === 'Accepted') rowColor = 'bg-green-900/10 border-green-800/50 hover:bg-green-900/20';
              else if (prob.status === 'Attempted/WA') rowColor = 'bg-yellow-900/10 border-yellow-800/50 hover:bg-yellow-900/20';

              return (
                <div key={prob.problem_id} className={`flex justify-between items-center py-4 text-white border border-dark-border px-4 rounded transition relative ${rowColor}`}>
                  <div className="w-12 text-left font-bold text-gray-500">{prob.order_index}</div>

                  <div className="w-1/2 font-medium flex items-center gap-3">
                    <Link to={`/arena/${prob.problem_id}`} className="hover:text-blue-400 transition text-lg">
                      {prob.title}
                    </Link>
                    {/* Custom Difficulty Override if any */}
                    {prob.custom_difficulty && (
                      <span className="px-2 py-0.5 rounded text-[10px] font-bold tracking-wider border text-purple-400 bg-purple-900/20 border-purple-700/50">
                        {prob.custom_difficulty}
                      </span>
                    )}
                  </div>

                  <div className="w-1/4 text-center">
                    <span className="text-sm font-mono text-gray-400">Standard: {prob.difficulty}</span>
                  </div>

                  <div className="w-1/4 text-right">
                    <span className={`px-3 py-1 text-xs font-bold rounded border ${getStatusColor(prob.status)} uppercase tracking-wide shadow-sm`}>
                      {prob.status}
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
