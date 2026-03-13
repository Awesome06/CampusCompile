import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Trophy, Clock, CheckCircle, AlertTriangle } from 'lucide-react'; 

export default function Leaderboard() {
  const { id: contestId } = useParams();
  const [leaderboard, setLeaderboard] = useState([]);
  const [auditStatus, setAuditStatus] = useState('pending');
  const [loading, setLoading] = useState(true);
  const [connectionError, setConnectionError] = useState(false);

  // 👇 Determine if the user is a Professor or Admin
  const userRole = localStorage.getItem('role');
  const isElevated = userRole === 'admin' || userRole === 'professor';

  useEffect(() => {
    const token = localStorage.getItem('token');
    if (!token) {
      setConnectionError(true);
      setLoading(false);
      return;
    }

    const sseUrl = `http://localhost:8080/api/contests/${contestId}/leaderboard/stream?token=${token}`;
    const source = new EventSource(sseUrl);

    source.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        setLeaderboard(data.leaderboard || []);
        setAuditStatus(data.audit_status || 'pending');
        setLoading(false);
        setConnectionError(false);
      } catch (err) {
        console.error("Failed to parse leaderboard data:", err);
      }
    };

    source.onerror = (err) => {
      console.error("SSE Connection Error. Retrying...", err);
      setConnectionError(true);
    };

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
    <div className="max-w-6xl mx-auto p-6">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-3xl font-bold text-white flex items-center gap-3">
          <Trophy className="text-yellow-500" size={32} />
          Live Standings
        </h2>
        
        <div className="flex items-center gap-2 font-mono text-sm">
          <span className="text-gray-400">STATUS:</span>
          {connectionError ? (
            <span className="text-red-500 font-bold animate-pulse">DISCONNECTED</span>
          ) : auditStatus === 'failed' ? (
            <span className="text-red-500 font-bold flex items-center gap-1 bg-red-900/20 px-2 py-0.5 rounded border border-red-800">
              <AlertTriangle size={14} />
              FAILED TO VERIFY
            </span>
          ) : auditStatus === 'completed' ? (
            <span className="text-blue-400 font-bold flex items-center gap-1 bg-blue-900/20 px-2 py-0.5 rounded border border-blue-800">
              <CheckCircle size={14} />
              VERIFIED
            </span>
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
          <div className="col-span-1 text-center">Rank</div>
          
          {/* 👇 Stretch the participant column to span 7 spaces if we are hiding the integrity column (2 spaces) */}
          <div className={isElevated ? "col-span-5" : "col-span-7"}>Participant</div>
          
          <div className="col-span-2 text-center flex items-center justify-center gap-1">
            <CheckCircle size={14} /> Solves
          </div>
          <div className="col-span-2 text-center flex items-center justify-center gap-1">
            <Clock size={14} /> Penalty
          </div>
          
          {/* 👇 Only render Integrity Header for Faculty/Admin */}
          {isElevated && (
            <div className="col-span-2 text-center flex items-center justify-center gap-1">
              Integrity
            </div>
          )}
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
                <div className="col-span-1 text-center font-mono text-lg font-bold">
                  {player.rank === 1 ? <span className="text-yellow-500">🏆 1</span> :
                   player.rank === 2 ? <span className="text-gray-300">🥈 2</span> :
                   player.rank === 3 ? <span className="text-amber-600">🥉 3</span> :
                   <span className="text-gray-500">{player.rank}</span>}
                </div>

                {/* Username */}
                <div className={`font-semibold text-white text-lg truncate ${isElevated ? 'col-span-5' : 'col-span-7'}`}>
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

                {/* 👇 Integrity / Alerts Column (Faculty Only) */}
                {isElevated && (
                  <div className="col-span-2 text-center flex items-center justify-center relative group">
                    {player.alerts && player.alerts.total > 0 && (
                      <>
                        <AlertTriangle className="text-red-500 cursor-help animate-pulse" size={22} />
                        
                        <div className="absolute bottom-full mb-2 hidden group-hover:block w-56 bg-[#2a2a2a] text-gray-300 text-sm rounded border border-red-800/50 shadow-2xl z-50 p-3 transform -translate-x-1/4">
                          <div className="font-bold text-red-400 border-b border-dark-border mb-2 pb-1 text-left uppercase tracking-wider text-xs">
                            Telemetry Flags ({player.alerts.total})
                          </div>
                          
                          <div className="space-y-1">
                            {player.alerts.blur > 0 && (
                              <div className="flex justify-between">
                                <span>Tab Switches:</span> 
                                <span className="font-mono text-red-400">{player.alerts.blur}</span>
                              </div>
                            )}
                            {player.alerts.paste_attempt > 0 && (
                              <div className="flex justify-between">
                                <span>Paste Attempts:</span> 
                                <span className="font-mono text-red-400">{player.alerts.paste_attempt}</span>
                              </div>
                            )}
                            {player.alerts.autotyper_suspected > 0 && (
                              <div className="flex justify-between">
                                <span>AutoTyper:</span> 
                                <span className="font-mono text-red-400">{player.alerts.autotyper_suspected}</span>
                              </div>
                            )}
                            {player.alerts.visibility_spoof_suspected > 0 && (
                              <div className="flex justify-between">
                                <span>Visibility Spoof:</span> 
                                <span className="font-mono text-red-400">{player.alerts.visibility_spoof_suspected}</span>
                              </div>
                            )}
                          </div>
                          <div className="absolute top-full left-1/2 transform -translate-x-1/2 border-4 border-transparent border-t-[#2a2a2a]"></div>
                        </div>
                      </>
                    )}
                  </div>
                )}

              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}