# --- Stage 1: Build the React App ---
FROM node:20-alpine AS builder

WORKDIR /app

# Accept the API URL from docker-compose.yml
ARG VITE_API_URL
ENV VITE_API_URL=$VITE_API_URL

COPY package.json package-lock.json* ./
RUN npm install --legacy-peer-deps

COPY . .
# This compiles the React code into highly optimized static files
RUN npm run build

# --- Stage 2: Serve with Nginx ---
FROM nginx:alpine

# Copy the built static files from Stage 1 into Nginx
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy our custom Nginx routing config
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Expose standard HTTP port 80 natively
EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]