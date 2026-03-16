import React, { createContext, useState, useEffect, useContext } from 'react';
import { jwtDecode } from 'jwt-decode';

const AuthContext = createContext();

export const AuthProvider = ({ children }) => {
    const [authState, setAuthState] = useState({
        token: localStorage.getItem('token') || null,
        currentUser: null,
        isLoggedIn: false,
        isLoading: true
    });

    useEffect(() => {
        const token = localStorage.getItem('token');
        if (token) {
            try {
                const decoded = jwtDecode(token);
                setAuthState({
                    token,
                    currentUser: {
                        id: decoded.user_id || decoded.sub || decoded.id,
                        role: decoded.role?.toLowerCase(),
                        isOnboarded: !!decoded.is_onboarded
                    },
                    isLoggedIn: true,
                    isLoading: false
                });
            } catch (err) {
                console.error("Failed to decode token:", err);
                localStorage.removeItem('token');
                setAuthState(prev => ({ ...prev, isLoading: false }));
            }
        } else {
            setAuthState(prev => ({ ...prev, isLoading: false }));
        }
    }, []);

    const login = (token) => {
        localStorage.setItem('token', token);
        const decoded = jwtDecode(token);
        setAuthState({
            token,
            currentUser: {
                id: decoded.user_id || decoded.sub || decoded.id,
                role: decoded.role?.toLowerCase()
            },
            isLoggedIn: true,
            isLoading: false
        });
    };

    const logout = () => {
        localStorage.clear();
        setAuthState({ token: null, currentUser: null, isLoggedIn: false, isLoading: false });
        window.location.href = '/'; 
    };

    return (
        <AuthContext.Provider value={{ ...authState, login, logout }}>
            {!authState.isLoading && children}
        </AuthContext.Provider>
    );
};

export const useAuth = () => useContext(AuthContext);