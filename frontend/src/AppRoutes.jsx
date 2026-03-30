import React from 'react';
import { Routes, Route, Navigate, Outlet } from 'react-router-dom';
import { jwtDecode } from 'jwt-decode';
import { useAuth } from './context/AuthContext';

import Navbar from './components/Navbar';

// Page Imports
import Login from './pages/Login';
import Landing from './pages/Landing';
import ProblemList from './pages/ProblemList';
import AddProblem from './pages/AddProblem';
import EditProblem from './pages/EditProblem';
import OAuthSuccess from './pages/OAuthSuccess';
import Onboarding from './pages/Onboarding';
import ContestList from './pages/ContestList';
import EditContest from './pages/EditContest';
import AddContest from './pages/AddContest';
import Leaderboard from './pages/Leaderboard';
import PlaylistList from './pages/PlaylistList';
import PlaylistView from './pages/PlaylistView';
import AddPlaylist from './pages/AddPlaylist';
import EditPlaylist from './pages/EditPlaylist';
import PlaylistAnalytics from './pages/PlaylistAnalytics';

// Phase 4 Imports
import ContestArenaLayout from './pages/Contest/ContestArenaLayout';
import ContestProblems from './pages/Contest/ContestProblems';
import PracticeArena from './pages/Arena/PracticeArena';
import ContestArena from './pages/Arena/ContestArena';

// The Bouncer
const ProtectedRoute = ({ children, requireOnboarding = true }) => {
    const { token, isLoading } = useAuth();

    if (isLoading) {
        return <div className="flex h-screen items-center justify-center bg-dark-bg text-white font-mono">Loading Session...</div>;
    }

    if (!token) return <Navigate to="/login" replace />;

    try {
        const decoded = jwtDecode(token);

        if (!requireOnboarding && decoded.is_onboarded) {
            return <Navigate to="/problems" replace />;
        }

        if (requireOnboarding && !decoded.is_onboarded) {
            return <Navigate to="/onboarding" replace />;
        }
    } catch (error) {
        return <Navigate to="/login" replace />;
    }

    return children;
};

// 👇 THE ROLE BOUNCER 👇
const RequireRole = ({ children, allowedRoles }) => {
    const { token, isLoading } = useAuth();

    if (isLoading) {
        return <div className="flex h-screen items-center justify-center bg-dark-bg text-white font-mono">Verifying Clearance...</div>;
    }

    if (!token) return <Navigate to="/login" replace />;

    try {
        const decoded = jwtDecode(token);
        if (!allowedRoles.includes(decoded.role?.toLowerCase())) {
            return <Navigate to="/problems" replace />;
        }
    } catch (error) {
        return <Navigate to="/login" replace />;
    }

    return children;
};

// 👇 THE NAVBAR WRAPPER 👇
const NavbarLayout = () => {
    return (
        <>
            <Navbar />
            <Outlet />
        </>
    );
};

export default function AppRoutes() {
    return (
        <Routes>
            {/* =========================================
          🟢 ROUTES WITH NAVBAR 🟢
      ========================================= */}
            <Route element={<NavbarLayout />}>
                <Route path="/" element={<Landing />} />
                <Route path="/login" element={<Login />} />
                <Route path="/oauth-success" element={<OAuthSuccess />} />

                <Route path="/onboarding" element={
                    <ProtectedRoute requireOnboarding={false}>
                        <Onboarding />
                    </ProtectedRoute>
                } />

                <Route path="/arena/:id" element={<ProtectedRoute><PracticeArena /></ProtectedRoute>} />
                <Route path="/problems" element={<ProtectedRoute><ProblemList /></ProtectedRoute>} />
                <Route path="/add-problem" element={<ProtectedRoute><AddProblem /></ProtectedRoute>} />
                <Route path="/edit-problem/:id" element={<ProtectedRoute><EditProblem /></ProtectedRoute>} />
                <Route path="/contests" element={<ProtectedRoute><ContestList /></ProtectedRoute>} />
                <Route path="/add-contest" element={<ProtectedRoute><AddContest /></ProtectedRoute>} />
                <Route path="/edit-contest/:id" element={<ProtectedRoute><EditContest /></ProtectedRoute>} />
                
                <Route path="/playlists" element={<ProtectedRoute><PlaylistList /></ProtectedRoute>} />
                <Route path="/playlists/:id" element={<ProtectedRoute><PlaylistView /></ProtectedRoute>} />
                <Route path="/add-playlist" element={
                    <ProtectedRoute>
                        <RequireRole allowedRoles={['admin', 'professor']}>
                            <AddPlaylist />
                        </RequireRole>
                    </ProtectedRoute>
                } />
                <Route path="/playlists/:id/edit" element={
                    <ProtectedRoute>
                        <RequireRole allowedRoles={['admin', 'professor']}>
                            <EditPlaylist />
                        </RequireRole>
                    </ProtectedRoute>
                } />
                <Route path="/playlists/:id/analytics" element={
                    <ProtectedRoute>
                        <RequireRole allowedRoles={['admin', 'professor']}>
                            <PlaylistAnalytics />
                        </RequireRole>
                    </ProtectedRoute>
                } />
            </Route>

            {/* =========================================
          🔴 SECURE CONTEST ROUTES (NO NAVBAR) 🔴
      ========================================= */}
            <Route path="/contests/:id/arena" element={<ProtectedRoute><ContestArenaLayout /></ProtectedRoute>}>
                <Route index element={<ContestProblems />} />
                <Route path="leaderboard" element={<Leaderboard />} />
            </Route>

            <Route
                path="/contests/:id/problem/:problemId"
                element={<ProtectedRoute><ContestArena /></ProtectedRoute>}
            />
        </Routes>
    );
}
