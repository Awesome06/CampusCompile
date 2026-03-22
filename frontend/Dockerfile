# --- Stage 1: Build the React App ---
FROM node:20-alpine AS builder

WORKDIR /app

# Accept the API URL from docker-compose.prod.yml
ARG VITE_API_URL
ENV VITE_API_URL=$VITE_API_URL

# Copy ONLY package files first. 
# Docker caches this layer. If you only change your React components, 
# Docker skips the slow 'npm install' step entirely.
COPY package.json package-lock.json* ./

# npm ci is stricter than npm install. It installs exact versions from package-lock.json,
# preventing unexpected bugs in production. 
RUN npm ci --legacy-peer-deps

# Copy the rest of the application code
COPY . .

# Compile the React code into static files
RUN npm run build

# --- Stage 2: Serve with Nginx ---
FROM nginx:alpine

# Remove default Nginx static assets to keep the image clean
RUN rm -rf /usr/share/nginx/html/*

# Copy the built static files from Stage 1 into Nginx
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy our custom Nginx routing config
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Expose standard HTTP port 80 natively
EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]