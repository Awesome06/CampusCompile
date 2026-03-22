import axios from 'axios';

const api = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json',
  },
});

api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

api.interceptors.response.use(
  (response) => response,
  (error) => {
    // 1. Handle Network / CORS / Server Down Errors (No Response)
    if (!error.response) {
      error.customMessage = "Network Error: Cannot connect to the Arena servers.";
      return Promise.reject(error);
    }

    const status = error.response.status;
    const data = error.response.data || {};

    // 2. The Guillotine (401 Unauthorized)
    if (status === 401) {
      localStorage.clear();
      window.location.href = '/login';
      return Promise.reject(error);
    }

    // 3. The Rate Limiter (429 Too Many Requests)
    if (status === 429) {
      // Prioritize the specific JSON error from Go, fallback to header
      const retryAfter = data.retry_in || error.response.headers['retry-after'];
      if (retryAfter) {
        error.customMessage = `${data.error || 'Too many requests.'} Wait ${retryAfter}s.`;
      } else {
        error.customMessage = data.error || "Too many requests. Please slow down.";
      }
      return Promise.reject(error);
    }

    // 4. The Circuit Breaker (503 Service Unavailable)
    if (status === 503) {
      error.customMessage = data.error || "System temporarily overloaded. Please retry in a moment.";
      return Promise.reject(error);
    }

    // 5. Standard Error Extraction
    error.customMessage = data.error || "An unexpected error occurred.";
    return Promise.reject(error);
  }
);

export default api;