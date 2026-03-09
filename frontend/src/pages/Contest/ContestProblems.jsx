import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import api from '../../services/api';
import { CheckCircle, Circle, XCircle } from 'lucide-react';

export default function ContestProblems() {
  const { id } = useParams();
  const [problems, setProblems] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchProblems = async () => {
      try {
        const res = await api.get(`/contests/${id}/problems`);
        setProblems(res.data || []);
      } catch (err) {
        console.error("Failed to fetch contest problems", err);
      } finally {
        setLoading(false);
      }
    };
    fetchProblems();
  }, [id]);

  if (loading) {
    return <div className="text-center text-gray-400 font-mono animate-pulse mt-20">Loading problem set...</div>;
  }

  return (
    <div className="bg-[#1e1e1e] border border-dark-border rounded-lg shadow-2xl overflow-hidden">
      <div className="grid grid-cols-12 gap-4 bg-[#2a2a2a] p-4 border-b border-dark-border text-xs font-bold text-gray-400 uppercase tracking-wider">
        <div className="col-span-1 text-center">Status</div>
        <div className="col-span-1 text-center">#</div>
        <div className="col-span-8">Problem Name</div>
        <div className="col-span-2 text-center">Points</div>
      </div>

      <div className="divide-y divide-dark-border">
        {problems.length === 0 ? (
          <div className="p-8 text-center text-gray-500 italic">The problem set has not been revealed yet.</div>
        ) : (
          problems.map((prob, index) => {
            // Note: In a fully wired backend, you would left join the user's latest submission status here.
            // Defaulting to 'unsolved' for the clean UI state.
            const status = prob.user_status || 'unsolved'; 

            return (
              <Link 
                to={`/contests/${id}/problem/${prob.problem_id}`} 
                key={prob.problem_id}
                className="grid grid-cols-12 gap-4 p-5 items-center transition-colors duration-200 hover:bg-[#252525] group cursor-pointer"
              >
                {/* Status Indicator */}
                <div className="col-span-1 flex justify-center">
                  {status === 'AC' ? <CheckCircle className="text-green-500" size={22} /> :
                   status === 'WA' ? <XCircle className="text-red-500" size={22} /> :
                   <Circle className="text-gray-600 group-hover:text-gray-400 transition-colors" size={22} />}
                </div>
                
                {/* ICPC Style Index (A, B, C...) */}
                <div className="col-span-1 text-center font-mono text-gray-500 font-bold text-lg">
                  {String.fromCharCode(65 + index)} 
                </div>
                
                {/* Title */}
                <div className="col-span-8 font-bold text-blue-400 group-hover:text-blue-300 transition-colors text-xl">
                  {prob.title}
                </div>
                
                {/* Points */}
                <div className="col-span-2 text-center font-mono font-bold text-yellow-500 text-lg">
                  {prob.points_value || 100}
                </div>
              </Link>
            );
          })
        )}
      </div>
    </div>
  );
}