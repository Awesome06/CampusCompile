# Use a lightweight Node image
FROM node:20-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy the package files and install dependencies
COPY package.json package-lock.json* ./
RUN npm install

# Copy the rest of your React code
COPY . .

# Expose Vite's default port
EXPOSE 5173

# Start the dev server and expose it to the host network
CMD ["npm", "run", "dev", "--", "--host"]