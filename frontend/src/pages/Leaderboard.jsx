import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Trophy, Clock, CheckCircle } from 'lucide-react'; // Assuming you use lucide-react for icons

export default function Leaderboard() {
  const { id: contestId } = useParams();
  const [leaderboard, setLeaderboard] = useState([]);
  const [loading, setLoading] = useState(true);
  const [connectionError, setConnectionError] = useState(false);

  useEffect(() => {
    // Grab the JWT from storage to authenticate the SSE connection
    const token = localStorage.getItem('token');
    if (!token) {
      setConnectionError(true);
      setLoading(false);
      return;
    }

    // Connect to the Go SSE Ticker endpoint
    const sseUrl = `http://localhost:8080/api/contests/${contestId}/leaderboard/stream?token=${token}`;
    const source = new EventSource(sseUrl);

    source.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        setLeaderboard(data);
        setLoading(false);
        setConnectionError(false);
      } catch (err) {
        console.error("Failed to parse leaderboard data:", err);
      }
    };

    source.onerror = (err) => {
      console.error("SSE Connection Error. Retrying...", err);
      // EventSource automatically attempts to reconnect, but we can flag the UI
      setConnectionError(true);
    };

    // Clean up the connection when the user leaves the arena page
    return () => {
      source.close();
    };
  }, [contestId]);

  if (loading) {
    return (
      <div className="flex justify-center items-center h-64 text-gray-400 animate-pulse font-mono">
        Establishing live telemetry with the Arena...
      </div>
    );
  }

  return (
    <div className="max-w-5xl mx-auto p-6">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-3xl font-bold text-white flex items-center gap-3">
          <Trophy className="text-yellow-500" size={32} />
          Live Standings
        </h2>
        
        {/* Connection Status Indicator */}
        <div className="flex items-center gap-2 font-mono text-sm">
          <span className="text-gray-400">STATUS:</span>
          {connectionError ? (
            <span className="text-red-500 font-bold animate-pulse">DISCONNECTED</span>
          ) : (
            <span className="text-green-500 font-bold flex items-center gap-1">
              <span className="h-2 w-2 bg-green-500 rounded-full animate-ping mr-1"></span>
              SYNCED
            </span>
          )}
        </div>
      </div>

      <div className="bg-[#1e1e1e] border border-dark-border rounded-lg shadow-2xl overflow-hidden">
        {/* Table Header */}
        <div className="grid grid-cols-12 gap-4 bg-[#2a2a2a] p-4 border-b border-dark-border text-xs font-bold text-gray-400 uppercase tracking-wider">
          <div className="col-span-2 text-center">Rank</div>
          <div className="col-span-6">Participant</div>
          <div className="col-span-2 text-center flex items-center justify-center gap-1">
            <CheckCircle size={14} /> Solves
          </div>
          <div className="col-span-2 text-center flex items-center justify-center gap-1">
            <Clock size={14} /> Penalty
          </div>
        </div>

        {/* Table Body */}
        <div className="divide-y divide-dark-border">
          {leaderboard.length === 0 ? (
            <div className="p-8 text-center text-gray-500 italic">
              No participants have solved a problem yet. The arena is quiet.
            </div>
          ) : (
            leaderboard.map((player) => (
              <div 
                key={player.user_id} 
                className={`grid grid-cols-12 gap-4 p-4 items-center transition-colors duration-300 hover:bg-[#252525] ${
                  player.rank === 1 ? 'bg-yellow-900/10' : 
                  player.rank === 2 ? 'bg-gray-400/10' : 
                  player.rank === 3 ? 'bg-amber-700/10' : ''
                }`}
              >
                {/* Rank */}
                <div className="col-span-2 text-center font-mono text-lg font-bold">
                  {player.rank === 1 ? <span className="text-yellow-500">🏆 1</span> :
                   player.rank === 2 ? <span className="text-gray-300">🥈 2</span> :
                   player.rank === 3 ? <span className="text-amber-600">🥉 3</span> :
                   <span className="text-gray-500">{player.rank}</span>}
                </div>

                {/* Username */}
                <div className="col-span-6 font-semibold text-white text-lg truncate">
                  {player.username}
                </div>

                {/* Solves */}
                <div className="col-span-2 text-center font-mono font-bold text-green-400 text-xl">
                  {player.solves}
                </div>

                {/* Penalty */}
                <div className="col-span-2 text-center font-mono text-gray-400">
                  {player.penalty} <span className="text-xs text-gray-600">min</span>
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}