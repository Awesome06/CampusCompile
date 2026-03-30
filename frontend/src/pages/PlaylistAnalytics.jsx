import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import api from '../services/api';
import Button from '../components/ui/Button';
import { ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip, CartesianGrid, Legend } from 'recharts';

export default function PlaylistAnalytics() {
  const { id } = useParams();
  const [data, setData] = useState([]);
  const [playlist, setPlaylist] = useState(null);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    setLoading(true);
    Promise.all([
      api.get(`/playlists/${id}`),
      api.get(`/playlists/${id}/analytics`)
    ])
      .then(([playlistRes, analyticsRes]) => {
        setPlaylist(playlistRes.data);
        const transformedData = (analyticsRes.data || []).map(item => ({
          name: `Prob ${item.order_index}`,
          completed: item.completed_count,
          total: item.total_students,
          dropoffRate: item.total_students > 0 ? ((item.total_students - item.completed_count) / item.total_students * 100).toFixed(1) : 0
        }));
        setData(transformedData);
        setLoading(false);
      })
      .catch((err) => {
        console.error("Error fetching analytics:", err);
        setLoading(false);
      });
  }, [id]);

  if (loading) return <div className="p-8 text-center font-mono text-gray-400 animate-pulse">Computing Matrix...</div>;
  if (!playlist) return <div className="p-8 text-center text-red-500 font-mono">Failed to load analytics</div>;

  const CustomTooltip = ({ active, payload, label }) => {
    if (active && payload && payload.length) {
      return (
        <div className="bg-[#1e1e1e] p-3 border border-dark-border shadow-lg rounded text-sm text-gray-300 font-mono">
          <p className="font-bold text-white mb-1">{label}</p>
          <p className="text-green-400">Completed By: {payload[0].value} users</p>
          <p className="text-blue-400">Total Enrolled: {payload[1].value} users</p>
          <p className="text-red-400 mt-1 border-t border-dark-border pt-1">
            Drop-off Rate: {(payload[1].value > 0 ? ((payload[1].value - payload[0].value) / payload[1].value * 100).toFixed(1) : 0)}%
          </p>
        </div>
      );
    }
    return null;
  };

  return (
    <div className="p-8 max-w-6xl mx-auto">
      <div className="flex justify-between items-center bg-[#1e1e1e] p-6 rounded-lg border border-dark-border mb-8 shadow-md">
        <div>
          <h1 className="text-2xl font-bold text-white mb-2">{playlist.title} - Analytics</h1>
          <p className="text-sm text-gray-400 font-mono">Track student progression and identify difficulty spikes.</p>
        </div>
        <Button onClick={() => navigate(`/playlists/${id}`)} variant="secondary" className="border-gray-600">
          &larr; Back to Playlist
        </Button>
      </div>

      <div className="bg-[#1e1e1e] p-6 rounded-lg border border-dark-border shadow-md h-[500px]">
        <h2 className="text-lg font-bold text-white mb-6">Completion vs Drop-off Rates</h2>
        {data.length === 0 ? (
          <p className="text-gray-500 italic text-center py-20">Not enough data to compute visualizations.</p>
        ) : (
          <ResponsiveContainer width="100%" height="90%">
            <BarChart data={data} margin={{ top: 20, right: 30, left: 0, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#333" vertical={false} />
              <XAxis dataKey="name" stroke="#888" tick={{ fill: '#888', fontSize: 12 }} />
              <YAxis allowDecimals={false} stroke="#888" tick={{ fill: '#888', fontSize: 12 }} />
              <Tooltip content={<CustomTooltip />} cursor={{ fill: 'rgba(255, 255, 255, 0.05)' }} />
              <Legend wrapperStyle={{ paddingTop: '20px' }} />
              <Bar dataKey="completed" name="Completed (AC)" fill="#4ade80" radius={[4, 4, 0, 0]} barSize={40} />
              <Bar dataKey="total" name="Total Attempts/Enrolled" fill="#3b82f6" radius={[4, 4, 0, 0]} barSize={40} />
            </BarChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  );
}
